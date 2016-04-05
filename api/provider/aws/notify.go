package aws

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/sns"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/client"
)

// Notify publishes an important message out to the world.
//
// On AWS messages are published to SNS. The Rack has an HTTP endpoint that is an SNS
// subscription, and when a message is delivered forwards them to all configured
// webhook services.
//
// Often the Rack has a Console webhook which facilitates forwarding notifications
// to Slack with additional formatting and filtering.
//
// Because these are important system events all notifications are also published
// to Segment and error notifications are published to Rollbar.
func (p *AWSProvider) Notify(n *structs.Notification) error {
	log := logger.New("ns=kernel")

	// convert Notification to "legacy" console NotifyEvent
	data := map[string]string{
		"rack": os.Getenv("RACK"),
	}

	if n.Error != nil {
		data["message"] = n.Error.Error()
	}

	for k, v := range n.Properties {
		data[k] = fmt.Sprintf("%v", v)
	}

	event := &client.NotifyEvent{
		Action:    fmt.Sprintf("%s:%s", n.Event, n.Step),
		Data:      data,
		Status:    n.State,
		Timestamp: time.Now().UTC(),
	}

	message, err := json.Marshal(event)
	if err != nil {
		helpers.Error(log, err) // report internal errors to Rollbar
		return err
	}

	// Publish NotifyEvent to SNS
	params := &sns.PublishInput{
		Message:   aws.String(string(message)), // Required
		Subject:   aws.String(event.Action),
		TargetArn: aws.String(os.Getenv("NOTIFICATION_TOPIC")),
	}
	resp, err := p.sns().Publish(params)
	if err != nil {
		helpers.Error(log, err) // report internal errors to Rollbar
		return err
	}

	log.At("Notify").Log("message-id=%q", *resp.MessageId)

	// report event to Segment and Rollbar
	if n.Error != nil {
		helpers.Error(log, n.Error)
		helpers.TrackError(n.Event, n.Error, n.Properties)
	} else {
		helpers.TrackSuccess(n.Event, n.Properties)
	}

	return nil
}
