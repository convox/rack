package local

func (p *Provider) AppStatus(name string) (string, error) {
	return "running", nil
}
