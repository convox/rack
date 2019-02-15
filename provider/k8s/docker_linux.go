package k8s

func (p *Provider) dockerSocket() string {
	return "/var/snap/microk8s/current/docker.sock"
}
