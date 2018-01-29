package helpers

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/convox/logger"
	"github.com/segmentio/analytics-go"
	"github.com/stvp/rollbar"
)

var regexpEmail = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
var segment *analytics.Client

func init() {
	rollbar.Token = os.Getenv("ROLLBAR_TOKEN")
	rollbar.Environment = os.Getenv("CLIENT_ID")

	segment = analytics.New(os.Getenv("SEGMENT_WRITE_KEY"))
	segment.Size = 1

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
		log.ErrorBacktrace(err)
	}

	if rollbar.Token != "" {
		extraData := map[string]string{
			"AWS_REGION": os.Getenv("AWS_REGION"),
			"CLIENT_ID":  os.Getenv("CLIENT_ID"),
			"RACK":       os.Getenv("RACK"),
			"RELEASE":    os.Getenv("RELEASE"),
			"VPC":        os.Getenv("VPC"),
		}
		extraField := &rollbar.Field{"env", extraData}
		rollbar.Error(rollbar.ERR, err, extraField)
	}
}

func TrackEvent(event string, params map[string]interface{}) {
	if params == nil {
		params = map[string]interface{}{}
	}

	params["client_id"] = os.Getenv("CLIENT_ID")
	params["rack"] = os.Getenv("RACK")
	params["release"] = os.Getenv("RELEASE")

	userId := RackId()

	segment.Track(&analytics.Track{
		Event:      event,
		UserId:     userId,
		Properties: params,
	})
}

// Convenience function to track success in a controller handler
// See also httperr.TrackErrorf and httperr.TrackServer
func TrackSuccess(event string, params map[string]interface{}) {
	params["state"] = "success"

	TrackEvent(event, params)
}

func TrackError(event string, err error, params map[string]interface{}) {
	params["error"] = fmt.Sprintf("%v", err)
	params["state"] = "error"

	TrackEvent(event, params)
}

func RackId() string {
	if stackId := os.Getenv("STACK_ID"); stackId != "" {
		parts := strings.Split(stackId, "/")
		return parts[len(parts)-1]
	}

	return os.Getenv("CLIENT_ID")
}
