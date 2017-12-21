package options

import "time"

func Int(value int) *int {
	v := value
	return &v
}

func String(value string) *string {
	v := value
	return &v
}

func Time(value time.Time) *time.Time {
	v := value
	return &v
}
