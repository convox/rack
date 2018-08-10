package sdk

import (
	"fmt"
	"io"
	"strings"

	"github.com/convox/rack/structs"
	"github.com/convox/stdsdk"
)

func (c *Client) AppCancel(name string) error {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	err = c.Post(fmt.Sprintf("/apps/%s/cancel", name), ro, nil)

	return err
}

func (c *Client) AppCreate(name string, opts structs.AppCreateOptions) (*structs.App, error) {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return nil, err
	}

	ro.Params["name"] = name

	var v *structs.App

	err = c.Post(fmt.Sprintf("/apps"), ro, &v)

	return v, err
}

func (c *Client) AppDelete(name string) error {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	err = c.Delete(fmt.Sprintf("/apps/%s", name), ro, nil)

	return err
}

func (c *Client) AppGet(name string) (*structs.App, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	var v *structs.App

	err = c.Get(fmt.Sprintf("/apps/%s", name), ro, &v)

	return v, err
}

func (c *Client) AppList() (structs.Apps, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	var v structs.Apps

	err = c.Get(fmt.Sprintf("/apps"), ro, &v)

	return v, err
}

func (c *Client) AppLogs(name string, opts structs.LogsOptions) (io.ReadCloser, error) {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return nil, err
	}

	var v io.ReadCloser

	r, err := c.Websocket(fmt.Sprintf("/apps/%s/logs", name), ro)
	if err != nil {
		return nil, err
	}

	v = r

	return v, err
}

func (c *Client) AppUpdate(name string, opts structs.AppUpdateOptions) error {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return err
	}

	err = c.Put(fmt.Sprintf("/apps/%s", name), ro, nil)

	return err
}

func (c *Client) BuildCreate(app string, url string, opts structs.BuildCreateOptions) (*structs.Build, error) {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return nil, err
	}

	ro.Params["url"] = url

	var v *structs.Build

	err = c.Post(fmt.Sprintf("/apps/%s/builds", app), ro, &v)

	return v, err
}

func (c *Client) BuildExport(app string, id string, w io.Writer) error {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	res, err := c.GetStream(fmt.Sprintf("/apps/%s/builds/%s.tgz", app, id), ro)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if _, err := io.Copy(w, res.Body); err != nil {
		return err
	}

	return err
}

func (c *Client) BuildGet(app string, id string) (*structs.Build, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	var v *structs.Build

	err = c.Get(fmt.Sprintf("/apps/%s/builds/%s", app, id), ro, &v)

	return v, err
}

func (c *Client) BuildImport(app string, r io.Reader) (*structs.Build, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	ro.Body = r

	var v *structs.Build

	err = c.Post(fmt.Sprintf("/apps/%s/builds/import", app), ro, &v)

	return v, err
}

func (c *Client) BuildList(app string, opts structs.BuildListOptions) (structs.Builds, error) {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return nil, err
	}

	var v structs.Builds

	err = c.Get(fmt.Sprintf("/apps/%s/builds", app), ro, &v)

	return v, err
}

func (c *Client) BuildLogs(app string, id string, opts structs.LogsOptions) (io.ReadCloser, error) {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return nil, err
	}

	var v io.ReadCloser

	r, err := c.Websocket(fmt.Sprintf("/apps/%s/builds/%s/logs", app, id), ro)
	if err != nil {
		return nil, err
	}

	v = r

	return v, err
}

func (c *Client) BuildUpdate(app string, id string, opts structs.BuildUpdateOptions) (*structs.Build, error) {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return nil, err
	}

	var v *structs.Build

	err = c.Put(fmt.Sprintf("/apps/%s/builds/%s", app, id), ro, &v)

	return v, err
}

func (c *Client) CapacityGet() (*structs.Capacity, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	var v *structs.Capacity

	err = c.Get(fmt.Sprintf("/system/capacity"), ro, &v)

	return v, err
}

func (c *Client) CertificateApply(app string, service string, port int, id string) error {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	ro.Params["id"] = id

	err = c.Put(fmt.Sprintf("/apps/%s/ssl/%s/%d", app, service, port), ro, nil)

	return err
}

func (c *Client) CertificateCreate(pub string, key string, opts structs.CertificateCreateOptions) (*structs.Certificate, error) {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return nil, err
	}

	ro.Params["pub"] = pub
	ro.Params["key"] = key

	var v *structs.Certificate

	err = c.Post(fmt.Sprintf("/certificates"), ro, &v)

	return v, err
}

func (c *Client) CertificateDelete(id string) error {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	err = c.Delete(fmt.Sprintf("/certificates/%s", id), ro, nil)

	return err
}

func (c *Client) CertificateGenerate(domains []string) (*structs.Certificate, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	ro.Params["domains"] = strings.Join(domains, ",")

	var v *structs.Certificate

	err = c.Post(fmt.Sprintf("/certificates/generate"), ro, &v)

	return v, err
}

func (c *Client) CertificateList() (structs.Certificates, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	var v structs.Certificates

	err = c.Get(fmt.Sprintf("/certificates"), ro, &v)

	return v, err
}

