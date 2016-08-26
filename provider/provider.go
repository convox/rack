package provider

import (
	"fmt"
	"io"
	"os"

	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/provider/aws"
)

type Provider interface {
	AppGet(name string) (*structs.App, error)
	AppDelete(name string) error

	BuildCopy(srcApp, id, destApp string) (*structs.Build, error)
	BuildCreateIndex(app string, index structs.Index, manifest, description string, cache bool) (*structs.Build, error)
	BuildCreateRepo(app, url, manifest, description string, cache bool) (*structs.Build, error)
	BuildCreateTar(app string, src io.Reader, manifest, description string, cache bool) (*structs.Build, error)
	BuildDelete(app, id string) (*structs.Build, error)
	BuildGet(app, id string) (*structs.Build, error)
	BuildGetLogs(app, id string) (string, error)
	BuildList(app string, limit int64) (structs.Builds, error)
	BuildRelease(*structs.Build) (*structs.Release, error)
	BuildSave(*structs.Build) error

	CapacityGet() (*structs.Capacity, error)

	CertificateCreate(pub, key, chain string) (*structs.Certificate, error)
	CertificateDelete(id string) error
	CertificateGenerate(domains []string) (*structs.Certificate, error)
	CertificateList() (structs.Certificates, error)

	EventSend(*structs.Event, error) error

	EnvironmentGet(app string) (structs.Environment, error)

	IndexDiff(*structs.Index) ([]string, error)
	IndexDownload(*structs.Index, string) error
	IndexUpload(string, []byte) error

	InstanceList() (structs.Instances, error)

	LogStream(app string, w io.Writer, opts structs.LogStreamOptions) error

	ReleaseDelete(app, buildID string) error
	ReleaseGet(app, id string) (*structs.Release, error)
	ReleaseList(app string, limit int64) (structs.Releases, error)
	ReleasePromote(app, id string) (*structs.Release, error)
	ReleaseSave(*structs.Release, string, string) error

	ServiceCreate(name, kind string, params map[string]string) (*structs.Service, error)
	ServiceDelete(name string) (*structs.Service, error)
	ServiceGet(name string) (*structs.Service, error)
	ServiceLink(name, app, process string) (*structs.Service, error)
	ServiceList() (structs.Services, error)
	ServiceUnlink(name, app, process string) (*structs.Service, error)
	ServiceUpdate(name string, params map[string]string) (*structs.Service, error)

	SystemGet() (*structs.System, error)
	SystemSave(system structs.System) error
}

// NewAwsProvider returns a new AWS provider
func NewAwsProvider(region, endpoint, access, secret, token string) Provider {
	return aws.NewProvider(region, endpoint, access, secret, token)
}

/** helpers ****************************************************************************************/

func die(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	os.Exit(1)
}
