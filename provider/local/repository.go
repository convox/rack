package local

import "fmt"

func (p *Provider) RepositoryAuth(app string) (string, string, error) {
	return "", "", nil
}

func (p *Provider) RepositoryHost(app string) (string, bool, error) {
	return fmt.Sprintf("%s/%s", p.Rack, app), false, nil
}
