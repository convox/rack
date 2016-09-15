package aws

import (
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/api/structs"
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
func (p *AWSProvider) EventSend(e *structs.Event, err error) error {
	e.Status = "success"
	e.Timestamp = time.Now().UTC()

	if p.IsTest() {
		e.Timestamp = time.Time{}
	}

	if err != nil {
		e.Data["message"] = err.Error()
		e.Status = "error"
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

	// report event to Segment
	params := map[string]interface{}{
		"action": e.Action,
		"status": e.Status,
	}

	for k, v := range e.Data {
		params[k] = v
	}

	helpers.TrackEvent("event", params)

	return nil
}