func (c *Client) EventSend(action string, opts structs.EventSendOptions) error {
	err := fmt.Errorf("not available via api")
	return err
}

func (c *Client) FilesDelete(app string, pid string, files []string) error {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	ro.Params["files"] = strings.Join(files, ",")

	err = c.Delete(fmt.Sprintf("/apps/%s/processes/%s/files", app, pid), ro, nil)

	return err
}

func (c *Client) FilesUpload(app string, pid string, r io.Reader) error {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	ro.Body = r

	err = c.Post(fmt.Sprintf("/apps/%s/processes/%s/files", app, pid), ro, nil)

	return err
}

func (c *Client) Initialize(opts structs.ProviderOptions) error {
	err := fmt.Errorf("not available via api")
	return err
}

func (c *Client) InstanceKeyroll() error {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	err = c.Post(fmt.Sprintf("/instances/keyroll"), ro, nil)

	return err
}

func (c *Client) InstanceList() (structs.Instances, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	var v structs.Instances

	err = c.Get(fmt.Sprintf("/instances"), ro, &v)

	return v, err
}

func (c *Client) InstanceShell(id string, rw io.ReadWriter, opts structs.InstanceShellOptions) (int, error) {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return 0, err
	}

	ro.Body = rw

	var v int

	v, err = c.WebsocketExit(fmt.Sprintf("/instances/%s/shell", id), ro, rw)
	if err != nil {
		return 0, err
	}

	return v, err
}

func (c *Client) InstanceTerminate(id string) error {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	err = c.Delete(fmt.Sprintf("/instances/%s", id), ro, nil)

	return err
}

func (c *Client) ObjectDelete(app string, key string) error {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	err = c.Delete(fmt.Sprintf("/apps/%s/objects/%s", app, key), ro, nil)

	return err
}

func (c *Client) ObjectExists(app string, key string) (bool, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	var v bool

	err = c.Head(fmt.Sprintf("/apps/%s/objects/%s", app, key), ro, &v)

	return v, err
}

func (c *Client) ObjectFetch(app string, key string) (io.ReadCloser, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	var v io.ReadCloser

	res, err := c.GetStream(fmt.Sprintf("/apps/%s/objects/%s", app, key), ro)
	if err != nil {
		return nil, err
	}

	v = res.Body

	return v, err
}

func (c *Client) ObjectList(app string, prefix string) ([]string, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	ro.Params["prefix"] = prefix

	var v []string

	err = c.Get(fmt.Sprintf("/apps/%s/objects", app), ro, &v)

	return v, err
}

func (c *Client) ObjectStore(app string, key string, r io.Reader, opts structs.ObjectStoreOptions) (*structs.Object, error) {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return nil, err
	}

	ro.Body = r

	var v *structs.Object

	err = c.Post(fmt.Sprintf("/apps/%s/objects/%s", app, key), ro, &v)

	return v, err
}

func (c *Client) ProcessExec(app string, pid string, command string, rw io.ReadWriter, opts structs.ProcessExecOptions) (int, error) {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return 0, err
	}

	ro.Headers["command"] = command
	ro.Body = rw

	var v int

	v, err = c.WebsocketExit(fmt.Sprintf("/apps/%s/processes/%s/exec", app, pid), ro, rw)
	if err != nil {
		return 0, err
	}

	return v, err
}

func (c *Client) ProcessGet(app string, pid string) (*structs.Process, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	var v *structs.Process

	err = c.Get(fmt.Sprintf("/apps/%s/processes/%s", app, pid), ro, &v)

	return v, err
}

func (c *Client) ProcessList(app string, opts structs.ProcessListOptions) (structs.Processes, error) {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return nil, err
	}

	var v structs.Processes

	err = c.Get(fmt.Sprintf("/apps/%s/processes", app), ro, &v)

	return v, err
}

func (c *Client) ProcessRun(app string, service string, opts structs.ProcessRunOptions) (*structs.Process, error) {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return nil, err
	}

	var v *structs.Process

	err = c.Post(fmt.Sprintf("/apps/%s/services/%s/processes", app, service), ro, &v)

	return v, err
}

func (c *Client) ProcessStop(app string, pid string) error {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	err = c.Delete(fmt.Sprintf("/apps/%s/processes/%s", app, pid), ro, nil)

	return err
}

func (c *Client) ProcessWait(app string, pid string) (int, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	var v int

	err = c.Get(fmt.Sprintf("/apps/%s/processes/%s/wait", app, pid), ro, &v)

	return v, err
}

func (c *Client) Proxy(host string, port int, rw io.ReadWriter) error {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	ro.Body = rw

	r, err := c.Websocket(fmt.Sprintf("/proxy/%s/%d", host, port), ro)
	if err != nil {
		return err
	}

	if _, err := io.Copy(rw, r); err != nil {
		return err
	}

	return err
}

func (c *Client) RegistryAdd(server string, username string, password string) (*structs.Registry, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	ro.Params["server"] = server
	ro.Params["username"] = username
	ro.Params["password"] = password

	var v *structs.Registry

	err = c.Post(fmt.Sprintf("/registries"), ro, &v)

	return v, err
}

