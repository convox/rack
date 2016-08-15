package provider

import (
	"io"

	"github.com/convox/rack/api/structs"
	"github.com/stretchr/testify/mock"
)

type TestProvider struct {
	mock.Mock
	App          structs.App
	Build        structs.Build
	Builds       structs.Builds
	Capacity     structs.Capacity
	Certificate  structs.Certificate
	Certificates structs.Certificates
	Instances    structs.Instances
	Release      structs.Release
	Releases     structs.Releases
	Service      structs.Service
}

func (p *TestProvider) AppGet(name string) (*structs.App, error) {
	p.Called(name)
	return &p.App, nil
}

// AppDelete deletes an app
func (p *TestProvider) AppDelete(name string) error {
	p.Called(name)
	return nil
}

func (p *TestProvider) BuildCopy(srcApp, id, destApp string) (*structs.Build, error) {
	p.Called(srcApp, id, destApp)
	return &p.Build, nil
}

func (p *TestProvider) BuildCreateIndex(app string, index structs.Index, manifest, description string, cache bool) (*structs.Build, error) {
	p.Called(app, index, manifest, description, cache)
	return &p.Build, nil
}

func (p *TestProvider) BuildCreateRepo(app, url, manifest, description string, cache bool) (*structs.Build, error) {
	p.Called(app, url, manifest, description, cache)
	return &p.Build, nil
}

func (p *TestProvider) BuildCreateTar(app string, src io.Reader, manifest, description string, cache bool) (*structs.Build, error) {
	p.Called(app, src, manifest, description, cache)
	return &p.Build, nil
}

func (p *TestProvider) BuildDelete(app, id string) (*structs.Build, error) {
	p.Called(app, id)
	return &p.Build, nil
}

func (p *TestProvider) BuildGet(app, id string) (*structs.Build, error) {
	p.Called(app, id)
	return &p.Build, nil
}

// BuildList returns a mock list of the latest builds
func (p *TestProvider) BuildList(app string, limit int64) (structs.Builds, error) {
	p.Called(app, limit)
	return p.Builds, nil
}

func (p *TestProvider) BuildRelease(b *structs.Build) (*structs.Release, error) {
	p.Called(b)
	return &p.Release, nil
}

func (p *TestProvider) BuildSave(b *structs.Build) error {
	p.Called(b)
	return nil
}

func (p *TestProvider) CapacityGet() (*structs.Capacity, error) {
	p.Called()
	return &p.Capacity, nil
}

func (p *TestProvider) CertificateCreate(pub, key, chain string) (*structs.Certificate, error) {
	p.Called(pub, key, chain)
	return &p.Certificate, nil
}

func (p *TestProvider) CertificateDelete(id string) error {
	p.Called(id)
	return nil
}

func (p *TestProvider) CertificateGenerate(domains []string) (*structs.Certificate, error) {
	p.Called(domains)
	return &p.Certificate, nil
}

func (p *TestProvider) CertificateList() (structs.Certificates, error) {
	p.Called()
	return p.Certificates, nil
}

func (p *TestProvider) EventSend(e *structs.Event, err error) error {
	p.Called(e, err)
	return nil
}

func (p *TestProvider) EnvironmentGet(app string) (structs.Environment, error) {
	p.Called()
	return nil, nil
}

func (p *TestProvider) IndexDiff(i *structs.Index) ([]string, error) {
	p.Called(i)
	return []string{}, nil
}

func (p *TestProvider) IndexDownload(i *structs.Index, dir string) error {
	p.Called(i, dir)
	return nil
}

func (p *TestProvider) IndexUpload(hash string, data []byte) error {
	p.Called(hash, data)
	return nil
}

func (p *TestProvider) InstanceList() (structs.Instances, error) {
	p.Called()
	return p.Instances, nil
}

func (p *TestProvider) LogStream(app string, w io.Writer, opts structs.LogStreamOptions) error {
	p.Called(app, w, opts)
	return nil
}

// ReleaseDelete deletes releases associated with app and buildID in batches
func (p *TestProvider) ReleaseDelete(app, buildID string) error {
	p.Called(app, buildID)
	return nil
}

func (p *TestProvider) ReleaseGet(app, id string) (*structs.Release, error) {
	p.Called(app, id)
	return &p.Release, nil
}

// ReleaseList returns a list of releases
func (p *TestProvider) ReleaseList(app string, limit int64) (structs.Releases, error) {
	args := p.Called(app, limit)
	return args.Get(0).(structs.Releases), args.Error(1)
}

func (p *TestProvider) ReleasePromote(app, id string) (*structs.Release, error) {
	p.Called(app, id)
	return &p.Release, nil
}

func (p *TestProvider) ReleaseSave(r *structs.Release, logdir, key string) error {
	p.Called(r, logdir, key)
	return nil
}

func (p *TestProvider) ServiceCreate(name, kind string, params map[string]string) (*structs.Service, error) {
	p.Called(name, kind, params)
	return &p.Service, nil
}

func (p *TestProvider) ServiceDelete(name string) (*structs.Service, error) {
	p.Called(name)
	return &p.Service, nil
}

func (p *TestProvider) ServiceGet(name string) (*structs.Service, error) {
	p.Called(name)
	return &p.Service, nil
}

func (p *TestProvider) ServiceLink(name, app, process string) (*structs.Service, error) {
	p.Called(name, app, process)
	return &p.Service, nil
}

func (p *TestProvider) ServiceUnlink(name, app, process string) (*structs.Service, error) {
	p.Called(name, app, process)
	return &p.Service, nil
}

func (p *TestProvider) SystemGet() (*structs.System, error) {
	p.Called()
	return nil, nil
}

func (p *TestProvider) SystemSave(system structs.System) error {
	p.Called(system)
	return nil
}
