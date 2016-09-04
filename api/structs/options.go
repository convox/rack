package structs

import "time"

type LogStreamOptions struct {
	Filter string
	Follow bool
	Since  time.Time
}
