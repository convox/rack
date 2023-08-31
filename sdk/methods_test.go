package sdk_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdapi"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

const statusCodePrefix = "F1E49A85-0AD7-4AEF-A618-C249C6E6568D:"

func TestAppCancel(t *testing.T) {
	app := "app1"

	s := stdapi.New("api", "api")
	s.Route("POST", fmt.Sprintf("/apps/%s/cancel", app), func(c *stdapi.Context) error {
		return c.RenderOK()
	})

	testServer(t, s, func(c *sdk.Client) {
		err := c.AppCancel(app)
		require.NoError(t, err)
	})
}

func TestAppCreate(t *testing.T) {
	a := &structs.App{
		Name:    "app1",
		Release: "r1234",
	}

	s := stdapi.New("api", "api")
	s.Route("POST", "/apps", func(c *stdapi.Context) error {
		require.Equal(t, a.Name, c.Form("name"))
		require.Equal(t, "2", c.Form("generation"))
		return c.RenderJSON(a)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.AppCreate(a.Name, structs.AppCreateOptions{
			Generation: options.String("2"),
		})
		require.NoError(t, err)
		require.Equal(t, a, got)
	})
}

func TestAppDelete(t *testing.T) {
	app := "app1"

	s := stdapi.New("api", "api")
	s.Route("DELETE", fmt.Sprintf("/apps/%s", app), func(c *stdapi.Context) error {
		return c.RenderOK()
	})

	testServer(t, s, func(c *sdk.Client) {
		err := c.AppDelete(app)
		require.NoError(t, err)
	})
}

func TestAppGet(t *testing.T) {
	a := &structs.App{
		Name:    "app1",
		Release: "r1234",
	}

	s := stdapi.New("api", "api")
	s.Route("GET", fmt.Sprintf("/apps/%s", a.Name), func(c *stdapi.Context) error {
		return c.RenderJSON(a)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.AppGet(a.Name)
		require.NoError(t, err)
		require.Equal(t, a, got)
	})
}

func TestAppList(t *testing.T) {
	as := structs.Apps{
		{
			Name:    "app1",
			Release: "r1234",
		},
		{
			Name:    "app2",
			Release: "r2",
		},
	}

	s := stdapi.New("api", "api")
	s.Route("GET", "/apps", func(c *stdapi.Context) error {
		return c.RenderJSON(as)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.AppList()
		require.NoError(t, err)
		require.Equal(t, as, got)
	})
}

func TestAppLogs(t *testing.T) {
	name := "app1"
	s := stdapi.New("api", "api")
	s.Route("GET", "/racks", func(c *stdapi.Context) error {
		return c.RenderOK()
	})
	s.Route("SOCKET", fmt.Sprintf("/apps/%s/logs", name), func(c *stdapi.Context) error {
		require.Equal(t, "test", c.Header("Filter"))
		return c.Websocket().WriteMessage(websocket.TextMessage, []byte("data"))
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.AppLogs(name, structs.LogsOptions{
			Filter: options.String("test"),
		})
		require.NoError(t, err)
		b, _ := io.ReadAll(got)
		require.Equal(t, "data", string(b))
	})
}

func TestAppMetrics(t *testing.T) {
	app := "app1"
	startTime := time.Now().UTC()
	ms := structs.Metrics{
		{
			Name: "test1",
			Values: structs.MetricValues{
				{
					Average: 2.3,
					Count:   3,
					Maximum: 5,
					Minimum: 2,
				},
				{
					Average: 2.3,
					Count:   3,
					Maximum: 5,
					Minimum: 2,
				},
			},
		},
	}

	s := stdapi.New("api", "api")
	s.Route("GET", fmt.Sprintf("/apps/%s/metrics", app), func(c *stdapi.Context) error {
		require.Equal(t, "120", c.Query("period"))
		require.Equal(t, startTime.Format("20060102.150405.000000000"), c.Query("start"))
		return c.RenderJSON(ms)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.AppMetrics(app, structs.MetricsOptions{
			Period: options.Int64(120),
			Start:  options.Time(startTime),
		})
		require.NoError(t, err)
		require.Equal(t, ms, got)
	})
}

func TestAppUpdate(t *testing.T) {
	a := &structs.App{
		Name:    "app1",
		Release: "r1234",
	}
	params := map[string]string{
		"hello": "world",
		"then":  "what",
	}

	s := stdapi.New("api", "api")
	s.Route("PUT", fmt.Sprintf("/apps/%s", a.Name), func(c *stdapi.Context) error {
		require.Equal(t, "true", c.Form("lock"))
		require.Equal(t, "hello=world&then=what", c.Form("parameters"))
		return c.RenderOK()
	})

	testServer(t, s, func(c *sdk.Client) {
		err := c.AppUpdate(a.Name, structs.AppUpdateOptions{
			Lock:       options.Bool(true),
			Parameters: params,
		})
		require.NoError(t, err)
	})
}

