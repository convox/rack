package models

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/convox/rack/api/crypt"
)

type Release struct {
	Id       string    `json:"id"`
	App      string    `json:"app"`
	Build    string    `json:"build"`
	Env      string    `json:"env"`
	Manifest string    `json:"manifest"`
	Created  time.Time `json:"created"`
}

type Releases []Release

func NewRelease(app string) Release {
	return Release{
		Id:  generateId("R", 10),
		App: app,
	}
}

func GetRelease(app, id string) (*Release, error) {
	if id == "" {
		return nil, fmt.Errorf("no release id")
	}

	req := &dynamodb.GetItemInput{
		ConsistentRead: aws.Bool(true),
		Key: map[string]*dynamodb.AttributeValue{
			"id": &dynamodb.AttributeValue{S: aws.String(id)},
		},
		TableName: aws.String(releasesTable(app)),
	}

	res, err := DynamoDB().GetItem(req)

	if err != nil {
		return nil, err
	}

	if res.Item == nil {
		return nil, fmt.Errorf("no such release: %s", id)
	}

	release := releaseFromItem(res.Item)

	return release, nil
}

func (r *Release) Save() error {
	if r.Id == "" {
		return fmt.Errorf("Id must not be blank")
	}

	if r.Created.IsZero() {
		r.Created = time.Now()
	}

	req := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"id":      &dynamodb.AttributeValue{S: aws.String(r.Id)},
			"app":     &dynamodb.AttributeValue{S: aws.String(r.App)},
			"created": &dynamodb.AttributeValue{S: aws.String(r.Created.Format(SortableTime))},
		},
		TableName: aws.String(releasesTable(r.App)),
	}

	if r.Build != "" {
		req.Item["build"] = &dynamodb.AttributeValue{S: aws.String(r.Build)}
	}

	if r.Env != "" {
		req.Item["env"] = &dynamodb.AttributeValue{S: aws.String(r.Env)}
	}

	if r.Manifest != "" {
		req.Item["manifest"] = &dynamodb.AttributeValue{S: aws.String(r.Manifest)}
	}

	_, err := DynamoDB().PutItem(req)

	if err != nil {
		return err
	}

	app, err := GetApp(r.App)

	if err != nil {
		return err
	}

	env := []byte(r.Env)

	if app.Parameters["Key"] != "" {
		cr := crypt.New(os.Getenv("AWS_REGION"), os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"))

		env, err = cr.Encrypt(app.Parameters["Key"], []byte(env))

		if err != nil {
			return err
		}
	}

	NotifySuccess("release:create", map[string]string{"id": r.Id, "app": r.App})

	return S3Put(app.Outputs["Settings"], fmt.Sprintf("releases/%s/env", r.Id), env, true)
}

