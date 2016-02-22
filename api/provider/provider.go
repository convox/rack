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
	AppList() (structs.Apps, error)
	AppGet(name string) (*structs.App, error)
	AppCreate(name string) error
	AppDelete(app *structs.App) error

	CapacityGet() (*structs.Capacity, error)

	InstanceList() (structs.Instances, error)

	NotifySuccess(action string, data map[string]string) error
	NotifyError(action string, err error, data map[string]string) error

	ProcessList(app string) (structs.Processes, error)
	ProcessGet(app, pid string) (*structs.Process, error)
	ProcessStop(app, pid string) error
	ProcessExec(app, pid, command string, rw io.ReadWriter) error
	ProcessStats(app, pid string) (*structs.ProcessStats, error)

	ReleaseList(app string) (structs.Releases, error)
	ReleaseGet(app, id string) (*structs.Release, error)
	ReleaseSave(release *structs.Release) error
	// ReleasePromote(app, release string) error

	RunAttached(app, process, command string, rw io.ReadWriter) error
	RunDetached(app, process, command string) error

	SystemGet() (*structs.System, error)
	SystemSave(system *structs.System) error
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

func AppList() (structs.Apps, error) {
	return CurrentProvider.AppList()
}

func AppGet(name string) (*structs.App, error) {
	return CurrentProvider.AppGet(name)
}

func AppCreate(name string) error {
	return CurrentProvider.AppCreate(name)
}

func AppDelete(app *structs.App) error {
	return CurrentProvider.AppDelete(app)
}

func CapacityGet() (*structs.Capacity, error) {
	return CurrentProvider.CapacityGet()
}

func InstanceList() (structs.Instances, error) {
	return CurrentProvider.InstanceList()
}

func NotifySuccess(action string, data map[string]string) error {
	return CurrentProvider.NotifySuccess(action, data)
}

func NotifyError(action string, err error, data map[string]string) error {
	return CurrentProvider.NotifyError(action, err, data)
}

func ProcessList(app string) (structs.Processes, error) {
	return CurrentProvider.ProcessList(app)
}

func ProcessGet(app, pid string) (*structs.Process, error) {
	return CurrentProvider.ProcessGet(app, pid)
}

func ProcessStop(app, pid string) error {
	return CurrentProvider.ProcessStop(app, pid)
}

func ProcessExec(app, pid, command string, rw io.ReadWriter) error {
	return CurrentProvider.ProcessExec(app, pid, command, rw)
}

func ProcessStats(app, pid string) (*structs.ProcessStats, error) {
	return CurrentProvider.ProcessStats(app, pid)
}

func ReleaseList(app string) (structs.Releases, error) {
	return CurrentProvider.ReleaseList(app)
}

func ReleaseGet(app, id string) (*structs.Release, error) {
	return CurrentProvider.ReleaseGet(app, id)
}

func ReleaseSave(release *structs.Release) error {
	return CurrentProvider.ReleaseSave(release)
}

func RunAttached(app, process, command string, rw io.ReadWriter) error {
	return CurrentProvider.RunAttached(app, process, command, rw)
}

func RunDetached(app, process, command string) error {
	return CurrentProvider.RunDetached(app, process, command)
}

func SystemGet() (*structs.System, error) {
	return CurrentProvider.SystemGet()
}

func SystemSave(system *structs.System) error {
	return CurrentProvider.SystemSave(system)
}

/** helpers ****************************************************************************************/

func die(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	os.Exit(1)
}
