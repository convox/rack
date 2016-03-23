package models

import (
	"fmt"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
)

func (s *Service) CreateS3() (*cloudformation.CreateStackInput, error) {
	var input interface{}

	formation, err := buildTemplate(fmt.Sprintf("service/%s", s.Type), "service", input)

	if err != nil {
		return nil, err
	}

	req := &cloudformation.CreateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(s.StackName()),
		TemplateBody: aws.String(formation),
	}

	// options reference another service by logical name, need to resolve to physical stack and resource name
	// e.g. `convox services create s3 --queue=myqueue` needs to resolve to Queue=convox-myqueue
	if q, ok := s.Parameters["Topic"]; ok {
		s.Parameters["Topic"] = shortNameToStackName(q)
	}

	return req, nil
}

// S3, SNS, SQS create an IAM user, therefore require IAM capabilitites for updates
func (s *Service) UpdateIAMService() (*cloudformation.UpdateStackInput, error) {
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
