package aws

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/api/structs"
)

func (p *AWSProvider) ServiceCreate(name, kind string, params map[string]string) (*structs.Service, error) {
	_, err := p.ServiceGet(name)
	if awsError(err) != "ValidationError" {
		return nil, fmt.Errorf("service named %s already exists", name)
	}

	s := &structs.Service{
		Name:       name,
		Parameters: cfParams(params),
		Type:       kind,
	}

	var req *cloudformation.CreateStackInput

	switch s.Type {
	case "mysql", "postgres", "redis", "sqs":
		req, err = createService(s)
	case "fluentd":
		req, err = createServiceURL(s, "tcp")
	case "s3":
		s.Parameters["Topic"] = fmt.Sprintf("%s-%s", os.Getenv("RACK"), s.Parameters["Topic"])
		req, err = createService(s)
	case "sns":
		s.Parameters["Queue"] = fmt.Sprintf("%s-%s", os.Getenv("RACK"), s.Parameters["Queue"])
		req, err = createService(s)
	case "syslog":
		req, err = createServiceURL(s, "tcp", "tcp+tls", "udp")
	case "webhook":
		s.Parameters["Url"] = fmt.Sprintf("http://%s/sns?endpoint=%s", os.Getenv("NOTIFICATION_HOST"), url.QueryEscape(s.Parameters["Url"]))
		s.Parameters["NotificationTopic"] = notificationTopic
		s.Parameters["CustomTopic"] = customTopic
		req, err = createServiceURL(s, "http", "https")
	default:
		err = fmt.Errorf("Invalid service type: %s", s.Type)
	}
	if err != nil {
		return s, err
	}

	// pass through service parameters as Cloudformation Parameters
	for key, value := range s.Parameters {
		req.Parameters = append(req.Parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String(key),
			ParameterValue: aws.String(value),
		})
	}

	// tag the service
	tags := map[string]string{
		"Rack":    os.Getenv("RACK"),
		"System":  "convox",
		"Service": s.Type,
		"Type":    "service",
		"Name":    s.Name,
	}

	for key, value := range tags {
		req.Tags = append(req.Tags, &cloudformation.Tag{Key: aws.String(key), Value: aws.String(value)})
	}

	_, err = p.cloudformation().CreateStack(req)

	p.EventSend(&structs.Event{
		Action: "service:create",
		Data: map[string]string{
			"name": s.Name,
			"type": s.Type,
		},
	}, err)

	return s, err
}

func (p *AWSProvider) ServiceDelete(name string) (*structs.Service, error) {
	s, err := p.ServiceGet(name)
	if err != nil {
		return nil, err
	}

	_, err = p.cloudformation().DeleteStack(&cloudformation.DeleteStackInput{
		StackName: aws.String(s.Stack),
	})

	p.EventSend(&structs.Event{
		Action: "service:delete",
		Data: map[string]string{
			"name": s.Name,
			"type": s.Type,
		},
	}, err)

	return s, err
}

