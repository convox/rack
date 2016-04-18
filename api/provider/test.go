package provider

import (
	"io"

	"github.com/convox/rack/api/structs"
	"github.com/stretchr/testify/mock"
)

var TestProvider = &TestProviderRunner{}

type TestProviderRunner struct {
	mock.Mock
	App       structs.App
	Build     structs.Build
	Builds    structs.Builds
	Capacity  structs.Capacity
	Instances structs.Instances
	Release   structs.Release
	Releases  structs.Releases
	Service   structs.Service
}

func (p *TestProviderRunner) AppGet(name string) (*structs.App, error) {
	p.Called(name)
	return &p.App, nil
}

func (p *TestProviderRunner) BuildCopy(srcApp, id, destApp string) (*structs.Build, error) {
	p.Called(srcApp, id, destApp)
	return &p.Build, nil
}

func (p *TestProviderRunner) BuildCreateIndex(app string, index structs.Index, manifest, description string, cache bool) (*structs.Build, error) {
	p.Called(app, index, manifest, description, cache)
	return &p.Build, nil
}

func (p *TestProviderRunner) BuildCreateRepo(app, url, manifest, description string, cache bool) (*structs.Build, error) {
	p.Called(app, url, manifest, description, cache)
	return &p.Build, nil
}

func (p *TestProviderRunner) BuildCreateTar(app string, src io.Reader, manifest, description string, cache bool) (*structs.Build, error) {
	p.Called(app, src, manifest, description, cache)
	return &p.Build, nil
}

func (p *TestProviderRunner) BuildDelete(app, id string) (*structs.Build, error) {
	p.Called(app, id)
	return &p.Build, nil
}

func (p *TestProviderRunner) BuildGet(app, id string) (*structs.Build, error) {
	p.Called(app, id)
	return &p.Build, nil
}

func (p *TestProviderRunner) BuildList(app string) (structs.Builds, error) {
	p.Called(app)
	return p.Builds, nil
}

func (p *TestProviderRunner) BuildRelease(b *structs.Build) (*structs.Release, error) {
	p.Called(b)
	return &p.Release, nil
}

func (p *TestProviderRunner) BuildSave(b *structs.Build) error {
	p.Called(b)
	return nil
}

func (p *TestProviderRunner) CapacityGet() (*structs.Capacity, error) {
	p.Called()
	return &p.Capacity, nil
}

func (p *TestProviderRunner) EventSend(e *structs.Event, err error) error {
	p.Called(e, err)
	return nil
}

func (p *TestProviderRunner) IndexDiff(i *structs.Index) ([]string, error) {
	p.Called(i)
	return []string{}, nil
}

func (p *TestProviderRunner) IndexDownload(i *structs.Index, dir string) error {
	p.Called(i, dir)
	return nil
}

func (p *TestProviderRunner) IndexUpload(hash string, data []byte) error {
	p.Called(hash, data)
	return nil
}

func (p *TestProviderRunner) InstanceList() (structs.Instances, error) {
	p.Called()
	return p.Instances, nil
}

func (p *TestProviderRunner) LogStream(app string, w io.Writer, opts structs.LogStreamOptions) error {
	p.Called(app, w, opts)
	return nil
}

func (p *TestProviderRunner) ReleaseDelete(app, id string) (*structs.Release, error) {
	p.Called(app, id)
	return &p.Release, nil
}

func (p *TestProviderRunner) ReleaseGet(app, id string) (*structs.Release, error) {
	p.Called(app, id)
	return &p.Release, nil
}

func (p *TestProviderRunner) ReleaseList(app string) (structs.Releases, error) {
	p.Called(app)
	return p.Releases, nil
}

func (p *TestProviderRunner) ReleasePromote(app, id string) (*structs.Release, error) {
	p.Called(app, id)
	return &p.Release, nil
}

func (p *TestProviderRunner) ReleaseSave(r *structs.Release, logdir, key string) error {
	p.Called(r, logdir, key)
	return nil
}

func (p *TestProviderRunner) ServiceCreate(name, kind string, params map[string]string) (*structs.Service, error) {
	p.Called(name, kind, params)
	return &p.Service, nil
}

func (p *TestProviderRunner) SystemGet() (*structs.System, error) {
	p.Called()
	return nil, nil
}

func (p *TestProviderRunner) SystemSave(system structs.System) error {
	p.Called(system)
	return nil
}
