package provider

import "github.com/convox/rack/api/structs"

var TestProvider = &TestProviderRunner{}

type TestProviderRunner struct {
	App       structs.App
	Instances structs.Instances
	Capacity  structs.Capacity
}

func (p *TestProviderRunner) AppGet(name string) (*structs.App, error) {
	return &p.App, nil
}

func (p *TestProviderRunner) CapacityGet() (*structs.Capacity, error) {
	return &p.Capacity, nil
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
