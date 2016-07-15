package models

import (

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

func (s *Service) UpdateLogger() (*cloudformation.UpdateStackInput, error) {
	formation, err := s.Formation()
	if err != nil {
		return nil, err
	}

  req := &cloudformation.UpdateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(s.StackName()),
		TemplateBody: aws.String(formation),
	}

	return req, nil
}
