package models

import (
	"fmt"
	"net/url"
	"os"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
)

var NotificationTopic = os.Getenv("NOTIFICATION_TOPIC")
var NotificationHost = os.Getenv("NOTIFICATION_HOST")

func (s *Service) CreateWebhook() (*cloudformation.CreateStackInput, error) {
	if s.Options["url"] == "" {
		return nil, fmt.Errorf("Webhook URL is required")
	}

	//ensure valid URL
	_, err := url.Parse(s.Options["url"])
	if err != nil {
		return nil, err
	}

	var input interface{}
	formation, err := buildTemplate("service/webhook", "service", input)

	if err != nil {
		return nil, err
	}

	encEndpoint := url.QueryEscape(s.Options["url"])
	//NOTE always assumes https instead of u.Scheme
	proxyEndpoint := "http://" + NotificationHost + "/sns?endpoint=" + encEndpoint

	params := map[string]string{
		"Url":               proxyEndpoint,
		"NotificationTopic": NotificationTopic,
		"CustomTopic":       CustomTopic,
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

	return req, nil
}
