package structs

import (
	"io"
)

type Provider interface {
	Initialize(opts ProviderOptions) error

	AppCancel(name string) error
	AppCreate(name string, opts AppCreateOptions) (*App, error)
	AppGet(name string) (*App, error)
	AppDelete(name string) error
	AppList() (Apps, error)
	AppLogs(name string, opts LogsOptions) (io.ReadCloser, error)
	AppUpdate(name string, opts AppUpdateOptions) error

	BuildCreate(app, url string, opts BuildCreateOptions) (*Build, error)
	BuildExport(app, id string, w io.Writer) error
	BuildGet(app, id string) (*Build, error)
	BuildImport(app string, r io.Reader) (*Build, error)
	BuildLogs(app, id string, opts LogsOptions) (io.ReadCloser, error)
	BuildList(app string, opts BuildListOptions) (Builds, error)
	BuildUpdate(app, id string, opts BuildUpdateOptions) (*Build, error)

	CapacityGet() (*Capacity, error)

	CertificateApply(app, service string, port int, id string) error
	CertificateCreate(pub, key string, opts CertificateCreateOptions) (*Certificate, error)
	CertificateDelete(id string) error
	CertificateGenerate(domains []string) (*Certificate, error)
	CertificateList() (Certificates, error)

	EventSend(action string, opts EventSendOptions) error

	FilesDelete(app, pid string, files []string) error
	FilesUpload(app, pid string, r io.Reader) error

	InstanceKeyroll() error
	InstanceList() (Instances, error)
	InstanceShell(id string, rw io.ReadWriter, opts InstanceShellOptions) (int, error)
	InstanceTerminate(id string) error

	ObjectDelete(app, key string) error
	ObjectExists(app, key string) (bool, error)
	ObjectFetch(app, key string) (io.ReadCloser, error)
	ObjectList(app, prefix string) ([]string, error)
	ObjectStore(app, key string, r io.Reader, opts ObjectStoreOptions) (*Object, error)

	ProcessExec(app, pid, command string, rw io.ReadWriter, opts ProcessExecOptions) (int, error)
	ProcessGet(app, pid string) (*Process, error)
	ProcessList(app string, opts ProcessListOptions) (Processes, error)
	ProcessRun(app, service string, opts ProcessRunOptions) (*Process, error)
	ProcessStop(app, pid string) error

	Proxy(host string, port int, rw io.ReadWriter) error

	RegistryAdd(server, username, password string) (*Registry, error)
	RegistryList() (Registries, error)
	RegistryRemove(server string) error

	ReleaseCreate(app string, opts ReleaseCreateOptions) (*Release, error)
	ReleaseGet(app, id string) (*Release, error)
	ReleaseList(app string, opts ReleaseListOptions) (Releases, error)
	ReleasePromote(app, id string) error

	ResourceCreate(kind string, opts ResourceCreateOptions) (*Resource, error)
	ResourceDelete(name string) error
	ResourceGet(name string) (*Resource, error)
	ResourceLink(name, app string) (*Resource, error)
	ResourceList() (Resources, error)
	ResourceTypes() (ResourceTypes, error)
	ResourceUnlink(name, app string) (*Resource, error)
	ResourceUpdate(name string, opts ResourceUpdateOptions) (*Resource, error)

	ServiceList(app string) (Services, error)
	ServiceUpdate(app, name string, opts ServiceUpdateOptions) error

	SystemGet() (*System, error)
	SystemInstall(opts SystemInstallOptions) (string, error)
	SystemLogs(opts LogsOptions) (io.ReadCloser, error)
	SystemProcesses(opts SystemProcessesOptions) (Processes, error)
	SystemReleases() (Releases, error)
	SystemUninstall(name string, opts SystemUninstallOptions) error
	SystemUpdate(opts SystemUpdateOptions) error

	Workers() error
}

type ProviderOptions struct {
	Logs io.Writer
}
