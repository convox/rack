package provider

import "github.com/convox/rack/api/structs"

var TestProvider = &TestProviderRunner{}

type TestProviderRunner struct {
	Instances structs.Instances
}

func (p *TestProviderRunner) InstanceList() (structs.Instances, error) {
	return p.Instances, nil
}

func (p *TestProviderRunner) SystemGet() (*structs.System, error) {
	return nil, nil
}

func (p *TestProviderRunner) SystemSave(system structs.System) error {
	return nil
}
