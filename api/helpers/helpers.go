package helpers

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/segmentio/analytics-go"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/stvp/rollbar"
)

var regexpEmail = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
var segment *analytics.Client

func init() {
	rollbar.Token = os.Getenv("ROLLBAR_TOKEN")
	rollbar.Environment = os.Getenv("CLIENT_ID")

	segment = analytics.New(os.Getenv("SEGMENT_WRITE_KEY"))

	if os.Getenv("DEVELOPMENT") == "true" {
		segment.Size = 1
	}

	clientId := os.Getenv("CLIENT_ID")

	if regexpEmail.MatchString(clientId) && clientId != "ci@convox.com" {
		segment.Identify(&analytics.Identify{
			UserId: RackId(),
			Traits: map[string]interface{}{
				"email": clientId,
			},
		})
	}
}

func Error(log *logger.Logger, err error) {
	if log != nil {
		log.Error(err)
	}

	if rollbar.Token != "" {
		extraData := map[string]string{
			"AWS_REGION": os.Getenv("AWS_REGION"),
			"RACK":       os.Getenv("RACK"),
			"RELEASE":    os.Getenv("RELEASE"),
			"VPC":        os.Getenv("VPC"),
		}
		extraField := &rollbar.Field{"env", extraData}
		rollbar.Error(rollbar.ERR, err, extraField)
	}
}

func TrackEvent(event string, params map[string]interface{}) {
	log := logrus.WithFields(logrus.Fields{"ns": "api.helpers", "at": "TrackEvent"})

	if params == nil {
		params = map[string]interface{}{}
	}

	params["client_id"] = os.Getenv("CLIENT_ID")

	userId := RackId()

	log.WithFields(logrus.Fields{"event": event, "user_id": userId}).WithFields(logrus.Fields(params)).Info()

	segment.Track(&analytics.Track{
		Event:      event,
		UserId:     userId,
		Properties: params,
	})
}

// Convenience function to track success in a controller handler
// See also httperr.TrackErrorf and httperr.TrackServer
func TrackSuccess(event string, params map[string]interface{}) {
	params["status"] = "success"

	TrackEvent(event, params)
}

func TrackError(event string, err error, params map[string]interface{}) {
	params["status"] = "error"
	params["error"] = fmt.Sprintf("%v", err)

	TrackEvent(event, params)
}

func RackId() string {
	if stackId := os.Getenv("STACK_ID"); stackId != "" {
		parts := strings.Split(stackId, "/")
		return parts[len(parts)-1]
	}

	return os.Getenv("CLIENT_ID")
}