func (c *Client) RegistryList() (structs.Registries, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	var v structs.Registries

	err = c.Get(fmt.Sprintf("/registries"), ro, &v)

	return v, err
}

func (c *Client) RegistryRemove(server string) error {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	err = c.Delete(fmt.Sprintf("/registries/%s", server), ro, nil)

	return err
}

func (c *Client) ReleaseCreate(app string, opts structs.ReleaseCreateOptions) (*structs.Release, error) {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return nil, err
	}

	var v *structs.Release

	err = c.Post(fmt.Sprintf("/apps/%s/releases", app), ro, &v)

	return v, err
}

func (c *Client) ReleaseGet(app string, id string) (*structs.Release, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	var v *structs.Release

	err = c.Get(fmt.Sprintf("/apps/%s/releases/%s", app, id), ro, &v)

	return v, err
}

func (c *Client) ReleaseList(app string, opts structs.ReleaseListOptions) (structs.Releases, error) {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return nil, err
	}

	var v structs.Releases

	err = c.Get(fmt.Sprintf("/apps/%s/releases", app), ro, &v)

	return v, err
}

func (c *Client) ReleasePromote(app string, id string) error {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	err = c.Post(fmt.Sprintf("/apps/%s/releases/%s/promote", app, id), ro, nil)

	return err
}

func (c *Client) ResourceCreate(kind string, opts structs.ResourceCreateOptions) (*structs.Resource, error) {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return nil, err
	}

	ro.Params["kind"] = kind

	var v *structs.Resource

	err = c.Post(fmt.Sprintf("/resources"), ro, &v)

	return v, err
}

func (c *Client) ResourceDelete(name string) error {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	err = c.Delete(fmt.Sprintf("/resources/%s", name), ro, nil)

	return err
}

func (c *Client) ResourceGet(name string) (*structs.Resource, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	var v *structs.Resource

	err = c.Get(fmt.Sprintf("/resources/%s", name), ro, &v)

	return v, err
}

func (c *Client) ResourceLink(name string, app string) (*structs.Resource, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	ro.Params["app"] = app

	var v *structs.Resource

	err = c.Post(fmt.Sprintf("/resources/%s/links", name), ro, &v)

	return v, err
}

func (c *Client) ResourceList() (structs.Resources, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	var v structs.Resources

	err = c.Get(fmt.Sprintf("/resources"), ro, &v)

	return v, err
}

func (c *Client) ResourceTypes() (structs.ResourceTypes, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	var v structs.ResourceTypes

	err = c.Options(fmt.Sprintf("/resources"), ro, &v)

	return v, err
}

func (c *Client) ResourceUnlink(name string, app string) (*structs.Resource, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	var v *structs.Resource

	err = c.Delete(fmt.Sprintf("/resources/%s/links/%s", name, app), ro, &v)

	return v, err
}

func (c *Client) ResourceUpdate(name string, opts structs.ResourceUpdateOptions) (*structs.Resource, error) {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return nil, err
	}

	var v *structs.Resource

	err = c.Put(fmt.Sprintf("/resources/%s", name), ro, &v)

	return v, err
}

func (c *Client) ServiceList(app string) (structs.Services, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	var v structs.Services

	err = c.Get(fmt.Sprintf("/apps/%s/services", app), ro, &v)

	return v, err
}

func (c *Client) ServiceUpdate(app string, name string, opts structs.ServiceUpdateOptions) error {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return err
	}

	err = c.Put(fmt.Sprintf("/apps/%s/services/%s", app, name), ro, nil)

	return err
}

func (c *Client) SystemGet() (*structs.System, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	var v *structs.System

	err = c.Get(fmt.Sprintf("/system"), ro, &v)

	return v, err
}

func (c *Client) SystemInstall(opts structs.SystemInstallOptions) (string, error) {
	err := fmt.Errorf("not available via api")
	return "", err
}

func (c *Client) SystemLogs(opts structs.LogsOptions) (io.ReadCloser, error) {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return nil, err
	}

	var v io.ReadCloser

	r, err := c.Websocket(fmt.Sprintf("/system/logs"), ro)
	if err != nil {
		return nil, err
	}

	v = r

	return v, err
}

func (c *Client) SystemProcesses(opts structs.SystemProcessesOptions) (structs.Processes, error) {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return nil, err
	}

	var v structs.Processes

	err = c.Get(fmt.Sprintf("/system/processes"), ro, &v)

	return v, err
}

func (c *Client) SystemReleases() (structs.Releases, error) {
	var err error

	ro := stdsdk.RequestOptions{Headers: stdsdk.Headers{}, Params: stdsdk.Params{}}

	var v structs.Releases

	err = c.Get(fmt.Sprintf("/system/releases"), ro, &v)

	return v, err
}

func (c *Client) SystemUninstall(name string, opts structs.SystemUninstallOptions) error {
	err := fmt.Errorf("not available via api")
	return err
}

func (c *Client) SystemUpdate(opts structs.SystemUpdateOptions) error {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return err
	}

	err = c.Put(fmt.Sprintf("/system"), ro, nil)

	return err
}

func (c *Client) Workers() error {
	err := fmt.Errorf("not available via api")
	return err
}

