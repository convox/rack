package models

import (
	"os"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
)

func (s *Service) CreateDatastore() (*cloudformation.CreateStackInput, error) {
	formation, err := s.Formation()

	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"Password": generateId("", 30),
		"Subnets":  os.Getenv("SUBNETS"),
		"Vpc":      os.Getenv("VPC"),
	}

	req := &cloudformation.CreateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(s.Name),
		TemplateBody: aws.String(formation),
	}

	for key, value := range params {
		req.Parameters = append(req.Parameters, &cloudformation.Parameter{ParameterKey: aws.String(key), ParameterValue: aws.String(value)})
	}

	return req, err
}