func (p *AWSProvider) ServiceGet(name string) (*structs.Service, error) {
	var res *cloudformation.DescribeStacksOutput
	var err error

	// try 'convox-myservice', and if not found try 'myservice'
	res, err = p.describeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(os.Getenv("RACK") + "-" + name),
	})
	if awsError(err) == "ValidationError" {
		res, err = p.describeStacks(&cloudformation.DescribeStacksInput{
			StackName: aws.String(name),
		})
	}
	if err != nil {
		return nil, err
	}
	if len(res.Stacks) != 1 {
		return nil, fmt.Errorf("could not load stack for service: %s", name)
	}

	s := serviceFromStack(res.Stacks[0])

	if s.Tags["Rack"] != "" && s.Tags["Rack"] != os.Getenv("RACK") {
		return nil, fmt.Errorf("no such stack on this rack: %s", name)
	}

	if s.Status == "failed" {
		eres, err := p.describeStackEvents(&cloudformation.DescribeStackEventsInput{
			StackName: aws.String(*res.Stacks[0].StackName),
		})
		if err != nil {
			return &s, err
		}

		for _, event := range eres.StackEvents {
			if *event.ResourceStatus == cloudformation.ResourceStatusCreateFailed {
				s.StatusReason = *event.ResourceStatusReason
				break
			}
		}
	}

	switch s.Tags["Service"] {
	case "mysql":
		s.Exports["URL"] = fmt.Sprintf("mysql://%s:%s@%s:%s/%s", s.Outputs["EnvMysqlUsername"], s.Outputs["EnvMysqlPassword"], s.Outputs["Port3306TcpAddr"], s.Outputs["Port3306TcpPort"], s.Outputs["EnvMysqlDatabase"])
	case "papertrail":
		s.Exports["URL"] = s.Parameters["Url"]
	case "postgres":
		s.Exports["URL"] = fmt.Sprintf("postgres://%s:%s@%s:%s/%s", s.Outputs["EnvPostgresUsername"], s.Outputs["EnvPostgresPassword"], s.Outputs["Port5432TcpAddr"], s.Outputs["Port5432TcpPort"], s.Outputs["EnvPostgresDatabase"])
	case "redis":
		s.Exports["URL"] = fmt.Sprintf("redis://%s:%s/%s", s.Outputs["Port6379TcpAddr"], s.Outputs["Port6379TcpPort"], s.Outputs["EnvRedisDatabase"])
	case "s3":
		s.Exports["URL"] = fmt.Sprintf("s3://%s:%s@%s", s.Outputs["AccessKey"], s.Outputs["SecretAccessKey"], s.Outputs["Bucket"])
	case "sns":
		s.Exports["URL"] = fmt.Sprintf("sns://%s:%s@%s", s.Outputs["AccessKey"], s.Outputs["SecretAccessKey"], s.Outputs["Topic"])
	case "sqs":
		if u, err := url.Parse(s.Outputs["Queue"]); err == nil {
			u.Scheme = "sqs"
			u.User = url.UserPassword(s.Outputs["AccessKey"], s.Outputs["SecretAccessKey"])
			s.Exports["URL"] = u.String()
		}
	case "webhook":
		if parsedURL, err := url.Parse(s.Parameters["Url"]); err == nil {
			s.Exports["URL"] = parsedURL.Query().Get("endpoint")
		}
	}

	// Populate linked apps
	for k, _ := range s.Outputs {
		if strings.HasSuffix(k, "Link") {
			n := DashName(k)
			app := n[:len(n)-4]

			a, err := p.AppGet(app)
			if err != nil {
				return &s, err
			}

			s.Apps = append(s.Apps, *a)
		}
	}

	return &s, nil
}

// ServiceList lists the Services
func (p *AWSProvider) ServiceList() (structs.Services, error) {
	res, err := p.describeStacks(&cloudformation.DescribeStacksInput{})
	if err != nil {
		return nil, err
	}

	services := structs.Services{}

	for _, stack := range res.Stacks {
		tags := stackTags(stack)

		// if it's a service and the Rack tag is either the current rack or blank
		if tags["System"] == "convox" && tags["Type"] == "service" {
			if tags["Rack"] == os.Getenv("RACK") || tags["Rack"] == "" {
				services = append(services, serviceFromStack(stack))
			}
		}
	}

	return services, nil
}

func (p *AWSProvider) ServiceLink(name, app, process string) (*structs.Service, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	s, err := p.ServiceGet(name)
	if err != nil {
		return nil, err
	}

	// already linked
	for _, linkedApp := range s.Apps {
		if a.Name == linkedApp.Name {
			return nil, fmt.Errorf("Service %s is already linked to app %s", s.Name, a.Name)
		}
	}

	// Update Service and/or App stacks
	switch s.Type {
	case "fluentd", "syslog":
		err = p.linkService(a, s) // Update service to know about App
	default:
		err = fmt.Errorf("Service type %s does not have a link strategy", s.Type)
	}

	return s, err
}

func (p *AWSProvider) ServiceUnlink(name, app, process string) (*structs.Service, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	s, err := p.ServiceGet(name)
	if err != nil {
		return nil, err
	}

	// already linked
	linked := false
	for _, linkedApp := range s.Apps {
		if a.Name == linkedApp.Name {
			linked = true
			break
		}
	}

	if !linked {
		return nil, fmt.Errorf("Service %s is not linked to app %s", s.Name, a.Name)
	}

	// Update Service and/or App stacks
	switch s.Type {
	case "fluentd", "syslog":
		err = p.unlinkService(a, s) // Update service to forget about App
	default:
		err = fmt.Errorf("Service type %s does not have a unlink strategy", s.Type)
	}

	return s, err
}

// ServiceUpdate updates a Service with new params
func (p *AWSProvider) ServiceUpdate(name string, params map[string]string) (*structs.Service, error) {
	s, err := p.ServiceGet(name)
	if err != nil {
		return nil, err
	}

	for key, value := range cfParams(params) {
		s.Parameters[key] = value
	}

	err = p.updateService(s)

	return s, err
}

