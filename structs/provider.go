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
	AppLogs(app string, opts LogsOptions) (io.ReadCloser, error)
	AppUpdate(app string, opts AppUpdateOptions) error

	BuildCreate(app, method, source string, opts BuildCreateOptions) (*Build, error)
	BuildExport(app, id string, w io.Writer) error
	BuildGet(app, id string) (*Build, error)
	BuildImport(app string, r io.Reader) (*Build, error)
	BuildLogs(app, id string, opts LogsOptions) (io.ReadCloser, error)
	BuildList(app string, opts BuildListOptions) (Builds, error)
	BuildUpdate(app, id string, opts BuildUpdateOptions) (*Build, error)

	CapacityGet() (*Capacity, error)

	CertificateApply(app, service string, port int, id string) error
	CertificateCreate(pub, key, chain string) (*Certificate, error)
	CertificateDelete(id string) error
	CertificateGenerate(domains []string) (*Certificate, error)
	CertificateList() (Certificates, error)

	// EnvironmentGet(app string) (Environment, error)
	// EnvironmentPut(app string, env Environment) (string, error)

	EventSend(*Event, error) error

	// FormationGet(app, process string) (*ProcessFormation, error)
	// FormationList(app string) (Formation, error)
	// FormationSave(app string, pf *ProcessFormation) error

	// IndexDiff(*Index) ([]string, error)
	// IndexDownload(*Index, string) error
	// IndexUpload(string, []byte) error

	InstanceKeyroll() error
	InstanceList() (Instances, error)
	InstanceShell(id string, rw io.ReadWriter, opts InstanceShellOptions) error
	InstanceTerminate(id string) error

	ObjectDelete(app, key string) error
	ObjectExists(app, key string) (bool, error)
	ObjectFetch(app, key string) (io.ReadCloser, error)
	ObjectList(app, prefix string) ([]string, error)
	ObjectStore(app, key string, r io.Reader, opts ObjectStoreOptions) (*Object, error)

	ProcessExec(app, pid, command string, opts ProcessExecOptions) (int, error)
	ProcessGet(app, pid string) (*Process, error)
	ProcessList(app string, opts ProcessListOptions) (Processes, error)
	ProcessRun(app string, opts ProcessRunOptions) (string, error)
	ProcessStop(app, pid string) error
	ProcessWait(app, pid string) (int, error)

	RegistryAdd(server, username, password string) (*Registry, error)
	RegistryList() (Registries, error)
	RegistryRemove(server string) error

	// ReleaseDelete(app, buildID string) error
	ReleaseCreate(app string, opts ReleaseCreateOptions) (*Release, error)
	ReleaseGet(app, id string) (*Release, error)
	ReleaseList(app string, opts ReleaseListOptions) (Releases, error)
	ReleasePromote(app, id string) error
	// ReleaseSave(*Release) error

	ResourceCreate(name, kind string, opts ResourceCreateOptions) (*Resource, error)
	ResourceDelete(name string) (*Resource, error)
	ResourceGet(name string) (*Resource, error)
	ResourceLink(name, app, process string) (*Resource, error)
	ResourceList() (Resources, error)
	ResourceUnlink(name, app, process string) (*Resource, error)
	ResourceUpdate(name string, params map[string]string) (*Resource, error)

	ServiceList(app string) (Services, error)
	ServiceUpdate(app, name string, opts ServiceUpdateOptions) error

	SettingDelete(name string) error
	SettingExists(name string) (bool, error)
	SettingGet(name string) (string, error)
	SettingList(opts SettingListOptions) ([]string, error)
	SettingPut(name, value string) error

	SystemDecrypt(data []byte) ([]byte, error)
	SystemEncrypt(data []byte) ([]byte, error)
	SystemGet() (*System, error)
	SystemLogs(opts LogsOptions) (io.ReadCloser, error)
	SystemProcesses(opts SystemProcessesOptions) (Processes, error)
	SystemUpdate(opts SystemUpdateOptions) error

	Workers() error
}

type ProviderOptions struct {
	Logs io.Writer
}
