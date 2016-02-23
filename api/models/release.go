package models

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/structs"
)

func NewRelease(app string) structs.Release {
	return structs.Release{
		Id:  generateId("R", 10),
		App: app,
	}
}

// func (r *Release) Cleanup() error {
//   app, err := provider.AppGet(r.App)

//   if err != nil {
//     return err
//   }

//   // delete env
//   err = s3Delete(app.Outputs["Settings"], fmt.Sprintf("releases/%s/env", r.Id))

//   if err != nil {
//     return err
//   }

//   return nil
// }

func ReleasePromote(r *structs.Release) error {
	app, err := provider.AppGet(r.App)

	if err != nil {
		return err
	}

	formation, err := releaseFormation(r)

	if err != nil {
		return err
	}

	// If release formation was saved in S3, get that instead
	f, err := s3Get(app.Outputs["Settings"], fmt.Sprintf("templates/%s", r.Id))

	if err != nil && awserrCode(err) != "NoSuchKey" {
		return err
	}

	if err == nil {
		formation = string(f)
	}

	fmt.Printf("ns=kernel at=release.promote at=s3Get found=%t\n", err == nil)

	existing, err := formationParameters(formation)

	if err != nil {
		return err
	}

	app.Parameters["Environment"] = releaseEnvironmentUrl(r)
	app.Parameters["Kernel"] = CustomTopic
	app.Parameters["Release"] = r.Id
	app.Parameters["Version"] = os.Getenv("RELEASE")

	if os.Getenv("ENCRYPTION_KEY") != "" {
		app.Parameters["Key"] = os.Getenv("ENCRYPTION_KEY")
	}

	// SubnetsPrivate is a List<AWS::EC2::Subnet::Id> and can not be empty
	// So reuse SUBNETS if SUBNETS_PRIVATE is not set
	subnetsPrivate := os.Getenv("SUBNETS_PRIVATE")
	if subnetsPrivate == "" {
		subnetsPrivate = os.Getenv("SUBNETS")
	}

	app.Parameters["SubnetsPrivate"] = subnetsPrivate

	params := []*cloudformation.Parameter{}

	for key, value := range app.Parameters {
		if _, ok := existing[key]; ok {
			params = append(params, &cloudformation.Parameter{ParameterKey: aws.String(key), ParameterValue: aws.String(value)})
		}
	}

	err = S3Put(app.Outputs["Settings"], fmt.Sprintf("templates/%s", r.Id), []byte(formation), false)

	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://s3.amazonaws.com/%s/templates/%s", app.Outputs["Settings"], r.Id)

	req := &cloudformation.UpdateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(r.App),
		TemplateURL:  aws.String(url),
		Parameters:   params,
	}

	_, err = CloudFormation().UpdateStack(req)

	provider.NotifySuccess("release:promote", map[string]string{
		"app": r.App,
		"id":  r.Id,
	})

	return err
}

func releaseEnvironmentUrl(r *structs.Release) string {
	app, err := provider.AppGet(r.App)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return ""
	}

	return fmt.Sprintf("https://%s.s3.amazonaws.com/releases/%s/env", app.Outputs["Settings"], r.Id)
}

func releaseFormation(r *structs.Release) (string, error) {
	manifest, err := LoadManifest(r.Manifest)

	if err != nil {
		return "", err
	}

	// try to figure out which process to map to the main load balancer
	primary, err := primaryProcess(r.App)

	if err != nil {
		return "", err
	}

	// if we dont have a primary default to a process named web
	if primary == "" && manifest.Entry("web") != nil {
		primary = "web"
	}

	// if we still dont have a primary try the first process with external ports
	if primary == "" && manifest.HasExternalPorts() {
		for _, entry := range manifest {
			if len(entry.ExternalPorts()) > 0 {
				primary = entry.Name
				break
			}
		}
	}

	app, err := provider.AppGet(r.App)

	if err != nil {
		return "", err
	}

	for i, entry := range manifest {
		if entry.Name == primary {
			manifest[i].primary = true
		}
	}

	// set the image
	for i, entry := range manifest {
		var imageName string
		if registryId := app.Outputs["RegistryId"]; registryId != "" {
			imageName = fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s.%s", registryId, os.Getenv("AWS_REGION"), app.Outputs["RegistryRepository"], entry.Name, r.Build)
		} else {
			imageName = fmt.Sprintf("%s/%s-%s:%s", os.Getenv("REGISTRY_HOST"), r.App, entry.Name, r.Build)
		}
		manifest[i].Image = imageName
	}

	manifest, err = resolveLinks(*app, &manifest)

	if err != nil {
		return "", err
	}

	return manifest.Formation()
}