func (r *Release) Promote() error {
	app, err := GetApp(r.App)

	if err != nil {
		return err
	}

	formation, err := r.Formation()

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

	app.Parameters["Environment"] = r.EnvironmentUrl()
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

	manifest, err := LoadManifest(r.Manifest, app)

	if err != nil {
		return err
	}

	healthOptions := []string{"port", "path", "timeout"}

	for _, entry := range manifest {
		entryName := UpperName(entry.Name)
		for _, option := range healthOptions {
			val := entry.Label(fmt.Sprintf("convox.health.%s", option))
			param := fmt.Sprintf("%sHealth%s", entryName, strings.Title(option))
			fmt.Printf("val %s param %s", val, param)
			app.Parameters[param] = val
		}

		for _, mapping := range entry.PortMappings() {
			certParam := fmt.Sprintf("%sPort%sCertificate", UpperName(entry.Name), mapping.Balancer)
			protoParam := fmt.Sprintf("%sPort%sProtocol", UpperName(entry.Name), mapping.Balancer)
			proxyParam := fmt.Sprintf("%sPort%sProxy", UpperName(entry.Name), mapping.Balancer)
			secureParam := fmt.Sprintf("%sPort%sSecure", UpperName(entry.Name), mapping.Balancer)

			app.Parameters[protoParam] = entry.Label(fmt.Sprintf("convox.port.%s.protocol", mapping.Balancer))

			// default protocol is tcp, or tls if they have a certificate set
			if app.Parameters[protoParam] == "" {
				if app.Parameters[certParam] == "" {
					app.Parameters[protoParam] = "tcp"
				} else {
					app.Parameters[protoParam] = "tls"
				}
			}

			if entry.Label(fmt.Sprintf("convox.port.%s.proxy", mapping.Balancer)) == "true" {
				app.Parameters[proxyParam] = "Yes"
			} else {
				app.Parameters[proxyParam] = "No"
			}

			// only change the secure parameter if a label is set for backwards compat
			switch entry.Label(fmt.Sprintf("convox.port.%s.secure", mapping.Balancer)) {
			case "true":
				app.Parameters[secureParam] = "Yes"
			case "false":
				app.Parameters[secureParam] = "No"
			}

			switch app.Parameters[protoParam] {
			case "https", "tls":
				if app.Parameters[certParam] == "" {
					name := fmt.Sprintf("cert-%d", time.Now().Unix())

					body, key, err := GenerateSelfSignedCertificate("*.*.elb.amazonaws.com")

					if err != nil {
						return err
					}

					input := &iam.UploadServerCertificateInput{
						CertificateBody:       aws.String(string(body)),
						PrivateKey:            aws.String(string(key)),
						ServerCertificateName: aws.String(name),
					}

					// upload certificate
					res, err := IAM().UploadServerCertificate(input)

					if err != nil {
						return err
					}

					app.Parameters[certParam] = *res.ServerCertificateMetadata.Arn
				}
			}
		}
	}

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
		StackName:    aws.String(app.StackName()),
		TemplateURL:  aws.String(url),
		Parameters:   params,
	}

	_, err = UpdateStack(req)

	NotifySuccess("release:promote", map[string]string{
		"app": r.App,
		"id":  r.Id,
	})

	return err
}

func (r *Release) EnvironmentUrl() string {
	app, err := GetApp(r.App)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return ""
	}

	return fmt.Sprintf("https://%s.s3.amazonaws.com/releases/%s/env", app.Outputs["Settings"], r.Id)
}

func (r *Release) Formation() (string, error) {
	app, err := GetApp(r.App)

	if err != nil {
		return "", err
	}

	manifest, err := LoadManifest(r.Manifest, app)

	if err != nil {
		return "", err
	}

	// Bound apps do not use the StackName as ELB name.
	if !app.IsBound() {
		// try to figure out which process to map to the main load balancer
		primary, err := primaryProcess(app.StackName())

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

		for i, entry := range manifest {
			if entry.Name == primary {
				manifest[i].primary = true
			}
		}
	}

	// set the image
	for i, entry := range manifest {
		manifest[i].Image = entry.RegistryImage(app, r.Build)
	}

	manifest, err = r.resolveLinks(*app, &manifest)

	if err != nil {
		return "", err
	}

	return manifest.Formation()
}

func (r *Release) resolveLinks(app App, manifest *Manifest) (Manifest, error) {
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
		// fmt.Printf("ns=kernel at=release.formation at=entry.inspect imageName=%q out=%q err=%q\n", imageName, string(out), err)

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
			host := fmt.Sprintf(`{ "Fn::If" : [ "Enabled%s", { "Fn::GetAtt" : [ "%s", "DNSName" ] }, "DISABLED" ] }`, UpperName(other.Name), mb.ResourceName())

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
			prefix = strings.Replace(prefix, "-", "_", -1)
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
func primaryProcess(stackName string) (string, error) {
	res, err := CloudFormation().GetTemplate(&cloudformation.GetTemplateInput{
		StackName: aws.String(stackName),
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

func releasesTable(app string) string {
	return os.Getenv("DYNAMO_RELEASES")
}

func releaseFromItem(item map[string]*dynamodb.AttributeValue) *Release {
	created, _ := time.Parse(SortableTime, coalesce(item["created"], ""))

	release := &Release{
		Id:       coalesce(item["id"], ""),
		App:      coalesce(item["app"], ""),
		Build:    coalesce(item["build"], ""),
		Env:      coalesce(item["env"], ""),
		Manifest: coalesce(item["manifest"], ""),
		Created:  created,
	}

	return release
}
