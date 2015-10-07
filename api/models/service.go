package models

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/kinesis"
)

type Service struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Type   string `json:"type"`
	URL    string `json:"url"`

	Outputs    map[string]string `json:"-"`
	Parameters map[string]string `json:"-"`
	Tags       map[string]string `json:"-"`
}

type Services []Service

func ListServices() (Services, error) {
	res, err := CloudFormation().DescribeStacks(&cloudformation.DescribeStacksInput{})

	if err != nil {
		return nil, err
	}

	services := make(Services, 0)

	for _, stack := range res.Stacks {
		tags := stackTags(stack)

		if tags["System"] == "convox" && tags["Type"] == "service" {
			services = append(services, *serviceFromStack(stack))
		}
	}

	sort.Sort(services)

	return services, nil
}

func GetService(name string) (*Service, error) {
	res, err := CloudFormation().DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String(name)})

	if err != nil {
		return nil, err
	}

	return serviceFromStack(res.Stacks[0]), nil
}

func (s *Service) Create() error {
	formation, err := s.Formation()

	if err != nil {
		return err
	}

	params := map[string]string{
		"Password": generateId("", 30),
		"Subnets":  os.Getenv("SUBNETS"),
		"Vpc":      os.Getenv("VPC"),
	}

	tags := map[string]string{
		"System":  "convox",
		"Type":    "service",
		"Service": s.Type,
	}

	req := &cloudformation.CreateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(s.Name),
		TemplateBody: aws.String(formation),
	}

	for key, value := range params {
		req.Parameters = append(req.Parameters, &cloudformation.Parameter{ParameterKey: aws.String(key), ParameterValue: aws.String(value)})
	}

	for key, value := range tags {
		req.Tags = append(req.Tags, &cloudformation.Tag{Key: aws.String(key), Value: aws.String(value)})
	}

	_, err = CloudFormation().CreateStack(req)

	return err
}

func (s *Service) CreatePapertrail() error {
	if s.URL == "" {
		return fmt.Errorf("Papertrail URL is required")
	}

	input := struct {
		ARNs []string
	}{
		[]string{},
	}

	formation, err := buildTemplate(fmt.Sprintf("service/%s", s.Type), "service", input)

	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	params := map[string]string{
		"Url": s.URL,
	}

	tags := map[string]string{
		"System":  "convox",
		"Type":    "service",
		"Service": s.Type,
	}

	req := &cloudformation.CreateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(s.Name),
		TemplateBody: aws.String(formation),
	}

	for key, value := range params {
		req.Parameters = append(req.Parameters, &cloudformation.Parameter{ParameterKey: aws.String(key), ParameterValue: aws.String(value)})
	}

	for key, value := range tags {
		req.Tags = append(req.Tags, &cloudformation.Tag{Key: aws.String(key), Value: aws.String(value)})
	}

	_, err = CloudFormation().CreateStack(req)

	return err
}

func (s *Service) LinkPapertrail(app App) error {
	// get full Kinesis ARN for app
	req, err := Kinesis().DescribeStream(&kinesis.DescribeStreamInput{
		StreamName: aws.String(app.Outputs["Kinesis"]),
	})

	arn := *req.StreamDescription.StreamARN

	if err != nil {
		return err
	}

	// get all ARNs linked to service
	arns := []string{}

	a := s.Parameters["Arns"]
	if a != "" {
		arns = strings.Split(a, ",")
	}

	// append app ARN if not already linked
	for _, a := range arns {
		if a == arn {
			return fmt.Errorf("app %s is already linked", app.Name)
		}
	}

	arns = append(arns, arn)

	// build Name -> ARN map for CloudFormation template EventSourceMapping resources
	m := map[string]string{}

	for _, v := range arns {
		// e.g. arn:aws:kinesis:us-east-1:901416387788:stream/convox-Kinesis-L6MUKT1VH451 -> convox
		a := strings.Split(strings.Split(v, "/")[1], "-")[0]
		m[a] = v
	}

	input := struct {
		ARNs map[string]string
	}{
		m,
	}

	formation, err := buildTemplate(fmt.Sprintf("service/%s", s.Type), "service", input)

	if err != nil {
		return err
	}

	// Update stack with all linked ARNs and EventSourceMappings
	_, err = CloudFormation().UpdateStack(&cloudformation.UpdateStackInput{
		StackName:    aws.String(s.Name),
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		Parameters: []*cloudformation.Parameter{
			&cloudformation.Parameter{
				ParameterKey:   aws.String("Arns"),
				ParameterValue: aws.String(strings.Join(arns, ",")),
			},
			&cloudformation.Parameter{
				ParameterKey:   aws.String("Url"),
				ParameterValue: aws.String(s.Parameters["Url"]),
			},
		},
		TemplateBody: aws.String(formation),
	})

	return err
}

