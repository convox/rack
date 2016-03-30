package provider

import "github.com/convox/rack/api/structs"

var TestProvider = &TestProviderRunner{}

type TestProviderRunner struct {
	App       structs.App
	Capacity  structs.Capacity
	Instances structs.Instances
	Release   structs.Release
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

func (p *TestProviderRunner) ImageDelete(urls []string) error {
	return nil
}

func (p *TestProviderRunner) ReleaseGet(app, id string) (*structs.Release, error) {
	return &p.Release, nil
}

func (p *TestProviderRunner) SystemGet() (*structs.System, error) {
	return nil, nil
}

func (p *TestProviderRunner) SystemSave(system structs.System) error {
	return nil
}
