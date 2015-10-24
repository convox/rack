package client

import "time"

//a NotifyEvent is the payload of any webhook services
//it is serialized to json
type NotifyEvent struct {
	Action    string            `json:"action"`
	Status    string            `json:"status"`
	Data      map[string]string `json:"data"`
	Timestamp time.Time         `json:"timestamp"`
}
