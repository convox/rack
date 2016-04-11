package models

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

func (s *Service) CreateSNS() (*cloudformation.CreateStackInput, error) {
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
	// e.g. `convox services create sns --queue=myqueue` needs to resolve to Queue=convox-myqueue
	if q, ok := s.Parameters["Queue"]; ok {
		s.Parameters["Queue"] = shortNameToStackName(q)
	}

	return req, nil
}
