package local

func (p *Provider) DockerSocket() string {
	return "/var/snap/microk8s/current/docker.sock"
}
