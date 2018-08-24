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

	err := s.provider(c).AppCancel(name)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).AppCreate(name, opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	err := s.provider(c).AppDelete(name)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	return c.RenderOK()
}

func (s *Server) AppGet(c *stdapi.Context) error {
	if err := s.hook("AppGetValidate", c); err != nil {
		return err
	}

	name := c.Var("name")

	v, err := s.provider(c).AppGet(name)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).AppList()
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).AppLogs(name, opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	if _, err := io.Copy(c, v); err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return nil
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

	err := s.provider(c).AppUpdate(name, opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).BuildCreate(app, url, opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	err := s.provider(c).BuildExport(app, id, w)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	return c.RenderOK()
}

func (s *Server) BuildGet(c *stdapi.Context) error {
	if err := s.hook("BuildGetValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	id := c.Var("id")

	v, err := s.provider(c).BuildGet(app, id)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).BuildImport(app, r)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).BuildList(app, opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).BuildLogs(app, id, opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
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

	v, err := s.provider(c).BuildUpdate(app, id, opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).CapacityGet()
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	err := s.provider(c).CertificateApply(app, service, port, id)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).CertificateCreate(pub, key, opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	err := s.provider(c).CertificateDelete(id)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	return c.RenderOK()
}

func (s *Server) CertificateGenerate(c *stdapi.Context) error {
	if err := s.hook("CertificateGenerateValidate", c); err != nil {
		return err
	}

	domains := strings.Split(c.Value("domains"), ",")

	v, err := s.provider(c).CertificateGenerate(domains)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).CertificateList()
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) EventSend(c *stdapi.Context) error {
	return stdapi.Errorf(404, "not available via api")
}

func (s *Server) FilesDelete(c *stdapi.Context) error {
	if err := s.hook("FilesDeleteValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	pid := c.Var("pid")
	files := strings.Split(c.Value("files"), ",")

	err := s.provider(c).FilesDelete(app, pid, files)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	return c.RenderOK()
}

func (s *Server) FilesUpload(c *stdapi.Context) error {
	if err := s.hook("FilesUploadValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	pid := c.Var("pid")
	r := c

	err := s.provider(c).FilesUpload(app, pid, r)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	err := s.provider(c).InstanceKeyroll()
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	return c.RenderOK()
}

func (s *Server) InstanceList(c *stdapi.Context) error {
	if err := s.hook("InstanceListValidate", c); err != nil {
		return err
	}

	v, err := s.provider(c).InstanceList()
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).InstanceShell(id, rw, opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	err := s.provider(c).InstanceTerminate(id)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	err := s.provider(c).ObjectDelete(app, key)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).ObjectExists(app, key)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).ObjectFetch(app, key)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
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

	v, err := s.provider(c).ObjectList(app, prefix)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).ObjectStore(app, key, r, opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).ProcessExec(app, pid, command, rw, opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).ProcessGet(app, pid)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).ProcessList(app, opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
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

	v, err := s.provider(c).ProcessRun(app, service, opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	err := s.provider(c).ProcessStop(app, pid)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	return c.RenderOK()
}

func (s *Server) ProcessWait(c *stdapi.Context) error {
	if err := s.hook("ProcessWaitValidate", c); err != nil {
		return err
	}

	app := c.Var("app")
	pid := c.Var("pid")

	v, err := s.provider(c).ProcessWait(app, pid)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return renderStatusCode(c, v)
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

	err := s.provider(c).Proxy(host, port, rw)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	return c.RenderOK()
}

func (s *Server) RegistryAdd(c *stdapi.Context) error {
	if err := s.hook("RegistryAddValidate", c); err != nil {
		return err
	}

	server := c.Value("server")
	username := c.Value("username")
	password := c.Value("password")

	v, err := s.provider(c).RegistryAdd(server, username, password)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).RegistryList()
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	err := s.provider(c).RegistryRemove(server)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).ReleaseCreate(app, opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).ReleaseGet(app, id)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).ReleaseList(app, opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	err := s.provider(c).ReleasePromote(app, id)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	return c.RenderOK()
}

func (s *Server) ResourceCreate(c *stdapi.Context) error {
	if err := s.hook("ResourceCreateValidate", c); err != nil {
		return err
	}

	kind := c.Value("kind")

	var opts structs.ResourceCreateOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).ResourceCreate(kind, opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) ResourceDelete(c *stdapi.Context) error {
	if err := s.hook("ResourceDeleteValidate", c); err != nil {
		return err
	}

	name := c.Var("name")

	err := s.provider(c).ResourceDelete(name)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	return c.RenderOK()
}

func (s *Server) ResourceGet(c *stdapi.Context) error {
	if err := s.hook("ResourceGetValidate", c); err != nil {
		return err
	}

	name := c.Var("name")

	v, err := s.provider(c).ResourceGet(name)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) ResourceLink(c *stdapi.Context) error {
	if err := s.hook("ResourceLinkValidate", c); err != nil {
		return err
	}

	name := c.Var("name")
	app := c.Value("app")

	v, err := s.provider(c).ResourceLink(name, app)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).ResourceList()
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) ResourceTypes(c *stdapi.Context) error {
	if err := s.hook("ResourceTypesValidate", c); err != nil {
		return err
	}

	v, err := s.provider(c).ResourceTypes()
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) ResourceUnlink(c *stdapi.Context) error {
	if err := s.hook("ResourceUnlinkValidate", c); err != nil {
		return err
	}

	name := c.Var("name")
	app := c.Var("app")

	v, err := s.provider(c).ResourceUnlink(name, app)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
}

func (s *Server) ResourceUpdate(c *stdapi.Context) error {
	if err := s.hook("ResourceUpdateValidate", c); err != nil {
		return err
	}

	name := c.Var("name")

	var opts structs.ResourceUpdateOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).ResourceUpdate(name, opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).ServiceList(app)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return c.RenderJSON(v)
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

	err := s.provider(c).ServiceUpdate(app, name, opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	return c.RenderOK()
}

func (s *Server) SystemGet(c *stdapi.Context) error {
	if err := s.hook("SystemGetValidate", c); err != nil {
		return err
	}

	v, err := s.provider(c).SystemGet()
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).SystemLogs(opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	if _, err := io.Copy(c, v); err != nil {
		return err
	}

	if vs, ok := interface{}(v).(Sortable); ok {
		sort.Slice(v, vs.Less)
	}

	return nil
}

func (s *Server) SystemProcesses(c *stdapi.Context) error {
	if err := s.hook("SystemProcessesValidate", c); err != nil {
		return err
	}

	var opts structs.SystemProcessesOptions
	if err := stdapi.UnmarshalOptions(c.Request(), &opts); err != nil {
		return err
	}

	v, err := s.provider(c).SystemProcesses(opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	v, err := s.provider(c).SystemReleases()
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
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

	err := s.provider(c).SystemUpdate(opts)
	if err != nil {
		if ae, ok := s.provider(c).(ApiErrorer); ok {
			return ae.ApiError(err)
		}
		return err
	}

	return c.RenderOK()
}

func (s *Server) Workers(c *stdapi.Context) error {
	return stdapi.Errorf(404, "not available via api")
}

