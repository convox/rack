package structs

import "time"

type Notification struct {
	Action    string            `json:"action"`
	Status    string            `json:"status"`
	Data      map[string]string `json:"data"`
	Timestamp time.Time         `json:"timestamp"`
}
