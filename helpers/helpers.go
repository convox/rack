package helpers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/stvp/rollbar"
)

func init() {
	rollbar.Token = os.Getenv("ROLLBAR_TOKEN")
}

func Error(log *logger.Logger, err error) {
	if log == nil {
		log.Error(err)
	}
	rollbar.Error(rollbar.ERR, err)
}

func SendMixpanelEvent(event string) {
	id := os.Getenv("AWS_ACCOUNTID")
	token := os.Getenv("MIXPANEL_TOKEN")

	if os.Getenv("MIXPANEL_DEV") == "true" {
		id = "dev"
	}

	message := fmt.Sprintf(`{"event": %q, "properties": {"aws_accountid": %q, "distinct_id": %q, "token": %q}}`, event, id, id, token)
	encMessage := base64.StdEncoding.EncodeToString([]byte(message))

	_, err := http.Get(fmt.Sprintf("http://api.mixpanel.com/track/?data=%s", encMessage))

	if err != nil {
		Error(nil, err)
	}
}
