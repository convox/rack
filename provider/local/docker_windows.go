package local

func (p *Provider) DockerSocket() string {
	return "/var/run/docker.sock"
}
