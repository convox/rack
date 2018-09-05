package helpers

import (
	"time"
)

func Tick(d time.Duration, fn func()) {
	fn()

	for range time.Tick(d) {
		fn()
	}
}
