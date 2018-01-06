package structs

import "time"

type LogsOptions struct {
	Filter string
	Follow bool
	Prefix bool
	Since  time.Time
}
