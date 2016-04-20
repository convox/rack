package structs

import "time"

type LogStreamOptions struct {
	Filter string        `json:"filter"`
	Follow bool          `json:"follow"`
	Since  time.Duration `json:"since"`
}
