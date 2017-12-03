package aws

func (p *AWSProvider) Workers() error {
	go p.workerAutoscale()
	go p.workerEvents()
	go p.workerHeartbeat()
	go p.workerMonitor()
	go p.workerSpotReplace()

	return nil
}
