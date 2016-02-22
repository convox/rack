package aws

import (
	"encoding/json"
	"os"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/sns"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/convox/logger"
	"github.com/convox/rack/api/structs"
)

var (
	NotificationTopic = os.Getenv("NOTIFICATION_TOPIC")
)

func (p *AWSProvider) NotifySuccess(action string, data map[string]string) error {
	return p.notify(action, "success", data)
}

func (p *AWSProvider) NotifyError(action string, err error, data map[string]string) error {
	data["message"] = err.Error()

	return p.notify(action, "error", data)
}

/** helpers ****************************************************************************************/

func (p *AWSProvider) notify(action, status string, data map[string]string) error {
	log := logger.New("ns=kernel")

	data["rack"] = os.Getenv("RACK")

	event := &structs.Notification{
		Action:    action,
		Status:    status,
		Data:      data,
		Timestamp: time.Now().UTC(),
	}

	message, err := json.Marshal(event)

	if err != nil {
		return err
	}

	res, err := p.sns().Publish(&sns.PublishInput{
		Message:   aws.String(string(message)), // Required
		Subject:   aws.String(action),
		TargetArn: aws.String(NotificationTopic),
	})

	if err != nil {
		return err
	}

	log.At("notify").Log("message-id=%q", *res.MessageId)

	return nil
}
