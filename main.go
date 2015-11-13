package main

func main() {
	go MonitorDisk()
	go MonitorDmesg()

	monitor := NewMonitor()
	monitor.Listen()
}
