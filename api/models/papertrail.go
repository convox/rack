package models

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/kinesis"
)

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
	// build map of app name -> arn of all linked services
	arns := map[string]string{}

	for k, v := range s.Outputs {
		if strings.HasSuffix(k, "Link") {
			n := DashName(k)
			arns[n[:len(n)-5]] = v
		}
	}

	// get full Kinesis ARN for app
	req, err := Kinesis().DescribeStream(&kinesis.DescribeStreamInput{
		StreamName: aws.String(app.Outputs["Kinesis"]),
	})

	arn := *req.StreamDescription.StreamARN

	if err != nil {
		return err
	}

	// append new ARN and update
	arns[app.Name] = arn
	return s.UpdatePapertrail(arns)
}

func (s *Service) UnlinkPapertrail(app App) error {
	// build map of app name -> arn of all linked services
	arns := map[string]string{}

	for k, v := range s.Outputs {
		if strings.HasSuffix(k, "Link") {
			n := DashName(k)
			arns[n[:len(n)-5]] = v
		}
	}

	// delete existing ARN and update
	delete(arns, app.Name)
	return s.UpdatePapertrail(arns)
}

func (s *Service) UpdatePapertrail(arns map[string]string) error {
	input := struct {
		ARNs map[string]string
	}{
		arns,
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
				ParameterKey:   aws.String("Url"),
				ParameterValue: aws.String(s.Parameters["Url"]),
			},
		},
		TemplateBody: aws.String(formation),
	})

	return err
}
