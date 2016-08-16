package provider

import (
	"io"

	"github.com/convox/rack/api/structs"
	"github.com/stretchr/testify/mock"
)

// TestProvider is a test provider
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
	Services     structs.Services
}

// AppGet gets an App
func (p *TestProvider) AppGet(name string) (*structs.App, error) {
	p.Called(name)
	return &p.App, nil
}

// AppDelete deletes an App
func (p *TestProvider) AppDelete(name string) error {
	p.Called(name)
	return nil
}

// BuildCopy copies an App
func (p *TestProvider) BuildCopy(srcApp, id, destApp string) (*structs.Build, error) {
	p.Called(srcApp, id, destApp)
	return &p.Build, nil
}

// BuildCreateIndex creates a Build from an Index
func (p *TestProvider) BuildCreateIndex(app string, index structs.Index, manifest, description string, cache bool) (*structs.Build, error) {
	p.Called(app, index, manifest, description, cache)
	return &p.Build, nil
}

// BuildCreateRepo creates a Build from a repository URL
func (p *TestProvider) BuildCreateRepo(app, url, manifest, description string, cache bool) (*structs.Build, error) {
	p.Called(app, url, manifest, description, cache)
	return &p.Build, nil
}

// BuildCreateTar creates a Build from a tarball
func (p *TestProvider) BuildCreateTar(app string, src io.Reader, manifest, description string, cache bool) (*structs.Build, error) {
	p.Called(app, src, manifest, description, cache)
	return &p.Build, nil
}

// BuildDelete deletes a Build
func (p *TestProvider) BuildDelete(app, id string) (*structs.Build, error) {
	p.Called(app, id)
	return &p.Build, nil
}

// BuildGet gets a Build
func (p *TestProvider) BuildGet(app, id string) (*structs.Build, error) {
	p.Called(app, id)
	return &p.Build, nil
}

// BuildList lists the Builds
func (p *TestProvider) BuildList(app string, limit int64) (structs.Builds, error) {
	p.Called(app, limit)
	return p.Builds, nil
}

// BuildRelease gets the Release for a Build
func (p *TestProvider) BuildRelease(b *structs.Build) (*structs.Release, error) {
	p.Called(b)
	return &p.Release, nil
}

// BuildSave saves a build
func (p *TestProvider) BuildSave(b *structs.Build) error {
	p.Called(b)
	return nil
}

// CapacityGet gets the Capacity
func (p *TestProvider) CapacityGet() (*structs.Capacity, error) {
	p.Called()
	return &p.Capacity, nil
}

// CertificateCreate creates a Certificate
func (p *TestProvider) CertificateCreate(pub, key, chain string) (*structs.Certificate, error) {
	p.Called(pub, key, chain)
	return &p.Certificate, nil
}

// CertificateDelete deletes a Certificate
func (p *TestProvider) CertificateDelete(id string) error {
	p.Called(id)
	return nil
}

// CertificateGenerate generates a Certificatge
func (p *TestProvider) CertificateGenerate(domains []string) (*structs.Certificate, error) {
	p.Called(domains)
	return &p.Certificate, nil
}

// CertificateList lists the Certificates
func (p *TestProvider) CertificateList() (structs.Certificates, error) {
	p.Called()
	return p.Certificates, nil
}

// EventSend sends an Event
func (p *TestProvider) EventSend(e *structs.Event, err error) error {
	p.Called(e, err)
	return nil
}

// EnvironmentGet gets the Environment
func (p *TestProvider) EnvironmentGet(app string) (structs.Environment, error) {
	p.Called()
	return nil, nil
}

// IndexDiff gets a list of missing Index hashes
func (p *TestProvider) IndexDiff(i *structs.Index) ([]string, error) {
	p.Called(i)
	return []string{}, nil
}

// IndexDownload downloads an Index into a directory
func (p *TestProvider) IndexDownload(i *structs.Index, dir string) error {
	p.Called(i, dir)
	return nil
}

// IndexUpload uploads Index changes
func (p *TestProvider) IndexUpload(hash string, data []byte) error {
	p.Called(hash, data)
	return nil
}

// InstanceList lists the Instances
func (p *TestProvider) InstanceList() (structs.Instances, error) {
	p.Called()
	return p.Instances, nil
}

// LogStream streams the Logs
func (p *TestProvider) LogStream(app string, w io.Writer, opts structs.LogStreamOptions) error {
	p.Called(app, w, opts)
	return nil
}

// ReleaseDelete deletes all releases for an App and Build
func (p *TestProvider) ReleaseDelete(app, buildID string) error {
	p.Called(app, buildID)
	return nil
}

// ReleaseGet gets a Release
func (p *TestProvider) ReleaseGet(app, id string) (*structs.Release, error) {
	p.Called(app, id)
	return &p.Release, nil
}

// ReleaseList lists the Releases
func (p *TestProvider) ReleaseList(app string, limit int64) (structs.Releases, error) {
	args := p.Called(app, limit)
	return args.Get(0).(structs.Releases), args.Error(1)
}

// ReleasePromote promotes a Release
func (p *TestProvider) ReleasePromote(app, id string) (*structs.Release, error) {
	p.Called(app, id)
	return &p.Release, nil
}

// ReleaseSave saves a Release
func (p *TestProvider) ReleaseSave(r *structs.Release, logdir, key string) error {
	p.Called(r, logdir, key)
	return nil
}

// ServiceCreate creates a Service
func (p *TestProvider) ServiceCreate(name, kind string, params map[string]string) (*structs.Service, error) {
	p.Called(name, kind, params)
	return &p.Service, nil
}

// ServiceDelete deletes a Service
func (p *TestProvider) ServiceDelete(name string) (*structs.Service, error) {
	p.Called(name)
	return &p.Service, nil
}

// ServiceGet gets a Service
func (p *TestProvider) ServiceGet(name string) (*structs.Service, error) {
	p.Called(name)
	return &p.Service, nil
}

// ServiceLink links a Service
func (p *TestProvider) ServiceLink(name, app, process string) (*structs.Service, error) {
	p.Called(name, app, process)
	return &p.Service, nil
}

// ServiceList lists the Services
func (p *TestProvider) ServiceList() (structs.Services, error) {
	p.Called()
	return p.Services, nil
}

// ServiceUnlink unlinks a Service
func (p *TestProvider) ServiceUnlink(name, app, process string) (*structs.Service, error) {
	p.Called(name, app, process)
	return &p.Service, nil
}

// ServiceUpdate updates a Service
func (p *TestProvider) ServiceUpdate(name string, params map[string]string) (*structs.Service, error) {
	p.Called(name, params)
	return &p.Service, nil
}

// SystemGet gets the System
func (p *TestProvider) SystemGet() (*structs.System, error) {
	p.Called()
	return nil, nil
}

// SystemSave saves the System
func (p *TestProvider) SystemSave(system structs.System) error {
	p.Called(system)
	return nil
}
