package helpers

import (
	"fmt"
	"time"

	humanize "github.com/dustin/go-humanize"
)

var (
	PrintableTime = "2006-01-02 15:04:05"
	SortableTime  = "20060102.150405.000000000"
)

func Ago(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return humanize.Time(t)
}

func Duration(start, end time.Time) string {
	d := end.Sub(start)

	if end.IsZero() {
		return ""
	}

	total := int64(d.Seconds())
	sec := total % 60
	min := total / 60

	dur := fmt.Sprintf("%ds", sec)

	if min >= 1 {
		dur = fmt.Sprintf("%dm", min) + dur
	}

	return dur
}
