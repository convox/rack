package aws

import (
	"fmt"
	"os"

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
	case "syslog":
		req, err = createSyslog(s)
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

	svc := serviceFromStack(res.Stacks[0])

	if svc.Tags["Rack"] != "" && svc.Tags["Rack"] != os.Getenv("RACK") {
		return nil, fmt.Errorf("no such stack on this rack: %s", name)
	}

	if svc.Status == "failed" {
		eres, err := p.describeStackEvents(&cloudformation.DescribeStackEventsInput{
			StackName: aws.String(*res.Stacks[0].StackName),
		})
		if err != nil {
			return &svc, err
		}

		for _, event := range eres.StackEvents {
			if *event.ResourceStatus == cloudformation.ResourceStatusCreateFailed {
				svc.StatusReason = *event.ResourceStatusReason
				break
			}
		}
	}

	return &svc, nil
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
	case "syslog":
		// noop
	default:
		return nil, fmt.Errorf("Service type %s can not be linked", s.Type)
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
	return fmt.Errorf("Replacing a process with a service is not yet implemented.")
}

func (p *AWSProvider) ServiceLinkSet(a *structs.App, s *structs.Service) error {
	return fmt.Errorf("Setting an environment variable for a service is not yet implemented.")
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

func createSyslog(s *structs.Service) (*cloudformation.CreateStackInput, error) {
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
	tags := stackTags(stack)
	if value, ok := tags["Name"]; ok {
		// StackName probably includes the Rack prefix, prefer Name tag.
		name = value
	}

	exports := make(map[string]string)

	return structs.Service{
		Name:       name,
		Type:       tags["Service"],
		Status:     humanStatus(*stack.StackStatus),
		Outputs:    stackOutputs(stack),
		Parameters: stackParameters(stack),
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
