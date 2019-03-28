package local

func (p *Provider) AppIdles(name string) (bool, error) {
	return true, nil
}

func (p *Provider) AppStatus(name string) (string, error) {
	return "running", nil
}
