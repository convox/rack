package local

import "fmt"

func (p *Provider) RepositoryAuth(app string) (string, string, error) {
	return "docker", "secret", nil
}

func (p *Provider) RepositoryHost(app string) (string, bool, error) {
	return fmt.Sprintf("registry.%s/%s", p.Rack, app), true, nil
}
