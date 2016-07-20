package models

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

// UpdateLogger needs an IAM role to update the lambda, therefore requires IAM capabilitites for updates
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
