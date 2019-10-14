package api

import (
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdapi"
)

func (s *Server) AppCancel(c *stdapi.Context) error {
	if err := s.hook("AppCancelValidate", c); err != nil {
		return err
	}

	name := c.Var("name")

	err := s.provider(c).WithContext(c.Context()).AppCancel(name)
	if err != nil {
		return err
	}

	return c.RenderOK()
}

func (s *Server) AppCreate(c *stdapi.Context) error {
	if err := s.hook("AppCreateValidate", c); err != nil {
		return err
	}

	name := c.Value("name")

	var opts structs.AppCreateOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).AppCreate(name, opts)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) AppDelete(c *stdapi.Context) error {
	if err := s.hook("AppDeleteValidate", c); err != nil {
		return err
	}

	name := c.Var("name")

	err := s.provider(c).WithContext(c.Context()).AppDelete(name)
	if err != nil {
		return err
	}

	return c.RenderOK()
}

func (s *Server) AppGet(c *stdapi.Context) error {
	if err := s.hook("AppGetValidate", c); err != nil {
		return err
	}

	name := c.Var("name")

	v, err := s.provider(c).WithContext(c.Context()).AppGet(name)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) AppList(c *stdapi.Context) error {
	if err := s.hook("AppListValidate", c); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).AppList()
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) AppLogs(c *stdapi.Context) error {
	if err := s.hook("AppLogsValidate", c); err != nil {
		return err
	}

	name := c.Var("name")

	var opts structs.LogsOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).AppLogs(name, opts)
	if err != nil {
		return err
	}

	if c, ok := interface{}(v).(io.Closer); ok {
		defer c.Close()
	}

	if _, err := io.Copy(c, v); err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return nil
}

func (s *Server) AppMetrics(c *stdapi.Context) error {
	if err := s.hook("AppMetricsValidate", c); err != nil {
		return err
	}

	name := c.Var("name")

	var opts structs.MetricsOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).AppMetrics(name, opts)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) AppUpdate(c *stdapi.Context) error {
	if err := s.hook("AppUpdateValidate", c); err != nil {
		return err
	}

	name := c.Var("name")

	var opts structs.AppUpdateOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	err := s.provider(c).WithContext(c.Context()).AppUpdate(name, opts)
	if err != nil {
		return err
	}

	return c.RenderOK()
}

func (s *Server) BuildCreate(c *stdapi.Context) error {
	if err := s.hook("BuildCreateValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	url := c.Value("url")

	var opts structs.BuildCreateOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).BuildCreate(app, url, opts)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) BuildExport(c *stdapi.Context) error {
	if err := s.hook("BuildExportValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	id := c.Var("id")
	w := c

	err := s.provider(c).WithContext(c.Context()).BuildExport(app, id, w)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) BuildGet(c *stdapi.Context) error {
	if err := s.hook("BuildGetValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	id := c.Var("id")

	v, err := s.provider(c).WithContext(c.Context()).BuildGet(app, id)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) BuildImport(c *stdapi.Context) error {
	if err := s.hook("BuildImportValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	r := c

	v, err := s.provider(c).WithContext(c.Context()).BuildImport(app, r)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) BuildList(c *stdapi.Context) error {
	if err := s.hook("BuildListValidate", c); err != nil {
		return err
	}

	app := c.Var("app")

	var opts structs.BuildListOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).BuildList(app, opts)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) BuildLogs(c *stdapi.Context) error {
	if err := s.hook("BuildLogsValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	id := c.Var("id")

	var opts structs.LogsOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).BuildLogs(app, id, opts)
	if err != nil {
		return err
	}

	if c, ok := interface{}(v).(io.Closer); ok {
		defer c.Close()
	}

	if _, err := io.Copy(c, v); err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return nil
}

func (s *Server) BuildUpdate(c *stdapi.Context) error {
	if err := s.hook("BuildUpdateValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	id := c.Var("id")

	var opts structs.BuildUpdateOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).BuildUpdate(app, id, opts)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) CapacityGet(c *stdapi.Context) error {
	if err := s.hook("CapacityGetValidate", c); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).CapacityGet()
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) CertificateApply(c *stdapi.Context) error {
	if err := s.hook("CertificateApplyValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	service := c.Var("service")
	id := c.Value("id")

	port, cerr := strconv.Atoi(c.Var("port"))
	if cerr != nil {
		return cerr
	}

	err := s.provider(c).WithContext(c.Context()).CertificateApply(app, service, port, id)
	if err != nil {
		return err
	}

	return c.RenderOK()
}

func (s *Server) CertificateCreate(c *stdapi.Context) error {
	if err := s.hook("CertificateCreateValidate", c); err != nil {
		return err
	}

	pub := c.Value("pub")
	key := c.Value("key")

	var opts structs.CertificateCreateOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).CertificateCreate(pub, key, opts)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) CertificateDelete(c *stdapi.Context) error {
	if err := s.hook("CertificateDeleteValidate", c); err != nil {
		return err
	}

	id := c.Var("id")

	err := s.provider(c).WithContext(c.Context()).CertificateDelete(id)
	if err != nil {
		return err
	}

	return c.RenderOK()
}

func (s *Server) CertificateGenerate(c *stdapi.Context) error {
	if err := s.hook("CertificateGenerateValidate", c); err != nil {
		return err
	}

	domains := strings.Split(c.Value("domains"), ",")

	v, err := s.provider(c).WithContext(c.Context()).CertificateGenerate(domains)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) CertificateList(c *stdapi.Context) error {
	if err := s.hook("CertificateListValidate", c); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).CertificateList()
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) EventSend(c *stdapi.Context) error {
	if err := s.hook("EventSendValidate", c); err != nil {
		return err
	}

	action := c.Value("action")

	var opts structs.EventSendOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	err := s.provider(c).WithContext(c.Context()).EventSend(action, opts)
	if err != nil {
		return err
	}

	return c.RenderOK()
}

