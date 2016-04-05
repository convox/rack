package structs

import "time"

type Notification struct {
	Event      string                 `json:"event"`      // "release"
	Step       string                 `json:"step"`       // "create", "promote", etc.
	State      string                 `json:"state"`      // "success" or "error"
	Error      error                  `json:"error"`      // errors.New("Send me to rollbar")
	Properties map[string]interface{} `json:"properties"` // {"client_id": "foo", "elapsed": 12}
	Timestamp  time.Time              `json:"timestamp"`
}
