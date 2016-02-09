package models

import (
	"net/url"
	"os"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
)

var NotificationTopic = os.Getenv("NOTIFICATION_TOPIC")
var NotificationHost = os.Getenv("NOTIFICATION_HOST")

func (s *Service) CreateWebhook() (*cloudformation.CreateStackInput, error) {
	var input interface{}
	formation, err := buildTemplate("service/webhook", "service", input)

	if err != nil {
		return nil, err
	}

	if s.Parameters["Url"] != "" {
		//ensure valid URL
		_, err = url.Parse(s.Parameters["Url"])
		if err != nil {
			return nil, err
		}

		encEndpoint := url.QueryEscape(s.Parameters["Url"])
		//NOTE: using http SNS notifier because https
		//      doesn't work with rack's self-signed cert
		proxyEndpoint := "http://" + NotificationHost + "/sns?endpoint=" + encEndpoint
		s.Parameters["Url"] = proxyEndpoint
	}

	s.Parameters["NotificationTopic"] = NotificationTopic
	s.Parameters["CustomTopic"] = CustomTopic

	req := &cloudformation.CreateStackInput{
		//TODO: do i need this?
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(s.StackName()),
		TemplateBody: aws.String(formation),
	}

	return req, nil
}
