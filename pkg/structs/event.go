package structs

// type Event struct {
//   Action    string            `json:"action"` // app:create, release:create, release:promote, etc.
//   Status    string            `json:"status"` // success or error
//   Data      map[string]string `json:"data"`   // {"rack": "example-rack", "app": "example-app", "id": "R123456789", "message": "unable to load release"}
//   Timestamp time.Time         `json:"timestamp"`
// }

type EventSendOptions struct {
	Data   map[string]string `param:"data"`
	Error  string            `params:"error"`
	Status string            `params:"status"`
}
