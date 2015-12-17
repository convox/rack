package models

import (
	"os"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
)

func (s *Service) CreateDatastore() error {
	formation, err := s.Formation()

	if err != nil {
		return err
	}

	params := map[string]string{
		"Password": generateId("", 30),
		"Subnets":  os.Getenv("SUBNETS"),
		"Vpc":      os.Getenv("VPC"),
	}

	for key, value := range s.Options {
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

	tags := map[string]string{
		"System":  os.Getenv("RACK"),
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
