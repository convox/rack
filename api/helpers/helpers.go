package helpers

import (
	"os"
	"regexp"

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

	if regexpEmail.MatchString(os.Getenv("CLIENT_ID")) {
		segment.Identify(&analytics.Identify{
			UserId: os.Getenv("CLIENT_ID"),
			Traits: map[string]interface{}{
				"email": os.Getenv("CLIENT_ID"),
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

func TrackEvent(event, message string) {
	segment.Track(&analytics.Track{
		Event:  event,
		UserId: os.Getenv("CLIENT_ID"),
		Properties: map[string]interface{}{
			"message": message,
		},
	})
}