func TestBuildCreate(t *testing.T) {
	b := &structs.Build{
		App:         "app1",
		Description: "d1",
		Entrypoint:  "",
		Logs:        "",
		Manifest:    "convox.yml",
		Process:     "ps1",
		Release:     "r1234",
		Status:      "pending",
	}

	s := stdapi.New("api", "api")
	s.Route("POST", fmt.Sprintf("/apps/%s/builds", b.App), func(c *stdapi.Context) error {
		require.Equal(t, "url", c.Form("url"))
		require.Equal(t, b.Description, c.Form("description"))
		require.Equal(t, b.Manifest, c.Form("manifest"))
		require.Equal(t, "true", c.Form("no-cache"))
		require.Equal(t, "true", c.Form("wildcard-domain"))
		return c.RenderJSON(b)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.BuildCreate(b.App, "url", structs.BuildCreateOptions{
			Description:    options.String(b.Description),
			Manifest:       options.String(b.Manifest),
			NoCache:        options.Bool(true),
			WildcardDomain: options.Bool(true),
		})
		require.NoError(t, err)
		require.Equal(t, b, got)
	})
}

func TestBuildExport(t *testing.T) {
	app := "app1"
	id := "id1"

	s := stdapi.New("api", "api")
	s.Route("GET", fmt.Sprintf("/apps/%s/builds/%s.tgz", app, id), func(c *stdapi.Context) error {
		_, err := c.Write([]byte("test"))
		return err
	})

	testServer(t, s, func(c *sdk.Client) {
		var b bytes.Buffer
		err := c.BuildExport(app, id, &b)
		require.NoError(t, err)
		require.Equal(t, "test", b.String())
	})
}

func TestBuildGet(t *testing.T) {
	b := &structs.Build{
		App:         "app1",
		Description: "d1",
		Entrypoint:  "",
		Logs:        "",
		Manifest:    "convox.yml",
		Process:     "ps1",
		Release:     "r1234",
		Status:      "pending",
	}

	s := stdapi.New("api", "api")
	s.Route("GET", fmt.Sprintf("/apps/%s/builds/%s", b.App, "id1"), func(c *stdapi.Context) error {
		return c.RenderJSON(b)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.BuildGet(b.App, "id1")
		require.NoError(t, err)
		require.Equal(t, b, got)
	})
}

func TestBuildImport(t *testing.T) {
	bld := &structs.Build{
		App:         "app1",
		Description: "d1",
		Entrypoint:  "",
		Logs:        "",
		Manifest:    "convox.yml",
		Process:     "ps1",
		Release:     "r1234",
		Status:      "pending",
	}

	s := stdapi.New("api", "api")
	s.Route("POST", fmt.Sprintf("/apps/%s/builds/import", bld.App), func(c *stdapi.Context) error {
		b, err := io.ReadAll(c.Request().Body)
		require.NoError(t, err)
		require.Equal(t, "test", string(b))
		return c.RenderJSON(bld)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.BuildImport(bld.App, bytes.NewReader([]byte("test")))
		require.NoError(t, err)
		require.Equal(t, bld, got)
	})
}

func TestBuildList(t *testing.T) {
	bs := structs.Builds{
		{
			App:         "app1",
			Description: "d1",
			Entrypoint:  "",
			Logs:        "",
			Manifest:    "convox.yml",
			Process:     "ps1",
			Release:     "r1234",
			Status:      "pending",
		},
	}

	s := stdapi.New("api", "api")
	s.Route("GET", fmt.Sprintf("/apps/%s/builds", bs[0].App), func(c *stdapi.Context) error {
		return c.RenderJSON(bs)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.BuildList(bs[0].App, structs.BuildListOptions{})
		require.NoError(t, err)
		require.Equal(t, bs, got)
	})
}

func TestBuildLogs(t *testing.T) {
	app := "app1"
	id := "id1"
	s := stdapi.New("api", "api")
	s.Route("GET", "/racks", func(c *stdapi.Context) error {
		return c.RenderOK()
	})
	s.Route("SOCKET", fmt.Sprintf("/apps/%s/builds/%s/logs", app, id), func(c *stdapi.Context) error {
		require.Equal(t, "test", c.Header("Filter"))
		return c.Websocket().WriteMessage(websocket.TextMessage, []byte("data"))
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.BuildLogs(app, id, structs.LogsOptions{
			Filter: options.String("test"),
		})
		require.NoError(t, err)
		b, _ := io.ReadAll(got)
		require.Equal(t, "data", string(b))
	})
}

func TestBuildUpdate(t *testing.T) {
	b := &structs.Build{
		App:         "app1",
		Description: "d1",
		Entrypoint:  "start.sh",
		Logs:        "logs",
		Manifest:    "convox.yml",
		Process:     "ps1",
		Release:     "r1234",
		Status:      "pending",
	}

	s := stdapi.New("api", "api")
	s.Route("PUT", fmt.Sprintf("/apps/%s/builds/%s", b.App, "id1"), func(c *stdapi.Context) error {
		require.Equal(t, b.Entrypoint, c.Form("entrypoint"))
		require.Equal(t, b.Logs, c.Form("logs"))
		require.Equal(t, b.Release, c.Form("release"))
		return c.RenderJSON(b)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.BuildUpdate(b.App, "id1", structs.BuildUpdateOptions{
			Entrypoint: options.String(b.Entrypoint),
			Logs:       options.String(b.Logs),
			Release:    options.String(b.Release),
		})
		require.NoError(t, err)
		require.Equal(t, b, got)
	})
}

func TestCapacityGet(t *testing.T) {
	cp := &structs.Capacity{
		ClusterCPU:     5,
		ClusterMemory:  1024,
		InstanceCPU:    4,
		InstanceMemory: 1024,
		ProcessCount:   5,
		ProcessCPU:     2,
		ProcessMemory:  128,
		ProcessWidth:   3,
	}

	s := stdapi.New("api", "api")
	s.Route("GET", "/system/capacity", func(c *stdapi.Context) error {
		return c.RenderJSON(cp)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.CapacityGet()
		require.NoError(t, err)
		require.Equal(t, cp, got)
	})
}

