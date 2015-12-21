package models

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/client"
)

type Service client.Service
type Services []Service

func ListServices() (Services, error) {
	res, err := CloudFormation().DescribeStacks(&cloudformation.DescribeStacksInput{})

	if err != nil {
		return nil, err
	}

	services := make(Services, 0)

	for _, stack := range res.Stacks {
		tags := stackTags(stack)

		//NOTE: services used to not have a tag so the empty "Rack"
		//      is for untagged services
		if tags["System"] == "convox" &&
			tags["Type"] == "service" &&
			(tags["Rack"] == os.Getenv("RACK") || tags["Rack"] == "") {
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

	service := serviceFromStack(res.Stacks[0])

	if service.Status == "failed" {
		eres, err := CloudFormation().DescribeStackEvents(
			&cloudformation.DescribeStackEventsInput{StackName: aws.String(name)},
		)

		if err != nil {
			return nil, err
		}

		for _, event := range eres.StackEvents {
			if *event.ResourceStatus == cloudformation.ResourceStatusCreateFailed {
				service.StatusReason = *event.ResourceStatusReason
				break
			}
		}
	}

	return service, nil
}

func (s *Service) Create() error {
	var req *cloudformation.CreateStackInput
	var err error

	switch s.Type {
	case "papertrail":
		req, err = s.CreatePapertrail()
	case "webhook":
		req, err = s.CreateWebhook()
	default:
		req, err = s.CreateDatastore()
	}

	if err != nil {
		return err
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
	}

	for key, value := range tags {
		req.Tags = append(req.Tags, &cloudformation.Tag{Key: aws.String(key), Value: aws.String(value)})
	}

	_, err = CloudFormation().CreateStack(req)

	if err != nil {
		NotifySuccess("service:create", map[string]string{
			"name": s.Name,
			"type": s.Type,
		})
	}

	return err
}

func (s *Service) Delete() error {
	name := s.Name

	_, err := CloudFormation().DeleteStack(&cloudformation.DeleteStackInput{StackName: aws.String(name)})

	if err != nil {
		return err
	}

	NotifySuccess("service:delete", map[string]string{
		"name": s.Name,
		"type": s.Type,
	})

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
	switch s.Tags["Service"] {
	case "postgres":
		go subscribeRDS(s.Name, s.Name, output, quit)
	case "redis":
		resources, err := ListResources(s.Name)

		if err != nil {
			return err
		}

		go subscribeKinesis(resources["Kinesis"].Id, output, quit)
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

//NOTE: let's figure out how to assemble the exports from the outputs
func serviceFromStack(stack *cloudformation.Stack) *Service {
	outputs := stackOutputs(stack)
	parameters := stackParameters(stack)
	tags := stackTags(stack)
	exports := make(map[string]string)

	if humanStatus(*stack.StackStatus) == "running" {
		switch tags["Service"] {
		case "papertrail":
			exports["URL"] = parameters["Url"]
		case "postgres":
			exports["URL"] = fmt.Sprintf("postgres://%s:%s@%s:%s/%s", outputs["EnvPostgresUsername"], outputs["EnvPostgresPassword"], outputs["Port5432TcpAddr"], outputs["Port5432TcpPort"], outputs["EnvPostgresDatabase"])
		case "redis":
			exports["URL"] = fmt.Sprintf("redis://u@%s:%s/%s", outputs["Port6379TcpAddr"], outputs["Port6379TcpPort"], outputs["EnvRedisDatabase"])
		case "webhook":
			if parsedUrl, err := url.Parse(parameters["Url"]); err == nil {
				exports["URL"] = parsedUrl.Query().Get("endpoint")
			}
		}
	}

	return &Service{
		Name:       cs(stack.StackName, "<unknown>"),
		Type:       tags["Service"],
		Status:     humanStatus(*stack.StackStatus),
		Outputs:    outputs,
		Parameters: parameters,
		Tags:       tags,
		Exports:    exports,
		// NOTE: this field is deprecated, use Exports instead
		URL: exports["URL"],
	}
}

// turns a dasherized map of key/value CLI params to
// parameters that CloudFormation expects
func CFParams(source map[string]string) map[string]string {
	params := make(map[string]string)

	for key, value := range source {
		var val string
		switch value {
		case "":
			val = "No"
		case "true":
			val = "Yes"
		default:
			val = value
		}
		params[AwsCamelize(key)] = val
	}

	return params
}

func AwsCamelize(dasherized string) string {
	tokens := strings.Split(dasherized, "-")

	for i, token := range tokens {
		switch token {
		case "az":
			tokens[i] = "AZ"
		default:
			tokens[i] = strings.Title(token)
		}
	}

	return strings.Join(tokens, "")
}
