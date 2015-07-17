package main

import (
	"time"

	"github.com/convox/kernel/helpers"
)

func main() {

	go heartbeat()
	go startClusterMonitor()
	startWeb()
}

func heartbeat() {
	helpers.SendMixpanelEvent("kernel-heartbeat")

	for _ = range time.Tick(1 * time.Hour) {
		helpers.SendMixpanelEvent("kernel-heartbeat")
	}
}