func TestCertificateApply(t *testing.T) {
	app := "app1"
	service := "web"
	port := 3000
	s := stdapi.New("api", "api")
	s.Route("PUT", fmt.Sprintf("/apps/%s/ssl/%s/%d", app, service, port), func(c *stdapi.Context) error {
		require.Equal(t, "id1", c.Form("id"))
		return c.RenderOK()
	})

	testServer(t, s, func(c *sdk.Client) {
		err := c.CertificateApply(app, service, port, "id1")
		require.NoError(t, err)
	})
}

func TestCertificateCreate(t *testing.T) {
	certResp := &structs.Certificate{
		Id:         "1",
		Domain:     "convox.com",
		Expiration: time.Now().UTC(),
	}

	s := stdapi.New("api", "api")
	s.Route("POST", "/certificates", func(c *stdapi.Context) error {
		require.Equal(t, "chain", c.Form("chain"))
		require.Equal(t, "public", c.Form("pub"))
		require.Equal(t, "private", c.Form("key"))
		return c.RenderJSON(certResp)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.CertificateCreate("public", "private", structs.CertificateCreateOptions{
			Chain: options.String("chain"),
		})
		require.NoError(t, err)
		require.Equal(t, certResp, got)
	})
}

func TestCertificateDelete(t *testing.T) {
	id := "id1"
	s := stdapi.New("api", "api")
	s.Route("DELETE", fmt.Sprintf("/certificates/%s", id), func(c *stdapi.Context) error {
		return c.RenderOK()
	})

	testServer(t, s, func(c *sdk.Client) {
		err := c.CertificateDelete(id)
		require.NoError(t, err)
	})
}

func TestCertificateGenerate(t *testing.T) {
	certResp := &structs.Certificate{
		Id:         "1",
		Domain:     "convox.com",
		Expiration: time.Now().UTC(),
	}

	s := stdapi.New("api", "api")
	s.Route("POST", "/certificates/generate", func(c *stdapi.Context) error {
		require.Equal(t, "convox.xom", c.Form("domains"))
		return c.RenderJSON(certResp)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.CertificateGenerate([]string{"convox.xom"})
		require.NoError(t, err)
		require.Equal(t, certResp, got)
	})
}

func TestCertificateList(t *testing.T) {
	certResp := structs.Certificates{{
		Id:         "1",
		Domain:     "convox.com",
		Expiration: time.Now().UTC(),
	}}

	s := stdapi.New("api", "api")
	s.Route("GET", "/certificates", func(c *stdapi.Context) error {
		return c.RenderJSON(certResp)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.CertificateList()
		require.NoError(t, err)
		require.Equal(t, certResp, got)
	})
}

func TestEventSend(t *testing.T) {
	action := "delete"
	opts := structs.EventSendOptions{
		Data: map[string]string{
			"hello": "world",
		},
		Error:  options.String("error"),
		Status: options.String("running"),
	}
	s := stdapi.New("api", "api")
	s.Route("POST", "/events", func(c *stdapi.Context) error {
		require.Equal(t, "hello=world", c.Form("data"))
		require.Equal(t, *opts.Error, c.Form("error"))
		require.Equal(t, *opts.Status, c.Form("status"))
		return c.RenderOK()
	})

	testServer(t, s, func(c *sdk.Client) {
		err := c.EventSend(action, opts)
		require.NoError(t, err)
	})
}

func TestFilesDelete(t *testing.T) {
	app := "app1"
	pid := "pid1"
	files := []string{"a.txt", "b.txt"}

	s := stdapi.New("api", "api")
	s.Route("DELETE", fmt.Sprintf("/apps/%s/processes/%s/files", app, pid), func(c *stdapi.Context) error {
		require.Equal(t, strings.Join(files, ","), c.Form("files"))
		return c.RenderOK()
	})

	testServer(t, s, func(c *sdk.Client) {
		err := c.FilesDelete(app, pid, files)
		require.NoError(t, err)
	})
}

func TestFilesDownload(t *testing.T) {
	app := "app1"
	pid := "pid1"
	file := "a.txt"

	s := stdapi.New("api", "api")
	s.Route("GET", fmt.Sprintf("/apps/%s/processes/%s/files", app, pid), func(c *stdapi.Context) error {
		require.Equal(t, file, c.Query("file"))
		_, err := c.Write([]byte("test"))
		return err
	})

	testServer(t, s, func(c *sdk.Client) {
		r, err := c.FilesDownload(app, pid, file)
		require.NoError(t, err)
		b, err := io.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, "test", string(b))
	})
}

func TestFilesUpload(t *testing.T) {
	app := "app1"
	pid := "pid1"

	s := stdapi.New("api", "api")
	s.Route("POST", fmt.Sprintf("/apps/%s/processes/%s/files", app, pid), func(c *stdapi.Context) error {
		b, err := io.ReadAll(c.Request().Body)
		require.NoError(t, err)
		require.Equal(t, "test", string(b))
		return c.RenderOK()
	})

	testServer(t, s, func(c *sdk.Client) {
		err := c.FilesUpload(app, pid, bytes.NewReader([]byte("test")))
		require.NoError(t, err)

	})
}

