package main

import (
	b64 "encoding/base64"
	"fmt"
	"github.com/convox/kernel/helpers"
	"net/http"
	"os"
	"time"
)

func main() {

	go heartbeat()
	go startWorker()
	startWeb()
}

func heartbeat() {
	for _ = range time.Tick(1 * time.Hour) {
		message := fmt.Sprintf(`{"event": "kernel-heartbeat", "properties": {"aws_accountid": %q, "token": %q}}`, os.Getenv("AWS_ACCOUNTID"), os.Getenv("MIXPANEL_TOKEN"))
		encMessage := b64.StdEncoding.EncodeToString([]byte(message))
		_, err := http.Get(fmt.Sprintf("http://api.mixpanel.com/track/?data=%s", encMessage))
		if err != nil {
			helpers.Error(nil, err)
		}
	}
}
