package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func cs(s *string, def string) string {
	if s != nil {
		return *s
	} else {
		return def
	}
}

func ct(t *time.Time) time.Time {
	if t != nil {
		return *t
	} else {
		return time.Time{}
	}
}

func exists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}

	return true
}

func sendMixpanelEvent(event, message string) {
	if os.Getenv("DEVELOPMENT") == "Yes" {
		return // don't log dev events
	}
	id, err := currentId()

	if err != nil {
		// TODO log this error somewhere
		return
	}

	token := "43fb68427548c5e99978a598a9b14e55"

	m := fmt.Sprintf(`{"event": %q, "properties": {"client_id": %q, "distinct_id": %q, "message": %q, "token": %q}}`, event, id, id, message, token)
	encMessage := base64.StdEncoding.EncodeToString([]byte(m))

	_, err = http.Get(fmt.Sprintf("http://api.mixpanel.com/track/?data=%s", encMessage))

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
