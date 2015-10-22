package models

import (
	"fmt"
	"os"

	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
)

var NotificationTopic = os.Getenv("NOTIFICATION_TOPIC")

func (s *Service) CreateWebhook() error {
	if s.URL == "" {
		return fmt.Errorf("Webhook URL is required")
	}

	var input interface{}
	formation, err := buildTemplate("service/webhook", "service", input)

	if err != nil {
		return err
	}

	params := map[string]string{
		"Url":               s.URL,
		"NotificationTopic": NotificationTopic,
		"CustomTopic":       CustomTopic,
	}

	tags := map[string]string{
		"System":  "convox",
		"Type":    "service",
		"Service": "webhook",
	}

	req := &cloudformation.CreateStackInput{
		//TODO: do i need this?
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
