package helpers

import (
	"context"
	"fmt"
	"time"
)

func Wait(interval time.Duration, timeout time.Duration, times int, fn func() (bool, error)) error {
	return WaitContext(context.Background(), interval, timeout, times, fn)
}

func WaitContext(ctx context.Context, interval time.Duration, timeout time.Duration, times int, fn func() (bool, error)) error {
	successes := 0
	errors := 0
	start := time.Now().UTC()

	tick := time.NewTicker(interval)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
			if start.Add(timeout).Before(time.Now().UTC()) {
				return fmt.Errorf("timeout")
			}

			success, err := fn()
			if err != nil {
				errors += 1
			} else {
				errors = 0
			}

			if errors >= times {
				return err
			}

			if success {
				successes += 1
			} else {
				successes = 0
			}

			if successes >= times {
				return nil
			}
		}
	}
}