func (s *Server) FilesDelete(c *stdapi.Context) error {
	if err := s.hook("FilesDeleteValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	pid := c.Var("pid")
	files := strings.Split(c.Value("files"), ",")

	err := s.provider(c).WithContext(c.Context()).FilesDelete(app, pid, files)
	if err != nil {
		return err
	}

	return c.RenderOK()
}

func (s *Server) FilesDownload(c *stdapi.Context) error {
	if err := s.hook("FilesDownloadValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	pid := c.Var("pid")
	file := c.Value("file")

	v, err := s.provider(c).WithContext(c.Context()).FilesDownload(app, pid, file)
	if err != nil {
		return err
	}

	if c, ok := interface{}(v).(io.Closer); ok {
		defer c.Close()
	}

	if _, err := io.Copy(c, v); err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return nil
}

func (s *Server) FilesUpload(c *stdapi.Context) error {
	if err := s.hook("FilesUploadValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	pid := c.Var("pid")
	r := c

	err := s.provider(c).WithContext(c.Context()).FilesUpload(app, pid, r)
	if err != nil {
		return err
	}

	return c.RenderOK()
}

func (s *Server) Initialize(c *stdapi.Context) error {
	return stdapi.Errorf(404, "not available via api")
}

func (s *Server) InstanceKeyroll(c *stdapi.Context) error {
	if err := s.hook("InstanceKeyrollValidate", c); err != nil {
		return err
	}

	err := s.provider(c).WithContext(c.Context()).InstanceKeyroll()
	if err != nil {
		return err
	}

	return c.RenderOK()
}

func (s *Server) InstanceList(c *stdapi.Context) error {
	if err := s.hook("InstanceListValidate", c); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).InstanceList()
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) InstanceShell(c *stdapi.Context) error {
	if err := s.hook("InstanceShellValidate", c); err != nil {
		return err
	}

	id := c.Var("id")
	rw := c

	var opts structs.InstanceShellOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).InstanceShell(id, rw, opts)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return renderStatusCode(c, v)
}

func (s *Server) InstanceTerminate(c *stdapi.Context) error {
	if err := s.hook("InstanceTerminateValidate", c); err != nil {
		return err
	}

	id := c.Var("id")

	err := s.provider(c).WithContext(c.Context()).InstanceTerminate(id)
	if err != nil {
		return err
	}

	return c.RenderOK()
}

func (s *Server) ObjectDelete(c *stdapi.Context) error {
	if err := s.hook("ObjectDeleteValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	key := c.Var("key")

	err := s.provider(c).WithContext(c.Context()).ObjectDelete(app, key)
	if err != nil {
		return err
	}

	return c.RenderOK()
}

func (s *Server) ObjectExists(c *stdapi.Context) error {
	if err := s.hook("ObjectExistsValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	key := c.Var("key")

	v, err := s.provider(c).WithContext(c.Context()).ObjectExists(app, key)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) ObjectFetch(c *stdapi.Context) error {
	if err := s.hook("ObjectFetchValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	key := c.Var("key")

	v, err := s.provider(c).WithContext(c.Context()).ObjectFetch(app, key)
	if err != nil {
		return err
	}

	if c, ok := interface{}(v).(io.Closer); ok {
		defer c.Close()
	}

	if _, err := io.Copy(c, v); err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return nil
}

func (s *Server) ObjectList(c *stdapi.Context) error {
	if err := s.hook("ObjectListValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	prefix := c.Value("prefix")

	v, err := s.provider(c).WithContext(c.Context()).ObjectList(app, prefix)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) ObjectStore(c *stdapi.Context) error {
	if err := s.hook("ObjectStoreValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	key := c.Var("key")
	r := c

	var opts structs.ObjectStoreOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).ObjectStore(app, key, r, opts)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) ProcessExec(c *stdapi.Context) error {
	if err := s.hook("ProcessExecValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	pid := c.Var("pid")
	command := c.Value("command")
	rw := c

	var opts structs.ProcessExecOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).ProcessExec(app, pid, command, rw, opts)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return renderStatusCode(c, v)
}

func (s *Server) ProcessGet(c *stdapi.Context) error {
	if err := s.hook("ProcessGetValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	pid := c.Var("pid")

	v, err := s.provider(c).WithContext(c.Context()).ProcessGet(app, pid)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) ProcessList(c *stdapi.Context) error {
	if err := s.hook("ProcessListValidate", c); err != nil {
		return err
	}

	app := c.Var("app")

	var opts structs.ProcessListOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).ProcessList(app, opts)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) ProcessLogs(c *stdapi.Context) error {
	if err := s.hook("ProcessLogsValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	pid := c.Var("pid")

	var opts structs.LogsOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).ProcessLogs(app, pid, opts)
	if err != nil {
		return err
	}

	if c, ok := interface{}(v).(io.Closer); ok {
		defer c.Close()
	}

	if _, err := io.Copy(c, v); err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return nil
}

func (s *Server) ProcessRun(c *stdapi.Context) error {
	if err := s.hook("ProcessRunValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	service := c.Var("service")

	var opts structs.ProcessRunOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).ProcessRun(app, service, opts)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) ProcessStop(c *stdapi.Context) error {
	if err := s.hook("ProcessStopValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	pid := c.Var("pid")

	err := s.provider(c).WithContext(c.Context()).ProcessStop(app, pid)
	if err != nil {
		return err
	}

	return c.RenderOK()
}

func (s *Server) Proxy(c *stdapi.Context) error {
	if err := s.hook("ProxyValidate", c); err != nil {
		return err
	}

	host := c.Var("host")
	rw := c

	port, cerr := strconv.Atoi(c.Var("port"))
	if cerr != nil {
		return cerr
	}

	var opts structs.ProxyOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	err := s.provider(c).WithContext(c.Context()).Proxy(host, port, rw, opts)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) RegistryAdd(c *stdapi.Context) error {
	if err := s.hook("RegistryAddValidate", c); err != nil {
		return err
	}

	server := c.Value("server")
	username := c.Value("username")
	password := c.Value("password")

	v, err := s.provider(c).WithContext(c.Context()).RegistryAdd(server, username, password)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) RegistryList(c *stdapi.Context) error {
	if err := s.hook("RegistryListValidate", c); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).RegistryList()
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) RegistryRemove(c *stdapi.Context) error {
	if err := s.hook("RegistryRemoveValidate", c); err != nil {
		return err
	}

	server := c.Var("server")

	err := s.provider(c).WithContext(c.Context()).RegistryRemove(server)
	if err != nil {
		return err
	}

	return c.RenderOK()
}

func (s *Server) ReleaseCreate(c *stdapi.Context) error {
	if err := s.hook("ReleaseCreateValidate", c); err != nil {
		return err
	}

	app := c.Var("app")

	var opts structs.ReleaseCreateOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).ReleaseCreate(app, opts)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) ReleaseGet(c *stdapi.Context) error {
	if err := s.hook("ReleaseGetValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	id := c.Var("id")

	v, err := s.provider(c).WithContext(c.Context()).ReleaseGet(app, id)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) ReleaseList(c *stdapi.Context) error {
	if err := s.hook("ReleaseListValidate", c); err != nil {
		return err
	}

	app := c.Var("app")

	var opts structs.ReleaseListOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).ReleaseList(app, opts)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) ReleasePromote(c *stdapi.Context) error {
	if err := s.hook("ReleasePromoteValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	id := c.Var("id")

	var opts structs.ReleasePromoteOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	err := s.provider(c).WithContext(c.Context()).ReleasePromote(app, id, opts)
	if err != nil {
		return err
	}

	return c.RenderOK()
}

func (s *Server) ResourceGet(c *stdapi.Context) error {
	if err := s.hook("ResourceGetValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	name := c.Var("name")

	v, err := s.provider(c).WithContext(c.Context()).ResourceGet(app, name)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) ResourceList(c *stdapi.Context) error {
	if err := s.hook("ResourceListValidate", c); err != nil {
		return err
	}

	app := c.Var("app")

	v, err := s.provider(c).WithContext(c.Context()).ResourceList(app)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) ServiceList(c *stdapi.Context) error {
	if err := s.hook("ServiceListValidate", c); err != nil {
		return err
	}

	app := c.Var("app")

	v, err := s.provider(c).WithContext(c.Context()).ServiceList(app)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) ServiceRestart(c *stdapi.Context) error {
	if err := s.hook("ServiceRestartValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	name := c.Var("name")

	err := s.provider(c).WithContext(c.Context()).ServiceRestart(app, name)
	if err != nil {
		return err
	}

	return c.RenderOK()
}

func (s *Server) ServiceUpdate(c *stdapi.Context) error {
	if err := s.hook("ServiceUpdateValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	name := c.Var("name")

	var opts structs.ServiceUpdateOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	err := s.provider(c).WithContext(c.Context()).ServiceUpdate(app, name, opts)
	if err != nil {
		return err
	}

	return c.RenderOK()
}

func (s *Server) SystemGet(c *stdapi.Context) error {
	if err := s.hook("SystemGetValidate", c); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).SystemGet()
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) SystemInstall(c *stdapi.Context) error {
	return stdapi.Errorf(404, "not available via api")
}

func (s *Server) SystemLogs(c *stdapi.Context) error {
	if err := s.hook("SystemLogsValidate", c); err != nil {
		return err
	}

	var opts structs.LogsOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).SystemLogs(opts)
	if err != nil {
		return err
	}

	if c, ok := interface{}(v).(io.Closer); ok {
		defer c.Close()
	}

	if _, err := io.Copy(c, v); err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return nil
}

func (s *Server) SystemMetrics(c *stdapi.Context) error {
	if err := s.hook("SystemMetricsValidate", c); err != nil {
		return err
	}

	var opts structs.MetricsOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).SystemMetrics(opts)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) SystemProcesses(c *stdapi.Context) error {
	if err := s.hook("SystemProcessesValidate", c); err != nil {
		return err
	}

	var opts structs.SystemProcessesOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).SystemProcesses(opts)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) SystemReleases(c *stdapi.Context) error {
	if err := s.hook("SystemReleasesValidate", c); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).SystemReleases()
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) SystemResourceCreate(c *stdapi.Context) error {
	if err := s.hook("SystemResourceCreateValidate", c); err != nil {
		return err
	}

	kind := c.Value("kind")

	var opts structs.ResourceCreateOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).SystemResourceCreate(kind, opts)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) SystemResourceDelete(c *stdapi.Context) error {
	if err := s.hook("SystemResourceDeleteValidate", c); err != nil {
		return err
	}

	name := c.Var("name")

	err := s.provider(c).WithContext(c.Context()).SystemResourceDelete(name)
	if err != nil {
		return err
	}

	return c.RenderOK()
}

func (s *Server) SystemResourceGet(c *stdapi.Context) error {
	if err := s.hook("SystemResourceGetValidate", c); err != nil {
		return err
	}

	name := c.Var("name")

	v, err := s.provider(c).WithContext(c.Context()).SystemResourceGet(name)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) SystemResourceLink(c *stdapi.Context) error {
	if err := s.hook("SystemResourceLinkValidate", c); err != nil {
		return err
	}

	name := c.Var("name")
	app := c.Value("app")

	v, err := s.provider(c).WithContext(c.Context()).SystemResourceLink(name, app)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) SystemResourceList(c *stdapi.Context) error {
	if err := s.hook("SystemResourceListValidate", c); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).SystemResourceList()
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) SystemResourceTypes(c *stdapi.Context) error {
	if err := s.hook("SystemResourceTypesValidate", c); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).SystemResourceTypes()
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) SystemResourceUnlink(c *stdapi.Context) error {
	if err := s.hook("SystemResourceUnlinkValidate", c); err != nil {
		return err
	}

	name := c.Var("name")
	app := c.Var("app")

	v, err := s.provider(c).WithContext(c.Context()).SystemResourceUnlink(name, app)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) SystemResourceUpdate(c *stdapi.Context) error {
	if err := s.hook("SystemResourceUpdateValidate", c); err != nil {
		return err
	}

	name := c.Var("name")

	var opts structs.ResourceUpdateOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).WithContext(c.Context()).SystemResourceUpdate(name, opts)
	if err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) SystemUninstall(c *stdapi.Context) error {
	return stdapi.Errorf(404, "not available via api")
}

func (s *Server) SystemUpdate(c *stdapi.Context) error {
	if err := s.hook("SystemUpdateValidate", c); err != nil {
		return err
	}

	var opts structs.SystemUpdateOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	err := s.provider(c).WithContext(c.Context()).SystemUpdate(opts)
	if err != nil {
		return err
	}

	return c.RenderOK()
}

func (s *Server) Workers(c *stdapi.Context) error {
	return stdapi.Errorf(404, "not available via api")
}