func resolveLinks(app structs.App, manifest *Manifest) (Manifest, error) {
	m := *manifest

	endpoint, err := AppDockerLogin(app)

	if err != nil {
		return m, fmt.Errorf("could not log into %q", endpoint)
	}

	for i, entry := range m {
		var inspect []struct {
			Config struct {
				Env []string
			}
		}

		imageName := entry.Image

		cmd := exec.Command("docker", "pull", imageName)
		out, err := cmd.CombinedOutput()
		fmt.Printf("ns=kernel at=release.formation at=entry.pull imageName=%q out=%q err=%q\n", imageName, string(out), err)

		if err != nil {
			return m, fmt.Errorf("could not pull %q", imageName)
		}

		cmd = exec.Command("docker", "inspect", imageName)
		out, err = cmd.CombinedOutput()
		fmt.Printf("ns=kernel at=release.formation at=entry.inspect imageName=%q out=%q err=%q\n", imageName, string(out), err)

		if err != nil {
			return m, fmt.Errorf("could not inspect %q", imageName)
		}

		err = json.Unmarshal(out, &inspect)

		if err != nil {
			fmt.Printf("ns=kernel at=release.formation at=entry.unmarshal err=%q\n", err)
			return m, fmt.Errorf("could not inspect %q", imageName)
		}

		entry.Exports = make(map[string]string)
		linkableEnvs := entry.Env
		if len(inspect) == 1 {
			//manifest entry gets priority for auto-link
			linkableEnvs = append(inspect[0].Config.Env, entry.Env...)
		}

		for _, val := range linkableEnvs {
			if strings.HasPrefix(val, "LINK_") {
				parts := strings.SplitN(val, "=", 2)
				if len(parts) == 2 {
					entry.Exports[parts[0]] = parts[1]
					m[i] = entry
				}
			}
		}
	}

	for i, entry := range m {
		entry.LinkVars = make(map[string]template.HTML)
		for _, link := range entry.Links {
			other := m.Entry(link)

			if other == nil {
				return m, fmt.Errorf("Cannot find link %q", link)
			}

			scheme := other.Exports["LINK_SCHEME"]
			if scheme == "" {
				scheme = "tcp"
			}

			mb := manifest.GetBalancer(link)
			if mb == nil {
				// commented out to be less strict, just don't create the link
				//return m, fmt.Errorf("Cannot discover balancer for link %q", link)
				continue
			}
			host := fmt.Sprintf(`{ "Fn::GetAtt" : [ "%s", "DNSName" ] }`, mb.ResourceName())

			if len(other.Ports) == 0 {
				// commented out to be less strict, just don't create the link
				// return m, fmt.Errorf("Cannot link to %q because it does not expose ports in the manifest", link)
				continue
			}

			port := other.Ports[0]
			port = strings.Split(port, ":")[0]

			path := other.Exports["LINK_PATH"]

			var userInfo string
			if other.Exports["LINK_USERNAME"] != "" || other.Exports["LINK_PASSWORD"] != "" {
				userInfo = fmt.Sprintf("%s:%s@", other.Exports["LINK_USERNAME"], other.Exports["LINK_PASSWORD"])
			}

			html := fmt.Sprintf(`{ "Fn::Join": [ "", [ "%s", "://", "%s", %s, ":", "%s", "%s" ] ] }`,
				scheme, userInfo, host, port, path)

			prefix := strings.ToUpper(link) + "_"

			entry.LinkVars[prefix+"HOST"] = template.HTML(host)
			entry.LinkVars[prefix+"SCHEME"] = template.HTML(fmt.Sprintf("%q", scheme))
			entry.LinkVars[prefix+"PORT"] = template.HTML(fmt.Sprintf("%q", port))
			entry.LinkVars[prefix+"PASSWORD"] = template.HTML(fmt.Sprintf("%q", other.Exports["LINK_PASSWORD"]))
			entry.LinkVars[prefix+"USERNAME"] = template.HTML(fmt.Sprintf("%q", other.Exports["LINK_USERNAME"]))
			entry.LinkVars[prefix+"PATH"] = template.HTML(fmt.Sprintf("%q", path))
			entry.LinkVars[prefix+"URL"] = template.HTML(html)
			m[i] = entry
		}
	}

	return m, nil
}

var regexpPrimaryProcess = regexp.MustCompile(`\[":",\["TCP",\{"Ref":"([A-Za-z]+)Port\d+Host`)

// try to determine which process to map to the main load balancer
func primaryProcess(app string) (string, error) {
	res, err := CloudFormation().GetTemplate(&cloudformation.GetTemplateInput{
		StackName: aws.String(app),
	})

	if err != nil {
		return "", err
	}

	/* bounce through json marshaling to make whitespace predictable */

	var body interface{}

	err = json.Unmarshal([]byte(*res.TemplateBody), &body)

	if err != nil {
		return "", err
	}

	data, err := json.Marshal(body)

	process := regexpPrimaryProcess.FindStringSubmatch(string(data))

	if len(process) > 1 {
		return DashName(process[1]), nil
	}

	return "", nil
}