func createService(s *structs.Service) (*cloudformation.CreateStackInput, error) {
	formation, err := serviceFormation(s.Type, nil)
	if err != nil {
		return nil, err
	}

	if err := appendSystemParameters(s); err != nil {
		return nil, err
	}

	if err := filterFormationParameters(s, formation); err != nil {
		return nil, err
	}

	req := &cloudformation.CreateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(fmt.Sprintf("%s-%s", os.Getenv("RACK"), s.Name)),
		TemplateBody: aws.String(formation),
	}

	return req, nil
}

func createServiceURL(s *structs.Service, allowedProtocols ...string) (*cloudformation.CreateStackInput, error) {
	if s.Parameters["Url"] == "" {
		return nil, fmt.Errorf("Must specify a URL")
	}

	u, err := url.Parse(s.Parameters["Url"])
	if err != nil {
		return nil, err
	}

	valid := false

	for _, p := range allowedProtocols {
		if u.Scheme == p {
			valid = true
			break
		}
	}

	if !valid {
		return nil, fmt.Errorf("Invalid URL scheme: %s. Allowed schemes are: %s", u.Scheme, strings.Join(allowedProtocols, ", "))
	}

	return createService(s)
}

func (p *AWSProvider) updateService(s *structs.Service) error {
	formation, err := serviceFormation(s.Type, s)
	if err != nil {
		return err
	}

	if err := appendSystemParameters(s); err != nil {
		return err
	}

	if err := filterFormationParameters(s, formation); err != nil {
		return err
	}

	req := &cloudformation.UpdateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(s.Stack),
		TemplateBody: aws.String(formation),
	}

	for key, value := range s.Parameters {
		req.Parameters = append(req.Parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String(key),
			ParameterValue: aws.String(value),
		})
	}

	_, err = p.cloudformation().UpdateStack(req)

	return err
}

// add to links
func (p *AWSProvider) linkService(a *structs.App, s *structs.Service) error {
	s.Apps = append(s.Apps, *a)

	return p.updateService(s)
}

// delete from links
func (p *AWSProvider) unlinkService(a *structs.App, s *structs.Service) error {
	apps := structs.Apps{}
	for _, linkedApp := range s.Apps {
		if a.Name != linkedApp.Name {
			apps = append(apps, linkedApp)
		}
	}

	s.Apps = apps

	return p.updateService(s)
}

func appendSystemParameters(s *structs.Service) error {
	password, err := generatePassword()
	if err != nil {
		return err
	}

	s.Parameters["Password"] = password
	s.Parameters["Subnets"] = os.Getenv("SUBNETS")
	s.Parameters["SubnetsPrivate"] = coalesceString(os.Getenv("SUBNETS_PRIVATE"), os.Getenv("SUBNETS"))
	s.Parameters["Vpc"] = os.Getenv("VPC")
	s.Parameters["VpcCidr"] = os.Getenv("VPCCIDR")

	return nil
}

func coalesceString(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

func filterFormationParameters(s *structs.Service, formation string) error {
	var params struct {
		Parameters map[string]interface{}
	}

	if err := json.Unmarshal([]byte(formation), &params); err != nil {
		return err
	}

	for key := range s.Parameters {
		if _, ok := params.Parameters[key]; !ok {
			delete(s.Parameters, key)
		}
	}

	return nil
}

func generatePassword() (string, error) {
	data := make([]byte, 1024)

	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)

	return hex.EncodeToString(hash[:])[0:30], nil
}

func serviceFormation(kind string, data interface{}) (string, error) {
	d, err := buildTemplate(fmt.Sprintf("service/%s", kind), "service", data)
	if err != nil {
		return "", err
	}

	return string(d), nil
}

func serviceFromStack(stack *cloudformation.Stack) structs.Service {
	params := stackParameters(stack)
	tags := stackTags(stack)
	name := coalesceString(tags["Name"], *stack.StackName)

	exports := map[string]string{}

	if url, ok := params["Url"]; ok {
		exports["URL"] = url
	}

	return structs.Service{
		Name:       name,
		Stack:      *stack.StackName,
		Type:       tags["Service"],
		Status:     humanStatus(*stack.StackStatus),
		Outputs:    stackOutputs(stack),
		Parameters: params,
		Tags:       tags,
		Exports:    exports,
	}
}
