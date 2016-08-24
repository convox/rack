package models

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/convox/rack/api/crypt"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/manifest"
)

// set to false when testing for deterministic ports
var ManifestRandomPorts = true

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
		App:     app,
		Created: time.Now(),
		Id:      generateId("R", 10),
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

	if !app.IsBound() {
		return fmt.Errorf("unbound apps are no longer supported for promotion")
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

	oldVersion := app.Parameters["Version"]

	app.Parameters["Environment"] = r.EnvironmentUrl()
	app.Parameters["Kernel"] = CustomTopic
	app.Parameters["Release"] = r.Id
	app.Parameters["Version"] = os.Getenv("RELEASE")
	app.Parameters["VPCCIDR"] = os.Getenv("VPCCIDR")

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

	m, err := manifest.Load([]byte(r.Manifest))
	if err != nil {
		return err
	}

	for _, entry := range m.Services {
		// set all of WebCount=1, WebCpu=0, WebMemory=256 and WebFormation=1,0,256 style parameters
		// so new deploys and rollbacks have the expected parameters
		if vals, ok := app.Parameters[fmt.Sprintf("%sFormation", UpperName(entry.Name))]; ok {
			parts := strings.SplitN(vals, ",", 3)
			if len(parts) != 3 {
				return fmt.Errorf("%s formation settings not in Count,Cpu,Memory format", entry.Name)
			}

			_, err = strconv.Atoi(parts[0])
			if err != nil {
				return fmt.Errorf("%s %s not numeric", entry.Name, "count")
			}

			_, err = strconv.Atoi(parts[1])
			if err != nil {
				return fmt.Errorf("%s %s not numeric", entry.Name, "CPU")
			}

			_, err = strconv.Atoi(parts[2])
			if err != nil {
				return fmt.Errorf("%s %s not numeric", entry.Name, "memory")
			}

			app.Parameters[fmt.Sprintf("%sDesiredCount", UpperName(entry.Name))] = parts[0]
			app.Parameters[fmt.Sprintf("%sCpu", UpperName(entry.Name))] = parts[1]
			app.Parameters[fmt.Sprintf("%sMemory", UpperName(entry.Name))] = parts[2]
		} else {
			parts := []string{"1", "0", "256"}

			if v := app.Parameters[fmt.Sprintf("%sDesiredCount", UpperName(entry.Name))]; v != "" {
				parts[0] = v
			}

			if v := app.Parameters[fmt.Sprintf("%sCpu", UpperName(entry.Name))]; v != "" {
				parts[1] = v
			}

			if v := app.Parameters[fmt.Sprintf("%sMemory", UpperName(entry.Name))]; v != "" {
				parts[2] = v
			}

			app.Parameters[fmt.Sprintf("%sFormation", UpperName(entry.Name))] = strings.Join(parts, ",")
		}

		for _, mapping := range entry.Ports {
			certParam := fmt.Sprintf("%sPort%dCertificate", UpperName(entry.Name), mapping.Balancer)
			protoParam := fmt.Sprintf("%sPort%dProtocol", UpperName(entry.Name), mapping.Balancer)
			proxyParam := fmt.Sprintf("%sPort%dProxy", UpperName(entry.Name), mapping.Balancer)
			secureParam := fmt.Sprintf("%sPort%dSecure", UpperName(entry.Name), mapping.Balancer)

			proto := entry.Labels[fmt.Sprintf("convox.port.%d.protocol", mapping.Balancer)]

			// if the proto param is set to a non-default value and doesnt match the label, error
			if ap, ok := app.Parameters[protoParam]; ok {
				if ap != "tcp" && ap != proto {
					return fmt.Errorf("%s parameter has been deprecated. Please set the convox.port.%d.protocol label instead", protoParam, mapping.Balancer)
				}
			}

			// if the proxy param is set and doesnt match the label, error
			if ap, ok := app.Parameters[proxyParam]; ok {
				if ap == "Yes" && entry.Labels[fmt.Sprintf("convox.port.%d.proxy", mapping.Balancer)] != "true" {
					return fmt.Errorf("%s parameter has been deprecated. Please set the convox.port.%d.proxy label instead", proxyParam, mapping.Balancer)
				}
			}

			// if the secure param is set and doesnt match the label, error
			if ap, ok := app.Parameters[secureParam]; ok {
				if ap == "Yes" && entry.Labels[fmt.Sprintf("convox.port.%d.secure", mapping.Balancer)] != "true" {
					return fmt.Errorf("%s parameter has been deprecated. Please set the convox.port.%d.secure label instead", secureParam, mapping.Balancer)
				}
			}

			switch proto {
			case "https", "tls":
				if app.Parameters[certParam] == "" {
					name := fmt.Sprintf("cert-%s-%d-%05d", os.Getenv("RACK"), time.Now().Unix(), rand.Intn(100000))

					body, key, err := generateSelfSignedCertificate("*.*.elb.amazonaws.com")
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

	// randomize the instance ports for older apps so we can upgrade smoothly
	if oldVersion < "20160818013241" {
		for key := range app.Parameters {
			if strings.HasSuffix(key, "Host") {
				app.Parameters[key] = strconv.Itoa(rand.Intn(50000) + 10000)
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

	// loop until we can find the template
	if err := waitForTemplate(app.Outputs["Settings"], r.Id); err != nil {
		return fmt.Errorf("error waiting for template: %s", err)
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

	manifest, err := manifest.Load([]byte(r.Manifest))
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
		_, ok := manifest.Services["web"]
		if primary == "" && ok {
			primary = "web"
		}

		// if we still dont have a primary try the first process with external ports
		if primary == "" && manifest.HasExternalPorts() {
			for _, entry := range manifest.Services {
				if len(entry.ExternalPorts()) > 0 {
					primary = entry.Name
					break
				}
			}
		}

		for _, entry := range manifest.Services {
			if entry.Name == primary {
				//TODO not sure this indirection is required
				dup := manifest.Services[entry.Name]
				dup.Primary = true
				manifest.Services[entry.Name] = dup
			}
		}
	}

	// set the image
	for i, entry := range manifest.Services {
		s := manifest.Services[i]
		s.Image = entry.RegistryImage(app.Name, r.Build, app.Outputs)
		manifest.Services[i] = s
	}

	manifest, err = r.resolveLinks(*app, manifest)
	if err != nil {
		return "", err
	}

	return app.Formation(*manifest)
}

func (r *Release) resolveLinks(app App, manifest *manifest.Manifest) (*manifest.Manifest, error) {
	m := *manifest

	// HACK: need an app of type structs.App for docker login.
	// Should be fixed/removed once proper logic is moved over to structs.App
	// That includes moving Formation() around
	sa := structs.App{
		Name:       app.Name,
		Release:    app.Release,
		Status:     app.Status,
		Outputs:    app.Outputs,
		Parameters: app.Parameters,
		Tags:       app.Tags,
	}
	endpoint, err := AppDockerLogin(sa)
	if err != nil {
		return &m, fmt.Errorf("could not log into %q", endpoint)
	}

	for i, entry := range m.Services {
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
			return &m, fmt.Errorf("could not pull %q", imageName)
		}

		cmd = exec.Command("docker", "inspect", imageName)
		out, err = cmd.CombinedOutput()
		// fmt.Printf("ns=kernel at=release.formation at=entry.inspect imageName=%q out=%q err=%q\n", imageName, string(out), err)
		if err != nil {
			return &m, fmt.Errorf("could not inspect %q", imageName)
		}

		err = json.Unmarshal(out, &inspect)
		if err != nil {
			fmt.Printf("ns=kernel at=release.formation at=entry.unmarshal err=%q\n", err)
			return &m, fmt.Errorf("could not inspect %q", imageName)
		}

		entry.Exports = make(map[string]string)
		linkableEnvs := make([]string, len(entry.Environment))
		for k, v := range entry.Environment {
			val := fmt.Sprintf("%s=%s", k, v)
			linkableEnvs = append(linkableEnvs, val)
		}

		if len(inspect) == 1 {
			linkableEnvs = append(linkableEnvs, inspect[0].Config.Env...)
		}

		for _, val := range linkableEnvs {
			if strings.HasPrefix(val, "LINK_") {
				parts := strings.SplitN(val, "=", 2)
				if len(parts) == 2 {
					entry.Exports[parts[0]] = parts[1]
					m.Services[i] = entry
				}
			}
		}
	}

	for i, entry := range m.Services {
		entry.LinkVars = make(map[string]template.HTML)
		for _, link := range entry.Links {
			other, ok := m.Services[link]
			if !ok {
				return &m, fmt.Errorf("Cannot find link %q", link)
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
			m.Services[i] = entry
		}
	}

	return &m, nil
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

func waitForTemplate(bucket string, id string) error {
	tick := time.Tick(1 * time.Second)
	timeout := time.Tick(1 * time.Minute)

	for {
		select {
		case <-tick:
			fmt.Printf("ns=kernel at=waitForTemplate.tick bucket=%q release=%q\n", bucket, id)
			_, err := s3Get(bucket, fmt.Sprintf("templates/%s", id))
			if err == nil {
				fmt.Printf("ns=kernel at=waitForTemplate.tick status=found bucket=%q release=%q\n", bucket, id)
				return nil
			}
		case <-timeout:
			fmt.Printf("ns=kernel at=waitForTemplate.tick error=timeout bucket=%q release=%q\n", bucket, id)
			return fmt.Errorf("timeout")
		}
	}

	fmt.Printf("ns=kernel at=waitForTemplate.tick error=unknown bucket=%q release=%q\n", bucket, id)
	return fmt.Errorf("unknown error")
}
