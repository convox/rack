package stdcli

import (
	"fmt"
	"time"
)

func Duration(start, end time.Time) string {
	d := end.Sub(start)

	total := int64(d.Seconds())
	sec := total % 60
	min := total / 60

	dur := fmt.Sprintf("%ds", sec)

	if min >= 1 {
		dur = fmt.Sprintf("%dm", min) + dur
	}

	return dur
}
