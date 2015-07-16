package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
)

func exists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}

	return true
}

func sendMixpanelEvent(event string, distinctId string) {
	token := "43fb68427548c5e99978a598a9b14e55"

	message := fmt.Sprintf(`{"event": %q, "properties": {"distinct_id": %q, "token": %q}}`, event, distinctId, token)
	encMessage := base64.StdEncoding.EncodeToString([]byte(message))

	_, err := http.Get(fmt.Sprintf("http://api.mixpanel.com/track/?data=%s", encMessage))

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}
}
