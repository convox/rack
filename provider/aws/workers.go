package aws

func (p *AWSProvider) Workers() error {
	go p.workerAgent()
	go p.workerAutoscale()
	go p.workerCleanup()
	go p.workerEvents()
	go p.workerHeartbeat()
	go p.workerMonitor()
	go p.workerSpotReplace()

	return nil
}
