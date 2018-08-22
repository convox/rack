package helpers

import (
	"math/rand"
	"time"
)

func Retry(times int, interval time.Duration, fn func() error) error {
	i := 0

	for {
		err := fn()
		if err == nil {
			return nil
		}

		// add 20% jitter
		time.Sleep(interval + time.Duration(rand.Intn(int(interval/20))))

		i++

		if i > times {
			return err
		}
	}
}