func TestInstanceList(t *testing.T) {
	is := structs.Instances{{
		Agent:     true,
		Cpu:       2,
		Id:        "1",
		Memory:    128,
		PrivateIp: "0.0.0.0",
		Processes: 10,
		PublicIp:  "1.0.0.0",
		Status:    "running",
	}}

	s := stdapi.New("api", "api")
	s.Route("GET", "/instances", func(c *stdapi.Context) error {
		return c.RenderJSON(is)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.InstanceList()
		require.NoError(t, err)
		require.Equal(t, is, got)
	})
}

func TestInstanceShell(t *testing.T) {
	id := "id1"
	s := stdapi.New("api", "api")
	s.Route("SOCKET", fmt.Sprintf("/instances/%s/shell", id), func(c *stdapi.Context) error {
		require.Equal(t, "test", c.Header("Command"))
		require.Equal(t, "1", c.Header("Height"))
		require.Equal(t, "2", c.Header("Width"))
		_, b, err := c.Websocket().ReadMessage()
		require.NoError(t, err)
		require.Equal(t, "cmd", string(b))
		return c.Websocket().WriteMessage(websocket.TextMessage, []byte("data"+statusCodePrefix+"0"))
	})

	testServer(t, s, func(c *sdk.Client) {
		rw := bufio.NewReadWriter(bufio.NewReader(bytes.NewReader([]byte("cmd"))), bufio.NewWriter(&bytes.Buffer{}))
		got, err := c.InstanceShell(id, rw, structs.InstanceShellOptions{
			Command: options.String("test"),
			Height:  options.Int(1),
			Width:   options.Int(2),
		})
		require.NoError(t, err)
		require.Equal(t, 0, got)
	})
}

func TestInstanceTerminate(t *testing.T) {
	id := "pid1"

	s := stdapi.New("api", "api")
	s.Route("DELETE", fmt.Sprintf("/instances/%s", id), func(c *stdapi.Context) error {
		return c.RenderOK()
	})

	testServer(t, s, func(c *sdk.Client) {
		err := c.InstanceTerminate(id)
		require.NoError(t, err)
	})
}

func TestObjectExists(t *testing.T) {
	app := "app1"
	key := "key1"

	s := stdapi.New("api", "api")
	s.Route("HEAD", fmt.Sprintf("/apps/%s/objects/%s", app, key), func(c *stdapi.Context) error {
		_, err := c.Write([]byte("true"))
		return err
	})

	testServer(t, s, func(c *sdk.Client) {
		v, err := c.ObjectExists(app, key)
		require.NoError(t, err)
		require.Equal(t, true, v)
	})
}

func TestObjectFetch(t *testing.T) {
	app := "app1"
	key := "key1"

	s := stdapi.New("api", "api")
	s.Route("GET", fmt.Sprintf("/apps/%s/objects/%s", app, key), func(c *stdapi.Context) error {
		_, err := c.Write([]byte("test"))
		return err
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.ObjectFetch(app, key)
		require.NoError(t, err)
		b, _ := io.ReadAll(got)
		require.Equal(t, "test", string(b))
	})
}

func TestObjectList(t *testing.T) {
	app := "app1"
	objs := []string{"obj1", "obj2"}

	s := stdapi.New("api", "api")
	s.Route("GET", fmt.Sprintf("/apps/%s/objects", app), func(c *stdapi.Context) error {
		require.Equal(t, "ob", c.Query("prefix"))
		return c.RenderJSON(objs)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.ObjectList(app, "ob")
		require.NoError(t, err)
		require.Equal(t, objs, got)
	})
}

func TestProcessExec(t *testing.T) {
	app := "app1"
	id := "id1"
	s := stdapi.New("api", "api")
	s.Route("SOCKET", fmt.Sprintf("/apps/%s/processes/%s/exec", app, id), func(c *stdapi.Context) error {
		require.Equal(t, "true", c.Header("Entrypoint"))
		require.Equal(t, "1", c.Header("Height"))
		require.Equal(t, "2", c.Header("Width"))
		return c.Websocket().WriteMessage(websocket.TextMessage, []byte("data"+statusCodePrefix+"0"))
	})

	testServer(t, s, func(c *sdk.Client) {
		rw := bufio.NewReadWriter(bufio.NewReader(bytes.NewReader([]byte{})), bufio.NewWriter(&bytes.Buffer{}))
		got, err := c.ProcessExec(app, id, "cmd", rw, structs.ProcessExecOptions{
			Entrypoint: options.Bool(true),
			Height:     options.Int(1),
			Width:      options.Int(2),
		})
		require.NoError(t, err)
		require.Equal(t, 0, got)
	})
}

func TestProcessGet(t *testing.T) {
	app := "app1"
	pid := "pid"
	p := &structs.Process{
		Id:             "id1",
		App:            "app1",
		Command:        "command",
		Cpu:            5,
		Host:           "host",
		Image:          "image",
		Instance:       "instance",
		Memory:         1024,
		Name:           "name",
		Ports:          []string{"5000"},
		Release:        "release",
		Status:         "status",
		TaskDefinition: "task_definition",
	}

	s := stdapi.New("api", "api")
	s.Route("GET", fmt.Sprintf("/apps/%s/processes/%s", app, pid), func(c *stdapi.Context) error {
		return c.RenderJSON(p)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.ProcessGet(app, pid)
		require.NoError(t, err)
		require.Equal(t, p, got)
	})
}

