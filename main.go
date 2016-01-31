package main

import "time"

var MONITOR_INTERVAL = 5 * time.Second

func main() {
	monitor := NewMonitor()

	go monitor.Containers()
	go monitor.Disk()
	go monitor.Dmesg()

	for {
		time.Sleep(60 * time.Second)
	}
}
