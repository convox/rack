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

	if s.Type != "redis" {
		s.Parameters["Password"] = generateId("", 30)
	}

	s.Parameters["Subnets"] = os.Getenv("SUBNETS")
	s.Parameters["Vpc"] = os.Getenv("VPC")

	req := &cloudformation.CreateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(s.StackName()),
		TemplateBody: aws.String(formation),
	}

	return req, err
}