func TestProcessList(t *testing.T) {
	app := "app1"
	ps := structs.Processes{
		{
			Id:             "id1",
			App:            "app1",
			Command:        "command",
			Cpu:            5,
			Host:           "host",
			Image:          "image",
			Instance:       "instance",
			Memory:         1024,
			Name:           "name",
			Ports:          []string{"5000"},
			Release:        "release",
			Status:         "status",
			TaskDefinition: "task_definition",
		},
	}

	s := stdapi.New("api", "api")
	s.Route("GET", fmt.Sprintf("/apps/%s/processes", app), func(c *stdapi.Context) error {
		require.Equal(t, "srv1", c.Query("service"))
		return c.RenderJSON(ps)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.ProcessList(app, structs.ProcessListOptions{
			Service: options.String("srv1"),
		})
		require.NoError(t, err)
		require.Equal(t, ps, got)
	})
}

func TestProcessLogs(t *testing.T) {
	app := "app1"
	pid := "id1"
	s := stdapi.New("api", "api")
	s.Route("GET", "/racks", func(c *stdapi.Context) error {
		return c.RenderOK()
	})
	s.Route("SOCKET", fmt.Sprintf("/apps/%s/processes/%s/logs", app, pid), func(c *stdapi.Context) error {
		require.Equal(t, "test", c.Header("Filter"))
		return c.Websocket().WriteMessage(websocket.TextMessage, []byte("data"))
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.ProcessLogs(app, pid, structs.LogsOptions{
			Filter: options.String("test"),
		})
		require.NoError(t, err)
		b, _ := io.ReadAll(got)
		require.Equal(t, "data", string(b))
	})
}

func TestProcessRun(t *testing.T) {
	app := "app1"
	service := "srv1"
	ps := &structs.Process{
		Id:             "id1",
		App:            "app",
		Command:        "command",
		Cpu:            5,
		Host:           "host",
		Image:          "image",
		Instance:       "instance",
		Memory:         1024,
		Name:           "name",
		Ports:          []string{"5000"},
		Release:        "release",
		Status:         "status",
		TaskDefinition: "task_definition",
	}

	s := stdapi.New("api", "api")
	s.Route("POST", fmt.Sprintf("/apps/%s/services/%s/processes", app, service), func(c *stdapi.Context) error {
		require.Equal(t, ps.Command, c.Header("command"))
		require.Equal(t, ps.Image, c.Header("image"))
		require.Equal(t, strconv.Itoa(int(ps.Memory)), c.Header("memory"))
		require.Equal(t, ps.Release, c.Header("release"))
		return c.RenderJSON(ps)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.ProcessRun(app, service, structs.ProcessRunOptions{
			Command: options.String("command"),
			Image:   options.String("image"),
			Memory:  options.Int(1024),
			Release: options.String("release"),
		})
		require.NoError(t, err)
		require.Equal(t, ps, got)
	})
}

func TestProcessStop(t *testing.T) {
	app := "app1"
	pid := "pid"
	s := stdapi.New("api", "api")
	s.Route("DELETE", fmt.Sprintf("/apps/%s/processes/%s", app, pid), func(c *stdapi.Context) error {
		return c.RenderOK()
	})

	testServer(t, s, func(c *sdk.Client) {
		err := c.ProcessStop(app, pid)
		require.NoError(t, err)
	})
}

func TestRegistryAdd(t *testing.T) {
	rs := &structs.Registry{
		Server:   "server",
		Username: "username",
		Password: "password",
	}
	s := stdapi.New("api", "api")
	s.Route("POST", "/registries", func(c *stdapi.Context) error {
		require.Equal(t, rs.Server, c.Form("server"))
		require.Equal(t, rs.Username, c.Form("username"))
		require.Equal(t, rs.Password, c.Form("password"))
		return c.RenderJSON(rs)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.RegistryAdd(rs.Server, rs.Username, rs.Password)
		require.NoError(t, err)
		require.Equal(t, rs, got)
	})
}

func TestRegistryList(t *testing.T) {
	rs := structs.Registries{{
		Server:   "server",
		Username: "username",
		Password: "password",
	}}
	s := stdapi.New("api", "api")
	s.Route("GET", "/registries", func(c *stdapi.Context) error {
		return c.RenderJSON(rs)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.RegistryList()
		require.NoError(t, err)
		require.Equal(t, rs, got)
	})
}

func TestRegistryRemove(t *testing.T) {
	server := "server"
	s := stdapi.New("api", "api")
	s.Route("DELETE", fmt.Sprintf("/registries/%s", server), func(c *stdapi.Context) error {
		return c.RenderOK()
	})

	testServer(t, s, func(c *sdk.Client) {
		err := c.RegistryRemove(server)
		require.NoError(t, err)
	})
}
func TestReleaseCreate(t *testing.T) {
	rs := &structs.Release{
		Id:          "id",
		App:         "app",
		Build:       "build",
		Env:         "env",
		Manifest:    "manifest",
		Description: "description",
	}
	s := stdapi.New("api", "api")
	s.Route("POST", fmt.Sprintf("/apps/%s/releases", rs.App), func(c *stdapi.Context) error {
		require.Equal(t, rs.Build, c.Form("build"))
		require.Equal(t, rs.Description, c.Form("description"))
		require.Equal(t, rs.Env, c.Form("env"))
		return c.RenderJSON(rs)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.ReleaseCreate(rs.App, structs.ReleaseCreateOptions{
			Build:       options.String(rs.Build),
			Description: options.String(rs.Description),
			Env:         options.String(rs.Env),
		})
		require.NoError(t, err)
		require.Equal(t, rs, got)
	})
}

