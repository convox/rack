package provider

import (
	"github.com/convox/rack/composure/structs"
	"github.com/stretchr/testify/mock"
)

var TestProvider = &TestProviderRunner{}

type TestProviderRunner struct {
	mock.Mock
	Manifest structs.Manifest
}

func (p *TestProviderRunner) Load(path string) (*structs.Manifest, error) {
	p.Called(path)
	return &p.Manifest, nil
}

func (p *TestProviderRunner) Pull(m *structs.Manifest) error {
	p.Called(m)
	return nil
}
