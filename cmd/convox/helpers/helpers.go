package helpers

import (
	"os"
	"time"

	"github.com/dustin/go-humanize"
)

func Exists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}

	return true
}

func HumanizeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	} else {
		return humanize.Time(t)
	}
}
