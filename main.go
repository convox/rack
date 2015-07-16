package main

import (
	b64 "encoding/base64"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/convox/kernel/helpers"
)

func main() {

	go heartbeat()
	go startClusterMonitor()
	startWeb()
}

func heartbeat() {
	for _ = range time.Tick(1 * time.Hour) {
		message := fmt.Sprintf(`{"event": "kernel-heartbeat", "properties": {"aws_accountid": %[1], "distinct_id": %[1], "token": %[2]}}`, os.Getenv("AWS_ACCOUNTID"), os.Getenv("MIXPANEL_TOKEN"))
		encMessage := b64.StdEncoding.EncodeToString([]byte(message))
		_, err := http.Get(fmt.Sprintf("http://api.mixpanel.com/track/?data=%s", encMessage))
		if err != nil {
			helpers.Error(nil, err)
		}
	}
}
