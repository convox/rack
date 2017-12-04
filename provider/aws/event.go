package aws

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/convox/rack/helpers"
	"github.com/convox/rack/structs"
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
	e.Timestamp = time.Now().UTC()

	if e.Data["timestamp"] != "" {
		t, err := time.Parse(time.RFC3339, e.Data["timestamp"])
		if err == nil {
			e.Timestamp = t
		}
	}

	if e.Status == "" {
		e.Status = "success"
	}

	if e.Data == nil {
		e.Data = map[string]string{}
	}
	e.Data["rack"] = p.Rack

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

	sendSegmentEvent(e)

	return nil
}

// sendSegmentEvent reports an event to Segment
func sendSegmentEvent(e *structs.Event) {
	action := strings.Split(e.Action, ":")

	obj := strings.Title(action[0])
	act := strings.Title(action[1])
	se := "unkown segment event"

	pst := map[string]string{
		"Create":  "Created",
		"Promote": "Promoted",
		"Delete":  "Deleted",
	}

	switch e.Status {
	case "start":
		se = fmt.Sprintf("%s %s %s", obj, act, "Started")
	case "error":
		se = fmt.Sprintf("%s %s %s", obj, act, "Failed")
	case "success":
		se = fmt.Sprintf("%s %s", obj, pst[act])
	}

	params := map[string]interface{}{}

	for k, v := range e.Data {
		params[k] = v
	}

	helpers.TrackEvent(se, params)
}
