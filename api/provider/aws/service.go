package aws

import (
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
	case "papertrail":
		err = fmt.Errorf("papertrail is no longer supported. Create a `syslog` service instead")
	case "syslog":
		req, err = createSyslog(s)
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
		StackName: aws.String(serviceStackName(s)),
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

	// Populate linked apps
	for k, _ := range s.Outputs {
		if strings.HasSuffix(k, "Link") {
			n := DashName(k)
			app := n[:len(n)-5]

			a, err := p.AppGet(app)
			if err != nil {
				return &s, err
			}

			s.Apps = append(s.Apps, *a)
		}
	}

	return &s, nil
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

	// can't link
	switch s.Type {
	case "papertrail":
		return nil, fmt.Errorf("Papertrail linking is no longer supported. Delete the papertrail service and create a new `syslog` service instead")
	}

	// can't link to process or process does not exist
	if process != "" {
		switch s.Type {
		default:
			return nil, fmt.Errorf("Service type %s can not replace a process", s.Type)
		}
		// TODO: Port Resource and Resources and validate that UpperName(process)+"ECSTaskDefinition" exists
	}

	// Update Service and/or App stacks
	switch s.Type {
	case "syslog":
		err = p.ServiceLinkSubscribe(a, s) // Update service to know about App
	case "s3", "sns", "sqs":
		err = p.ServiceLinkSet(a, s) // Updates app with S3_URL
	case "postgres":
		err = p.ServiceLinkReplace(a, s) // Updates app with POSTGRES_URL and PostgresCount=0
	default:
		err = fmt.Errorf("Service type %s does not have a link strategy", s.Type)
	}

	return s, err
}

func (p *AWSProvider) ServiceLinkReplace(a *structs.App, s *structs.Service) error {
	return fmt.Errorf("Replacing a process with a service is not yet implemented")
}

func (p *AWSProvider) ServiceLinkSet(a *structs.App, s *structs.Service) error {
	return fmt.Errorf("Setting an environment variable for a service is not yet implemented")
}

func (p *AWSProvider) ServiceLinkSubscribe(a *structs.App, s *structs.Service) error {
	s.Apps = append(s.Apps, *a)

	formation, err := serviceFormation(s.Type, s)
	if err != nil {
		return err
	}

	req := &cloudformation.UpdateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(serviceStackName(s)),
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

func (p *AWSProvider) ServiceUnlink(name, app, process string) (*structs.Service, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	s, err := p.ServiceGet(name)
	if err != nil {
		return nil, err
	}

	// can't unlink
	switch s.Type {
	case "papertrail":
		return nil, fmt.Errorf("Papertrail unlinking is no longer supported. Delete the papertrail service and create a new `syslog` service instead")
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
	case "syslog":
		err = p.ServiceUnlinkSubscribe(a, s) // Update service to forget about App
	case "s3", "sns", "sqs":
		err = p.ServiceUnlinkSet(a, s) // Updates app without S3_URL
	case "postgres":
		err = p.ServiceUnlinkReplace(a, s) // Updates app without POSTGRES_URL and PostgresCount=1
	default:
		err = fmt.Errorf("Service type %s does not have a unlink strategy", s.Type)
	}

	return s, err
}

func (p *AWSProvider) ServiceUnlinkReplace(a *structs.App, s *structs.Service) error {
	return fmt.Errorf("Un-replacing a process with a service is not yet implemented")
}

func (p *AWSProvider) ServiceUnlinkSet(a *structs.App, s *structs.Service) error {
	return fmt.Errorf("Un-setting an environment variable for a service is not yet implemented")
}

func (p *AWSProvider) ServiceUnlinkSubscribe(a *structs.App, s *structs.Service) error {
	// delete from links
	apps := structs.Apps{}
	for _, linkedApp := range s.Apps {
		if a.Name != linkedApp.Name {
			apps = append(apps, linkedApp)
		}
	}

	s.Apps = apps

	formation, err := serviceFormation(s.Type, s)
	if err != nil {
		return err
	}

	req := &cloudformation.UpdateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(serviceStackName(s)),
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

func createSyslog(s *structs.Service) (*cloudformation.CreateStackInput, error) {
	u, err := url.Parse(s.Parameters["Url"])
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "udp", "tcp", "tcp+tls":
		// proceed
	default:
		return nil, fmt.Errorf("Invalid url scheme `%s`. Allowed schemes are `udp`, `tcp`, `tcp+tls`", u.Scheme)
	}

	formation, err := serviceFormation(s.Type, nil)
	if err != nil {
		return nil, err
	}

	req := &cloudformation.CreateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(serviceStackName(s)),
		TemplateBody: aws.String(formation),
	}

	return req, nil
}

func serviceFormation(kind string, data interface{}) (string, error) {
	d, err := buildTemplate(fmt.Sprintf("service/%s", kind), "service", data)
	if err != nil {
		return "", err
	}

	return string(d), nil
}

func serviceFromStack(stack *cloudformation.Stack) structs.Service {
	name := *stack.StackName
	params := stackParameters(stack)
	tags := stackTags(stack)
	if value, ok := tags["Name"]; ok {
		// StackName probably includes the Rack prefix, prefer Name tag.
		name = value
	}

	exports := make(map[string]string)

	if humanStatus(*stack.StackStatus) == "running" {
		switch tags["Service"] {
		case "syslog":
			exports["URL"] = params["Url"]
		}
	}

	return structs.Service{
		Name:       name,
		Type:       tags["Service"],
		Status:     humanStatus(*stack.StackStatus),
		Outputs:    stackOutputs(stack),
		Parameters: params,
		Tags:       tags,
		Exports:    exports,
	}
}

func serviceStackName(s *structs.Service) string {
	// Tags are present but "Name" tag is not so we have an "unbound" service with no rack name prefix
	if s.Tags != nil && s.Tags["Name"] == "" {
		return s.Name
	}

	// otherwise prefix the stack name with the rack name
	return fmt.Sprintf("%s-%s", os.Getenv("RACK"), s.Name)
}
