// +build !darwin,!linux

package changes

import "time"

func startScanner(dir string) {
}

func waitForNextScan(dir string) {
	time.Sleep(700 * time.Millisecond)
}
