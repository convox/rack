package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
)

func exists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}

	return true
}

func sendMixpanelEvent(event string, id string) {
	token := "43fb68427548c5e99978a598a9b14e55"

	if Version == "dev" {
		id = "dev"
	}

	message := fmt.Sprintf(`{"event": %q, "properties": {"aws_accountid": %q, "distinct_id": %q, "token": %q}}`, event, id, id, token)
	encMessage := base64.StdEncoding.EncodeToString([]byte(message))

	_, err := http.Get(fmt.Sprintf("http://api.mixpanel.com/track/?data=%s", encMessage))

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}
}

func upperName(name string) string {
	us := strings.ToUpper(name[0:1]) + name[1:]

	for {
		i := strings.Index(us, "-")

		if i == -1 {
			break
		}

		s := us[0:i]

		if len(us) > i+1 {
			s += strings.ToUpper(us[i+1 : i+2])
		}

		if len(us) > i+2 {
			s += us[i+2:]
		}

		us = s
	}

	return us
}
