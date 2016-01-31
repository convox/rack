package main

import "time"

var MONITOR_INTERVAL = 5 * time.Minute

func main() {
	monitor := NewMonitor()

	go monitor.Disk()
	go monitor.Dmesg()
	monitor.Listen()
}
