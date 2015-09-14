package main

import (
	"fmt"
	"time"

	"/github.com/ddollar/logger"
	"github.com/convox/kernel/helpers"
)

func recoverWith(f func(err error)) {
	if r := recover(); r != nil {
		// coerce r to error type
		err, ok := r.(error)
		if !ok {
			err = fmt.Errorf("%v", r)
		}

		f(err)
	}
}

func main() {

	go heartbeat()
	go startClusterMonitor()
	go pullAppImages()
	startWeb()
}

func heartbeat() {
	log := logger.New("ns=heartbeat")
	defer recoverWith(func(err error) {
		helpers.Error(log, err)
	})

	helpers.SendMixpanelEvent("kernel-heartbeat", "")

	for _ = range time.Tick(1 * time.Hour) {
		helpers.SendMixpanelEvent("kernel-heartbeat", "")
	}
}