func TestReleaseGet(t *testing.T) {
	rs := &structs.Release{
		Id:          "id",
		App:         "app",
		Build:       "build",
		Env:         "env",
		Manifest:    "manifest",
		Description: "description",
	}
	s := stdapi.New("api", "api")
	s.Route("GET", fmt.Sprintf("/apps/%s/releases/%s", rs.App, rs.Id), func(c *stdapi.Context) error {
		return c.RenderJSON(rs)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.ReleaseGet(rs.App, rs.Id)
		require.NoError(t, err)
		require.Equal(t, rs, got)
	})
}

func TestReleaseList(t *testing.T) {
	rs := structs.Releases{{
		Id:          "id",
		App:         "app",
		Build:       "build",
		Env:         "env",
		Manifest:    "manifest",
		Description: "description",
	}}
	s := stdapi.New("api", "api")
	s.Route("GET", fmt.Sprintf("/apps/%s/releases", rs[0].App), func(c *stdapi.Context) error {
		require.Equal(t, "10", c.Query("limit"))
		return c.RenderJSON(rs)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.ReleaseList(rs[0].App, structs.ReleaseListOptions{
			Limit: options.Int(10),
		})
		require.NoError(t, err)
		require.Equal(t, rs, got)
	})
}

func TestReleasePromote(t *testing.T) {
	rs := &structs.Release{
		Id:          "id",
		App:         "app",
		Build:       "build",
		Env:         "env",
		Manifest:    "manifest",
		Description: "description",
	}
	s := stdapi.New("api", "api")
	s.Route("POST", fmt.Sprintf("/apps/%s/releases/%s/promote", rs.App, rs.Id), func(c *stdapi.Context) error {
		require.Equal(t, "true", c.Form("development"))
		require.Equal(t, "true", c.Form("force"))
		require.Equal(t, "false", c.Form("idle"))
		require.Equal(t, "1", c.Form("min"))
		require.Equal(t, "2", c.Form("max"))
		require.Equal(t, "25", c.Form("timeout"))
		return c.RenderOK()
	})

	testServer(t, s, func(c *sdk.Client) {
		err := c.ReleasePromote(rs.App, rs.Id, structs.ReleasePromoteOptions{
			Development: options.Bool(true),
			Force:       options.Bool(true),
			Idle:        options.Bool(false),
			Min:         options.Int(1),
			Max:         options.Int(2),
			Timeout:     options.Int(25),
		})
		require.NoError(t, err)
	})
}

func TestResourceGet(t *testing.T) {
	rs := &structs.Resource{
		Name:       "name",
		Parameters: map[string]string{"hello": "world"},
		Status:     "status,omitempty",
		Type:       "type",
		Url:        "url",
		Apps:       structs.Apps{{Name: "app1"}},
	}
	s := stdapi.New("api", "api")
	s.Route("GET", fmt.Sprintf("/apps/%s/resources/%s", rs.Apps[0].Name, rs.Name), func(c *stdapi.Context) error {
		return c.RenderJSON(rs)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.ResourceGet(rs.Apps[0].Name, rs.Name)
		require.NoError(t, err)
		require.Equal(t, rs, got)
	})
}

func TestResourceList(t *testing.T) {
	rs := structs.Resources{{
		Name:       "name",
		Parameters: map[string]string{"hello": "world"},
		Status:     "status,omitempty",
		Type:       "type",
		Url:        "url",
		Apps:       structs.Apps{{Name: "app1"}},
	}}
	s := stdapi.New("api", "api")
	s.Route("GET", fmt.Sprintf("/apps/%s/resources", rs[0].Apps[0].Name), func(c *stdapi.Context) error {
		return c.RenderJSON(rs)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.ResourceList(rs[0].Apps[0].Name)
		require.NoError(t, err)
		require.Equal(t, rs, got)
	})
}

func TestServiceList(t *testing.T) {
	app := "app1"
	srv := structs.Services{{
		Count:  2,
		Cpu:    4,
		Domain: "domain",
		Memory: 128,
		Name:   "name",
		Ports:  []structs.ServicePort{{Balancer: 3000, Certificate: "cert1", Container: 1}},
	}}
	s := stdapi.New("api", "api")
	s.Route("GET", fmt.Sprintf("/apps/%s/services", app), func(c *stdapi.Context) error {
		return c.RenderJSON(srv)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.ServiceList(app)
		require.NoError(t, err)
		require.Equal(t, srv, got)
	})
}

func TestServiceMetrics(t *testing.T) {
	app := "app1"
	service := "srv1"
	startTime := time.Now().UTC()
	ms := structs.Metrics{
		{
			Name: "test1",
			Values: structs.MetricValues{
				{
					Average: 2.3,
					Count:   3,
					Maximum: 5,
					Minimum: 2,
				},
				{
					Average: 2.3,
					Count:   3,
					Maximum: 5,
					Minimum: 2,
				},
			},
		},
	}

	s := stdapi.New("api", "api")
	s.Route("GET", fmt.Sprintf("/apps/%s/services/%s/metrics", app, service), func(c *stdapi.Context) error {
		require.Equal(t, "120", c.Query("period"))
		require.Equal(t, startTime.Format("20060102.150405.000000000"), c.Query("start"))
		return c.RenderJSON(ms)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.ServiceMetrics(app, service, structs.MetricsOptions{
			Period: options.Int64(120),
			Start:  options.Time(startTime),
		})
		require.NoError(t, err)
		require.Equal(t, ms, got)
	})
}

