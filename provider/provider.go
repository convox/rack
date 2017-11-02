package provider

import (
	"fmt"
	"io"
	"os"

	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/provider/aws"
)

type Provider interface {
	Initialize(opts structs.ProviderOptions) error

	AppCancel(name string) error
	AppCreate(name string) (*structs.App, error)
	AppGet(name string) (*structs.App, error)
	AppDelete(name string) error

	BuildCreate(app, method, source string, opts structs.BuildOptions) (*structs.Build, error)
	BuildDelete(app, id string) (*structs.Build, error)
	BuildExport(app, id string, w io.Writer) error
	BuildGet(app, id string) (*structs.Build, error)
	BuildImport(app string, r io.Reader) (*structs.Build, error)
	BuildLogs(app, id string, w io.Writer) error
	BuildList(app string, limit int64) (structs.Builds, error)
	BuildRelease(*structs.Build) (*structs.Release, error)
	BuildSave(*structs.Build) error

	CapacityGet() (*structs.Capacity, error)

	CertificateCreate(pub, key, chain string) (*structs.Certificate, error)
	CertificateDelete(id string) error
	CertificateGenerate(domains []string) (*structs.Certificate, error)
	CertificateList() (structs.Certificates, error)

	EventSend(*structs.Event, error) error

	KeyDecrypt(data []byte) ([]byte, error)
	KeyEncrypt(data []byte) ([]byte, error)

	EnvironmentGet(app string) (structs.Environment, error)

	FormationList(app string) (structs.Formation, error)
	FormationGet(app, process string) (*structs.ProcessFormation, error)
	FormationSave(app string, pf *structs.ProcessFormation) error

	IndexDiff(*structs.Index) ([]string, error)
	IndexDownload(*structs.Index, string) error
	IndexUpload(string, []byte) error

	InstanceList() (structs.Instances, error)
	InstanceTerminate(id string) error

	LogStream(app string, w io.Writer, opts structs.LogStreamOptions) error

	ObjectDelete(key string) error
	ObjectExists(key string) bool
	ObjectFetch(key string) (io.ReadCloser, error)
	ObjectList(prefix string) ([]string, error)
	ObjectStore(key string, r io.Reader, opts structs.ObjectOptions) (string, error)

	ProcessExec(app, pid, command string, stream io.ReadWriter, opts structs.ProcessExecOptions) error
	ProcessGet(app, pid string) (*structs.Process, error)
	ProcessList(app string) (structs.Processes, error)
	ProcessRun(app, process string, opts structs.ProcessRunOptions) (string, error)
	ProcessStop(app, pid string) error

	RegistryAdd(server, username, password string) (*structs.Registry, error)
	RegistryDelete(server string) error
	RegistryList() (structs.Registries, error)

	ReleaseDelete(app, buildID string) error
	ReleaseGet(app, id string) (*structs.Release, error)
	ReleaseList(app string, limit int64) (structs.Releases, error)
	ReleasePromote(*structs.Release) error
	ReleaseSave(*structs.Release) error

	ResourceCreate(name, kind string, params map[string]string) (*structs.Resource, error)
	ResourceDelete(name string) (*structs.Resource, error)
	ResourceGet(name string) (*structs.Resource, error)
	ResourceLink(name, app, process string) (*structs.Resource, error)
	ResourceList() (structs.Resources, error)
	ResourceUnlink(name, app, process string) (*structs.Resource, error)
	ResourceUpdate(name string, params map[string]string) (*structs.Resource, error)

	SystemGet() (*structs.System, error)
	SystemLogs(w io.Writer, opts structs.LogStreamOptions) error
	SystemProcesses(opts structs.SystemProcessesOptions) (structs.Processes, error)
	SystemReleases() (structs.Releases, error)
	SystemSave(system structs.System) error
}

// FromEnv returns a new Provider from env vars
func FromEnv() Provider {
	switch os.Getenv("PROVIDER") {
	case "aws":
		return aws.FromEnv()
	case "test":
		return &MockProvider{}
	default:
		panic(fmt.Errorf("must set PROVIDER to one of (aws, test)"))
	}
}