func (s *Service) UnlinkPapertrail(app App) error {
	// get full Kinesis ARN for app
	req, err := Kinesis().DescribeStream(&kinesis.DescribeStreamInput{
		StreamName: aws.String(app.Outputs["Kinesis"]),
	})

	arn := *req.StreamDescription.StreamARN

	if err != nil {
		return err
	}

	// get all ARNs linked to service
	arns := []string{}

	a := s.Parameters["Arns"]

	if a != "" {
		arns = strings.Split(a, ",")
	}

	if !strings.Contains(a, arn) {
		return fmt.Errorf("app %s is not linked", app.Name)
	}

	// append ARNs except for app to unlink
	newArns := []string{}
	for _, a := range arns {
		if a != arn {
			newArns = append(newArns, a)
		}
	}

	arns = newArns

	// build Name -> ARN map for CloudFormation template EventSourceMapping resources
	m := map[string]string{}

	for _, v := range arns {
		// e.g. arn:aws:kinesis:us-east-1:901416387788:stream/convox-Kinesis-L6MUKT1VH451 -> convox
		a := strings.Split(strings.Split(v, "/")[1], "-")[0]
		m[a] = v
	}

	input := struct {
		ARNs map[string]string
	}{
		m,
	}

	formation, err := buildTemplate(fmt.Sprintf("service/%s", s.Type), "service", input)

	if err != nil {
		return err
	}

	// Update stack with all linked ARNs and EventSourceMappings
	_, err = CloudFormation().UpdateStack(&cloudformation.UpdateStackInput{
		StackName:    aws.String(s.Name),
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		Parameters: []*cloudformation.Parameter{
			&cloudformation.Parameter{
				ParameterKey:   aws.String("Arns"),
				ParameterValue: aws.String(strings.Join(arns, ",")),
			},
			&cloudformation.Parameter{
				ParameterKey:   aws.String("Url"),
				ParameterValue: aws.String(s.Parameters["Url"]),
			},
		},
		TemplateBody: aws.String(formation),
	})

	return err
}

func (s *Service) Delete() error {
	name := s.Name

	_, err := CloudFormation().DeleteStack(&cloudformation.DeleteStackInput{StackName: aws.String(name)})

	if err != nil {
		return err
	}

	return nil
}

func (s *Service) Formation() (string, error) {
	data, err := buildTemplate(fmt.Sprintf("service/%s", s.Type), "service", nil)

	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (s *Service) SubscribeLogs(output chan []byte, quit chan bool) error {
	done := make(chan bool)

	switch s.Tags["Service"] {
	case "postgres":
		go subscribeRDS(s.Name, s.Name, output, done)
	case "redis":
		resources, err := ListResources(s.Name)

		if err != nil {
			return err
		}

		go subscribeKinesis(resources["Kinesis"].Id, output, done)
	}
	return nil
}

func (ss Services) Len() int {
	return len(ss)
}

func (ss Services) Less(i, j int) bool {
	return ss[i].Name < ss[j].Name
}

func (ss Services) Swap(i, j int) {
	ss[i], ss[j] = ss[j], ss[i]
}

func serviceFromStack(stack *cloudformation.Stack) *Service {
	outputs := stackOutputs(stack)
	parameters := stackParameters(stack)
	tags := stackTags(stack)
	url := ""

	if humanStatus(*stack.StackStatus) == "running" {
		switch tags["Service"] {
		case "postgres":
			url = fmt.Sprintf("postgres://%s:%s@%s:%s/%s", outputs["EnvPostgresUsername"], outputs["EnvPostgresPassword"], outputs["Port5432TcpAddr"], outputs["Port5432TcpPort"], outputs["EnvPostgresDatabase"])
		case "redis":
			url = fmt.Sprintf("redis://u@%s:%s/%s", outputs["Port6379TcpAddr"], outputs["Port6379TcpPort"], outputs["EnvRedisDatabase"])
		}
	}

	return &Service{
		Name:       cs(stack.StackName, "<unknown>"),
		Type:       tags["Service"],
		Status:     humanStatus(*stack.StackStatus),
		Outputs:    outputs,
		Parameters: parameters,
		Tags:       tags,
		URL:        url,
	}
}
