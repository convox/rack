package aws

import (
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/convox/rack/pkg/structs"
)

// EventSend publishes an important message out to the world.
//
// On AWS messages are published to SNS. The Rack has an HTTP endpoint that is an SNS
// subscription, and when a message is delivered forwards them to all configured
// webhook services.
//
// Often the Rack has a Console webhook which facilitates forwarding events
// to Slack with additional formatting and filtering.
//
// Because these are important system events, they are also published to Segment
// for operational metrics.

type event struct {
	Action    string            `json:"action"`
	Data      map[string]string `json:"data"`
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
}

func (p *Provider) EventSend(action string, opts structs.EventSendOptions) error {
	e := event{
		Action:    action,
		Data:      opts.Data,
		Status:    coalesces(opts.Status, "success"),
		Timestamp: time.Now().UTC(),
	}

	if e.Data["timestamp"] != "" {
		t, err := time.Parse(time.RFC3339, e.Data["timestamp"])
		if err == nil {
			e.Timestamp = t
		}
	}

	if opts.Error != "" {
		e.Status = "error"
		e.Data["message"] = opts.Error
	}

	e.Data["rack"] = p.Rack

	if p.IsTest() {
		e.Timestamp = time.Time{}
	}

	msg, err := json.Marshal(e)
	if err != nil {
		return err
	}

	// Publish Event to SNS
	_, err = p.sns().Publish(&sns.PublishInput{
		Message:   aws.String(string(msg)), // Required
		Subject:   aws.String(e.Action),
		TargetArn: aws.String(p.NotificationTopic),
	})
	if err != nil {
		return err
	}

	return nil
}
