package provider

import "github.com/convox/rack/api/structs"

type TestProvider struct {
}

func (p TestProvider) InstanceList() (structs.Instances, error) {
	return nil, nil
}

func (p TestProvider) SystemGet() (*structs.System, error) {
	return nil, nil
}

func (p TestProvider) SystemSave(system structs.System) error {
	return nil
}
