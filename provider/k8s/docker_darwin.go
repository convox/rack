package k8s

func (p *Provider) dockerSocket() string {
	return "/var/run/docker.sock"
}
