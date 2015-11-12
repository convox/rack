package main

func main() {
	go MonitorDisk()

	monitor := NewMonitor()
	monitor.Listen()
}
