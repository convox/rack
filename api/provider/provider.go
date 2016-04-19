package provider

import (
	"fmt"
	"io"
	"os"

	"github.com/convox/rack/api/provider/aws"
	"github.com/convox/rack/api/structs"
)

var CurrentProvider Provider

type Provider interface {
	AppGet(name string) (*structs.App, error)

	BuildCopy(srcApp, id, destApp string) (*structs.Build, error)
	BuildCreateIndex(app string, index structs.Index, manifest, description string, cache bool) (*structs.Build, error)
	BuildCreateRepo(app, url, manifest, description string, cache bool) (*structs.Build, error)
	BuildCreateTar(app string, src io.Reader, manifest, description string, cache bool) (*structs.Build, error)
	BuildDelete(app, id string) (*structs.Build, error)
	BuildGet(app, id string) (*structs.Build, error)
	BuildList(app string) (structs.Builds, error)
	BuildRelease(*structs.Build) (*structs.Release, error)
	BuildSave(*structs.Build) error

	CapacityGet() (*structs.Capacity, error)

	EventSend(*structs.Event, error) error

	IndexDiff(*structs.Index) ([]string, error)
	IndexDownload(*structs.Index, string) error
	IndexUpload(string, []byte) error

	InstanceList() (structs.Instances, error)

	LogStream(app string, w io.Writer, opts structs.LogStreamOptions) error

	ReleaseDelete(app, id string) (*structs.Release, error)
	ReleaseGet(app, id string) (*structs.Release, error)
	ReleaseList(app string) (structs.Releases, error)
	ReleasePromote(app, id string) (*structs.Release, error)
	ReleaseSave(*structs.Release, string, string) error

	ServiceCreate(name, kind string, params map[string]string) (*structs.Service, error)
	ServiceDelete(name string) (*structs.Service, error)
	ServiceGet(name string) (*structs.Service, error)
	ServiceLink(name, app, process string) (*structs.Service, error)
	ServiceUnlink(name, app, process string) (*structs.Service, error)

	SystemGet() (*structs.System, error)
	SystemSave(system structs.System) error
}

func init() {
	var err error

	switch os.Getenv("PROVIDER") {
	case "aws":
		CurrentProvider, err = aws.NewProvider(os.Getenv("AWS_REGION"), os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"), os.Getenv("AWS_ENDPOINT"))
	case "test":
		CurrentProvider = TestProvider
	default:
		die(fmt.Errorf("PROVIDER must be one of (aws)"))
	}

	if err != nil {
		die(err)
	}
}

/** package-level functions ************************************************************************/

func AppGet(name string) (*structs.App, error) {
	return CurrentProvider.AppGet(name)
}

func BuildCopy(srcApp, id, destApp string) (*structs.Build, error) {
	return CurrentProvider.BuildCopy(srcApp, id, destApp)
}

func BuildCreateIndex(app string, index structs.Index, manifest, description string, cache bool) (*structs.Build, error) {
	return CurrentProvider.BuildCreateIndex(app, index, manifest, description, cache)
}

func BuildCreateRepo(app, url, manifest, description string, cache bool) (*structs.Build, error) {
	return CurrentProvider.BuildCreateRepo(app, url, manifest, description, cache)
}

func BuildCreateTar(app string, src io.Reader, manifest, description string, cache bool) (*structs.Build, error) {
	return CurrentProvider.BuildCreateTar(app, src, manifest, description, cache)
}

func BuildDelete(app, id string) (*structs.Build, error) {
	return CurrentProvider.BuildDelete(app, id)
}

func BuildGet(app, id string) (*structs.Build, error) {
	return CurrentProvider.BuildGet(app, id)
}

func BuildList(app string) (structs.Builds, error) {
	return CurrentProvider.BuildList(app)
}

func BuildRelease(b *structs.Build) (*structs.Release, error) {
	return CurrentProvider.BuildRelease(b)
}

func BuildSave(b *structs.Build) error {
	return CurrentProvider.BuildSave(b)
}

func CapacityGet() (*structs.Capacity, error) {
	return CurrentProvider.CapacityGet()
}

func EventSend(e *structs.Event, err error) error {
	return CurrentProvider.EventSend(e, err)
}

func IndexDiff(i *structs.Index) ([]string, error) {
	return CurrentProvider.IndexDiff(i)
}

func IndexDownload(i *structs.Index, dir string) error {
	return CurrentProvider.IndexDownload(i, dir)
}

func IndexUpload(hash string, data []byte) error {
	return CurrentProvider.IndexUpload(hash, data)
}

func InstanceList() (structs.Instances, error) {
	return CurrentProvider.InstanceList()
}

func LogStream(app string, w io.Writer, opts structs.LogStreamOptions) error {
	return CurrentProvider.LogStream(app, w, opts)
}

func ReleaseDelete(app, id string) (*structs.Release, error) {
	return CurrentProvider.ReleaseDelete(app, id)
}

func ReleaseGet(app, id string) (*structs.Release, error) {
	return CurrentProvider.ReleaseGet(app, id)
}

func ReleaseList(app string) (structs.Releases, error) {
	return CurrentProvider.ReleaseList(app)
}

func ReleasePromote(app, id string) (*structs.Release, error) {
	return CurrentProvider.ReleasePromote(app, id)
}

func ReleaseSave(r *structs.Release, logdir, key string) error {
	return CurrentProvider.ReleaseSave(r, logdir, key)
}

func ServiceCreate(name, kind string, params map[string]string) (*structs.Service, error) {
	return CurrentProvider.ServiceCreate(name, kind, params)
}

func ServiceDelete(name string) (*structs.Service, error) {
	return CurrentProvider.ServiceDelete(name)
}

func ServiceGet(name string) (*structs.Service, error) {
	return CurrentProvider.ServiceGet(name)
}

func ServiceLink(name, app, process string) (*structs.Service, error) {
	return CurrentProvider.ServiceLink(name, app, process)
}

func ServiceUnlink(name, app, process string) (*structs.Service, error) {
	return CurrentProvider.ServiceUnlink(name, app, process)
}

func SystemGet() (*structs.System, error) {
	return CurrentProvider.SystemGet()
}

func SystemSave(system structs.System) error {
	return CurrentProvider.SystemSave(system)
}

/** helpers ****************************************************************************************/

func die(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	os.Exit(1)
}