func TestServiceRestart(t *testing.T) {
	app := "app1"
	service := "srv1"
	s := stdapi.New("api", "api")
	s.Route("POST", fmt.Sprintf("/apps/%s/services/%s/restart", app, service), func(c *stdapi.Context) error {
		return c.RenderOK()
	})

	testServer(t, s, func(c *sdk.Client) {
		err := c.ServiceRestart(app, service)
		require.NoError(t, err)
	})
}

func TestServiceUpdate(t *testing.T) {
	app := "app1"
	service := "srv1"
	s := stdapi.New("api", "api")
	s.Route("PUT", fmt.Sprintf("/apps/%s/services/%s", app, service), func(c *stdapi.Context) error {
		require.Equal(t, "2", c.Form("count"))
		require.Equal(t, "4", c.Form("cpu"))
		require.Equal(t, "1024", c.Form("memory"))
		return c.RenderOK()
	})

	testServer(t, s, func(c *sdk.Client) {
		err := c.ServiceUpdate(app, service, structs.ServiceUpdateOptions{
			Count:  options.Int(2),
			Cpu:    options.Int(4),
			Memory: options.Int(1024),
		})
		require.NoError(t, err)
	})
}

func TestSystemGet(t *testing.T) {
	ss := &structs.System{
		Count:      3,
		Domain:     "domain",
		Name:       "name",
		Outputs:    map[string]string{"looks": "good"},
		Parameters: map[string]string{"hello": "world"},
		Provider:   "provider",
		Region:     "region",
		Status:     "status",
		Type:       "type",
		Version:    "version",
	}
	s := stdapi.New("api", "api")
	s.Route("GET", "/system", func(c *stdapi.Context) error {
		return c.RenderJSON(ss)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.SystemGet()
		require.NoError(t, err)
		require.Equal(t, ss, got)
	})
}

func TestSystemLogs(t *testing.T) {
	s := stdapi.New("api", "api")
	s.Route("GET", "/racks", func(c *stdapi.Context) error {
		return c.RenderOK()
	})
	s.Route("SOCKET", "/system/logs", func(c *stdapi.Context) error {
		require.Equal(t, "test", c.Header("Filter"))
		return c.Websocket().WriteMessage(websocket.TextMessage, []byte("data"))
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.SystemLogs(structs.LogsOptions{
			Filter: options.String("test"),
		})
		require.NoError(t, err)
		b, _ := io.ReadAll(got)
		require.Equal(t, "data", string(b))
	})
}

func TestSystemMetrics(t *testing.T) {
	startTime := time.Now().UTC()
	ms := structs.Metrics{
		{
			Name: "test1",
			Values: structs.MetricValues{
				{
					Average: 2.3,
					Count:   3,
					Maximum: 5,
					Minimum: 2,
				},
				{
					Average: 2.3,
					Count:   3,
					Maximum: 5,
					Minimum: 2,
				},
			},
		},
	}

	s := stdapi.New("api", "api")
	s.Route("GET", "/system/metrics", func(c *stdapi.Context) error {
		require.Equal(t, "120", c.Query("period"))
		require.Equal(t, startTime.Format("20060102.150405.000000000"), c.Query("start"))
		return c.RenderJSON(ms)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.SystemMetrics(structs.MetricsOptions{
			Period: options.Int64(120),
			Start:  options.Time(startTime),
		})
		require.NoError(t, err)
		require.Equal(t, ms, got)
	})
}

func TestSystemProcesses(t *testing.T) {
	ps := structs.Processes{
		{
			Id:             "id1",
			App:            "app1",
			Command:        "command",
			Cpu:            5,
			Host:           "host",
			Image:          "image",
			Instance:       "instance",
			Memory:         1024,
			Name:           "name",
			Ports:          []string{"5000"},
			Release:        "release",
			Status:         "status",
			TaskDefinition: "task_definition",
		},
	}

	s := stdapi.New("api", "api")
	s.Route("GET", "/system/processes", func(c *stdapi.Context) error {
		require.Equal(t, "true", c.Query("all"))
		return c.RenderJSON(ps)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.SystemProcesses(structs.SystemProcessesOptions{
			All: options.Bool(true),
		})
		require.NoError(t, err)
		require.Equal(t, ps, got)
	})
}

func TestSystemReleases(t *testing.T) {
	rs := structs.Releases{{
		Id:          "id",
		App:         "app",
		Build:       "build",
		Env:         "env",
		Manifest:    "manifest",
		Description: "description",
	}}

	s := stdapi.New("api", "api")
	s.Route("GET", "/system/releases", func(c *stdapi.Context) error {
		return c.RenderJSON(rs)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.SystemReleases()
		require.NoError(t, err)
		require.Equal(t, rs, got)
	})
}

func TestSystemResourceCreate(t *testing.T) {
	rs := &structs.Resource{
		Name:       "name",
		Parameters: map[string]string{"hello": "world"},
		Status:     "status,omitempty",
		Type:       "type",
		Url:        "url",
		Apps:       structs.Apps{{Name: "app1"}},
	}
	s := stdapi.New("api", "api")
	s.Route("POST", "/resources", func(c *stdapi.Context) error {
		require.Equal(t, rs.Type, c.Form("kind"))
		require.Equal(t, rs.Name, c.Form("name"))
		require.Equal(t, "hello=world", c.Form("parameters"))
		return c.RenderJSON(rs)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.SystemResourceCreate(rs.Type, structs.ResourceCreateOptions{
			Name:       &rs.Name,
			Parameters: rs.Parameters,
		})
		require.NoError(t, err)
		require.Equal(t, rs, got)
	})
}

