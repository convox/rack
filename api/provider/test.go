package provider

import (
	"io"

	"github.com/convox/rack/api/structs"
)

var TestProvider = &TestProviderRunner{}

type TestProviderRunner struct {
	Apps structs.Apps
	App  *structs.App

	Instances structs.Instances

	Processes structs.Processes
}

func (p *TestProviderRunner) AppList() (structs.Apps, error) {
	return p.Apps, nil
}

func (p *TestProviderRunner) AppGet(name string) (*structs.App, error) {
	return p.App, nil
}

func (p *TestProviderRunner) AppCreate(name string) error {
	return nil
}

func (p *TestProviderRunner) AppDelete(app *structs.App) error {
	return nil
}

func (p *TestProviderRunner) CapacityGet() (*structs.Capacity, error) {
	return nil, nil
}

func (p *TestProviderRunner) InstanceList() (structs.Instances, error) {
	return p.Instances, nil
}

func (p *TestProviderRunner) EnvironmentGet(app string) (structs.Environment, error) {
	return nil, nil
}

func (p *TestProviderRunner) EnvironmentSet(app string, env structs.Environment) (string, error) {
	return "", nil
}

func (p *TestProviderRunner) NotifySuccess(action string, data map[string]string) error {
	return nil
}

func (p *TestProviderRunner) NotifyError(action string, err error, data map[string]string) error {
	return nil
}

func (p *TestProviderRunner) ProcessList(app string) (structs.Processes, error) {
	return p.Processes, nil
}

func (p *TestProviderRunner) ProcessGet(app, pid string) (*structs.Process, error) {
	return nil, nil
}

func (p *TestProviderRunner) ProcessStop(app, pid string) error {
	return nil
}

func (p *TestProviderRunner) ProcessExec(app, pid, command string, rw io.ReadWriter) error {
	return nil
}

func (p *TestProviderRunner) ProcessStats(app, pid string) (*structs.ProcessStats, error) {
	return nil, nil
}

func (p *TestProviderRunner) ReleaseList(app string) (structs.Releases, error) {
	return nil, nil
}

func (p *TestProviderRunner) ReleaseGet(app, id string) (*structs.Release, error) {
	return nil, nil
}

func (p *TestProviderRunner) ReleaseSave(release *structs.Release) error {
	return nil
}

func (p *TestProviderRunner) ReleaseFork(app string) (*structs.Release, error) {
	return nil, nil
}

func (p *TestProviderRunner) RunAttached(app, process, command string, rw io.ReadWriter) error {
	return nil
}

func (p *TestProviderRunner) RunDetached(app, process, command string) error {
	return nil
}

func (p *TestProviderRunner) SystemGet() (*structs.System, error) {
	return nil, nil
}

func (p *TestProviderRunner) SystemSave(system *structs.System) error {
	return nil
}
