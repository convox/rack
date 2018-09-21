package aws

func (p *Provider) Workers() error {
	go p.workerCleanup()
	go p.workerEvents()
	go p.workerHeartbeat()
	go p.workerMonitor()
	go p.workerSpotReplace()

	return nil
}