func TestSystemResourceDelete(t *testing.T) {
	name := "rs1"
	s := stdapi.New("api", "api")
	s.Route("DELETE", fmt.Sprintf("/resources/%s", name), func(c *stdapi.Context) error {
		return c.RenderOK()
	})

	testServer(t, s, func(c *sdk.Client) {
		err := c.SystemResourceDelete(name)
		require.NoError(t, err)
	})
}

func TestSystemResourceGet(t *testing.T) {
	name := "rs1"
	rs := &structs.Resource{
		Name:       name,
		Parameters: map[string]string{"hello": "world"},
		Status:     "status,omitempty",
		Type:       "type",
		Url:        "url",
		Apps:       structs.Apps{{Name: "app1"}},
	}
	s := stdapi.New("api", "api")
	s.Route("GET", fmt.Sprintf("/resources/%s", name), func(c *stdapi.Context) error {
		return c.RenderJSON(rs)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.SystemResourceGet(name)
		require.NoError(t, err)
		require.Equal(t, rs, got)
	})
}

func TestSystemResourceLink(t *testing.T) {
	name := "rs1"
	app := "app1"
	rs := &structs.Resource{
		Name:       name,
		Parameters: map[string]string{"hello": "world"},
		Status:     "status,omitempty",
		Type:       "type",
		Url:        "url",
		Apps:       structs.Apps{{Name: app}},
	}
	s := stdapi.New("api", "api")
	s.Route("POST", fmt.Sprintf("/resources/%s/links", name), func(c *stdapi.Context) error {
		require.Equal(t, app, c.Form("app"))
		return c.RenderJSON(rs)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.SystemResourceLink(name, app)
		require.NoError(t, err)
		require.Equal(t, rs, got)
	})
}

func TestSystemResourceList(t *testing.T) {
	name := "rs1"
	rs := structs.Resources{{
		Name:       name,
		Parameters: map[string]string{"hello": "world"},
		Status:     "status,omitempty",
		Type:       "type",
		Url:        "url",
		Apps:       structs.Apps{{Name: "app1"}},
	}}
	s := stdapi.New("api", "api")
	s.Route("GET", "/resources", func(c *stdapi.Context) error {
		return c.RenderJSON(rs)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.SystemResourceList()
		require.NoError(t, err)
		require.Equal(t, rs, got)
	})
}

func TestSystemResourceTypes(t *testing.T) {
	name := "rs1"
	rs := structs.ResourceTypes{{
		Name: name,
		Parameters: structs.ResourceParameters{
			{
				Default:     "1",
				Description: "d1",
				Name:        "p1",
			},
		},
	}}
	s := stdapi.New("api", "api")
	s.Route("OPTIONS", "/resources", func(c *stdapi.Context) error {
		return c.RenderJSON(rs)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.SystemResourceTypes()
		require.NoError(t, err)
		require.Equal(t, rs, got)
	})
}

func TestSystemResourceUnlink(t *testing.T) {
	name := "rs1"
	app := "app1"
	rs := &structs.Resource{
		Name:       name,
		Parameters: map[string]string{"hello": "world"},
		Status:     "status,omitempty",
		Type:       "type",
		Url:        "url",
		Apps:       nil,
	}
	s := stdapi.New("api", "api")
	s.Route("DELETE", fmt.Sprintf("/resources/%s/links/%s", name, app), func(c *stdapi.Context) error {
		return c.RenderJSON(rs)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.SystemResourceUnlink(name, app)
		require.NoError(t, err)
		require.Equal(t, rs, got)
	})
}

func TestSystemResourceUpdate(t *testing.T) {
	rs := &structs.Resource{
		Name:       "name",
		Parameters: map[string]string{"hello": "world"},
		Status:     "status,omitempty",
		Type:       "type",
		Url:        "url",
		Apps:       structs.Apps{{Name: "app1"}},
	}
	s := stdapi.New("api", "api")
	s.Route("PUT", fmt.Sprintf("/resources/%s", rs.Name), func(c *stdapi.Context) error {
		require.Equal(t, "hello=world", c.Form("parameters"))
		return c.RenderJSON(rs)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.SystemResourceUpdate(rs.Name, structs.ResourceUpdateOptions{
			Parameters: rs.Parameters,
		})
		require.NoError(t, err)
		require.Equal(t, rs, got)
	})
}

func TestSystemUpdate(t *testing.T) {
	s := stdapi.New("api", "api")
	s.Route("PUT", "/system", func(c *stdapi.Context) error {
		require.Equal(t, "2", c.Form("count"))
		require.Equal(t, "hello=world", c.Form("parameters"))
		require.Equal(t, "type", c.Form("type"))
		require.Equal(t, "version", c.Form("version"))
		return c.RenderOK()
	})

	testServer(t, s, func(c *sdk.Client) {
		err := c.SystemUpdate(structs.SystemUpdateOptions{
			Count:      options.Int(2),
			Parameters: map[string]string{"hello": "world"},
			Type:       options.String("type"),
			Version:    options.String("version"),
		})
		require.NoError(t, err)
	})
}
