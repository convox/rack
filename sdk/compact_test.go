package sdk_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdapi"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestAppParametersGet(t *testing.T) {
	app := "app1"
	params := map[string]string{
		"hello":  "world",
		"convox": "rocks",
	}

	s := stdapi.New("api", "api")
	s.Route("GET", fmt.Sprintf("/apps/%s/parameters", app), func(c *stdapi.Context) error {
		return c.RenderJSON(params)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.AppParametersGet(app)
		require.NoError(t, err)
		require.Equal(t, params, got)
	})
}

func TestAppParametersSet(t *testing.T) {
	app := "app1"
	params := map[string]string{
		"hello":  "world",
		"convox": "rocks",
	}

	s := stdapi.New("api", "api")
	s.Route("POST", fmt.Sprintf("/apps/%s/parameters", app), func(c *stdapi.Context) error {
		for k, v := range params {
			require.Equal(t, v, c.Form(k))
		}
		return c.RenderOK()
	})

	testServer(t, s, func(c *sdk.Client) {
		err := c.AppParametersSet(app, params)
		require.NoError(t, err)
	})
}

func TestBuildCreateUpload(t *testing.T) {
	app := "app1"
	build := structs.Build{
		Id:  "1",
		App: app,
	}

	s := stdapi.New("api", "api")
	s.Route("POST", fmt.Sprintf("/apps/%s/builds", app), func(c *stdapi.Context) error {
		require.Equal(t, "convox", c.Form("description"))
		require.Equal(t, "true", c.Form("cache"))

		fp, _, err := c.Request().FormFile("source")
		require.NoError(t, err)
		b, err := io.ReadAll(fp)
		require.NoError(t, err)
		require.Equal(t, "test", string(b))

		return c.RenderJSON(build)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.BuildCreateUpload(app, bytes.NewReader([]byte("test")), structs.BuildCreateOptions{
			Description: options.String("convox"),
		})
		require.NoError(t, err)
		require.Equal(t, &build, got)
	})
}

func TestBuildImportMultipart(t *testing.T) {
	app := "app1"
	build := structs.Build{
		Id:  "1",
		App: app,
	}

	s := stdapi.New("api", "api")
	s.Route("POST", fmt.Sprintf("/apps/%s/builds", app), func(c *stdapi.Context) error {
		fp, _, err := c.Request().FormFile("image")
		require.NoError(t, err)
		b, err := io.ReadAll(fp)
		require.NoError(t, err)
		require.Equal(t, "test", string(b))
		return c.RenderJSON(build)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.BuildImportMultipart(app, bytes.NewReader([]byte("test")))
		require.NoError(t, err)
		require.Equal(t, &build, got)
	})
}

func TestBuildImportUrl(t *testing.T) {
	app := "app1"
	key := ""
	build := structs.Build{
		Id:  "1",
		App: app,
	}

	s := stdapi.New("api", "api")
	s.Route("POST", fmt.Sprintf("/apps/%s/builds/import", app), func(c *stdapi.Context) error {
		require.Equal(t, "url", c.Form("url"))
		return c.RenderJSON(build)
	})

	s.Route("POST", fmt.Sprintf("/apps/%s/objects/%s", app, key), func(c *stdapi.Context) error {
		b, err := io.ReadAll(c.Request().Body)
		require.NoError(t, err)
		require.Equal(t, "test", string(b))
		return c.RenderJSON(structs.Object{
			Url: "url",
		})
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.BuildImportUrl(app, bytes.NewReader([]byte("test")))
		require.NoError(t, err)
		require.Equal(t, &build, got)
	})
}

func TestCertificateCreateClassic(t *testing.T) {
	certResp := &structs.Certificate{
		Id:         "1",
		Domain:     "convox.com",
		Expiration: time.Now().UTC(),
	}

	s := stdapi.New("api", "api")
	s.Route("POST", "/certificates", func(c *stdapi.Context) error {
		require.Equal(t, "chain", c.Form("chain"))
		require.Equal(t, "public", c.Form("public"))
		require.Equal(t, "private", c.Form("private"))
		return c.RenderJSON(certResp)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.CertificateCreateClassic("public", "private", structs.CertificateCreateOptions{
			Chain: options.String("chain"),
		})
		require.NoError(t, err)
		require.Equal(t, certResp, got)
	})
}

func TestEnvironmentSet(t *testing.T) {
	app := "app1"
	releaseId := "release1"
	releaseResp := &structs.Release{
		Id:    "1",
		App:   app,
		Build: "1234",
	}

	s := stdapi.New("api", "api")
	s.Route("POST", fmt.Sprintf("/apps/%s/environment", app), func(c *stdapi.Context) error {
		b, err := io.ReadAll(c.Request().Body)
		require.NoError(t, err)
		require.Equal(t, "hello=world", string(b))
		c.Response().Header().Set("Release-Id", releaseId)
		return nil
	})

	s.Route("GET", fmt.Sprintf("/apps/%s/releases/%s", app, releaseId), func(c *stdapi.Context) error {
		return c.RenderJSON(releaseResp)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.EnvironmentSet(app, []byte("hello=world"))
		require.NoError(t, err)
		require.Equal(t, releaseResp, got)
	})
}

func TestEnvironmentUnset(t *testing.T) {
	app := "app1"
	key := "key1"
	releaseId := "release1"
	releaseResp := &structs.Release{
		Id:    "1",
		App:   app,
		Build: "1234",
	}

	s := stdapi.New("api", "api")
	s.Route("DELETE", fmt.Sprintf("/apps/%s/environment/%s", app, key), func(c *stdapi.Context) error {
		c.Response().Header().Set("Release-Id", releaseId)
		return nil
	})

	s.Route("GET", fmt.Sprintf("/apps/%s/releases/%s", app, releaseId), func(c *stdapi.Context) error {
		return c.RenderJSON(releaseResp)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.EnvironmentUnset(app, key)
		require.NoError(t, err)
		require.Equal(t, releaseResp, got)
	})
}

func TestFormationGet(t *testing.T) {
	app := "app1"
	fs := []struct {
		Balancer string
		Count    int
		Cpu      int
		Hostname string
		Memory   int
		Name     string
		Ports    []int
	}{
		{
			Balancer: "b1",
			Count:    1,
			Cpu:      2,
			Hostname: "h1",
			Memory:   100,
			Name:     "s1",
			Ports:    []int{3001},
		},
		{
			Balancer: "b2",
			Count:    2,
			Cpu:      4,
			Hostname: "h2",
			Memory:   200,
			Name:     "s2",
			Ports:    []int{5000},
		},
	}
	ssls := []struct {
		Certificate string
		Process     string
		Port        int
	}{
		{
			Certificate: "c1",
			Process:     "s1",
			Port:        3001,
		},
		{
			Certificate: "c2",
			Process:     "s2",
			Port:        5000,
		},
	}

	ss := structs.Services{}
	for i := range fs {
		ss = append(ss, structs.Service{
			Count:  fs[i].Count,
			Cpu:    fs[i].Cpu,
			Domain: fs[i].Balancer,
			Memory: fs[i].Memory,
			Name:   fs[i].Name,
			Ports: []structs.ServicePort{
				{
					Balancer:    fs[i].Ports[0],
					Certificate: ssls[i].Certificate,
				},
			},
		})
	}

	s := stdapi.New("api", "api")
	s.Route("GET", fmt.Sprintf("/apps/%s/formation", app), func(c *stdapi.Context) error {
		return c.RenderJSON(fs)
	})

	s.Route("GET", fmt.Sprintf("/apps/%s/ssl", app), func(c *stdapi.Context) error {
		return c.RenderJSON(ssls)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.FormationGet(app)
		require.NoError(t, err)
		require.Equal(t, ss, got)
	})
}

func TestFormationUpdate(t *testing.T) {
	app := "app1"
	service := "service1"

	s := stdapi.New("api", "api")
	s.Route("POST", fmt.Sprintf("/apps/%s/formation/%s", app, service), func(c *stdapi.Context) error {
		require.Equal(t, "2", c.Form("count"))
		require.Equal(t, "128", c.Form("cpu"))
		require.Equal(t, "1024", c.Form("memory"))
		return c.RenderOK()
	})

	testServer(t, s, func(c *sdk.Client) {
		err := c.FormationUpdate(app, service, structs.ServiceUpdateOptions{
			Count:  options.Int(2),
			Cpu:    options.Int(128),
			Memory: options.Int(1024),
		})
		require.NoError(t, err)
	})
}

func TestProcessRunDetached(t *testing.T) {
	app := "app1"
	service := "service1"
	envs := map[string]string{
		"hello": "world",
		"then":  "what",
	}

	s := stdapi.New("api", "api")
	s.Route("POST", fmt.Sprintf("/apps/%s/processes/%s/run", app, service), func(c *stdapi.Context) error {
		require.Equal(t, "echo help", c.Form("command"))
		require.Equal(t, "release1", c.Form("release"))

		require.Equal(t, "hello=world&then=what", c.Header("Environment"))
		require.Equal(t, "10", c.Header("Height"))
		require.Equal(t, "image1", c.Header("Image"))
		return c.RenderJSON(struct {
			Pid string
		}{"1234"})
	})

	testServer(t, s, func(c *sdk.Client) {
		id, err := c.ProcessRunDetached(app, service, structs.ProcessRunOptions{
			Command:     options.String("echo help"),
			Environment: envs,
			Height:      options.Int(10),
			Image:       options.String("image1"),
			Release:     options.String("release1"),
		})
		require.NoError(t, err)
		require.Equal(t, "1234", id)
	})
}

func TestInstanceShellClassic(t *testing.T) {
	id := "id1"
	s := stdapi.New("api", "api")
	s.Route("SOCKET", fmt.Sprintf("/instances/%s/ssh", id), func(c *stdapi.Context) error {
		require.Equal(t, "test", c.Header("Command"))
		require.Equal(t, "2", c.Header("Height"))
		require.Equal(t, "1", c.Header("Width"))
		_, b, err := c.Websocket().ReadMessage()
		require.NoError(t, err)
		require.Equal(t, "cmd", string(b))
		return c.Websocket().WriteMessage(websocket.TextMessage, []byte("data"+statusCodePrefix+"0"))
	})

	testServer(t, s, func(c *sdk.Client) {
		rw := bufio.NewReadWriter(bufio.NewReader(bytes.NewReader([]byte("cmd"))), bufio.NewWriter(&bytes.Buffer{}))
		got, err := c.InstanceShellClassic(id, rw, structs.InstanceShellOptions{
			Command: options.String("test"),
			Height:  options.Int(1),
			Width:   options.Int(2),
		})
		require.NoError(t, err)
		require.Equal(t, 0, got)
	})
}

func TestRegistryRemoveClassic(t *testing.T) {
	server := "server1"

	s := stdapi.New("api", "api")
	s.Route("DELETE", "/registries", func(c *stdapi.Context) error {
		require.Equal(t, server, c.Query("server"))
		return c.RenderOK()
	})

	testServer(t, s, func(c *sdk.Client) {
		err := c.RegistryRemoveClassic(server)
		require.NoError(t, err)
	})
}

func TestResourceCreateClassic(t *testing.T) {
	kind := "postgres"
	r := &structs.Resource{
		Name:       "r1",
		Parameters: map[string]string{"hello": "world"},
		Status:     "running",
		Type:       kind,
	}

	s := stdapi.New("api", "api")
	s.Route("POST", "/resources", func(c *stdapi.Context) error {
		require.Equal(t, kind, c.Form("type"))
		require.Equal(t, r.Name, c.Form("name"))
		for k, v := range r.Parameters {
			require.Equal(t, v, c.Form(k))
		}
		return c.RenderJSON(r)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.ResourceCreateClassic(kind, structs.ResourceCreateOptions{
			Name:       &r.Name,
			Parameters: r.Parameters,
		})
		require.NoError(t, err)
		require.Equal(t, r, got)
	})
}

func TestResourceUpdateClassic(t *testing.T) {
	r := &structs.Resource{
		Name:       "r1",
		Parameters: map[string]string{"hello": "world"},
		Status:     "running",
		Type:       "postgres",
	}

	s := stdapi.New("api", "api")
	s.Route("PUT", fmt.Sprintf("/resources/%s", r.Name), func(c *stdapi.Context) error {
		for k, v := range r.Parameters {
			require.Equal(t, v, c.Form(k))
		}
		return c.RenderJSON(r)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.ResourceUpdateClassic(r.Name, structs.ResourceUpdateOptions{
			Parameters: r.Parameters,
		})
		require.NoError(t, err)
		require.Equal(t, r, got)
	})
}

func TestSystemResourceCreateClassic(t *testing.T) {
	kind := "postgres"
	r := &structs.Resource{
		Name: "r1",
		Parameters: map[string]string{
			"hello": "world",
			"then":  "what",
		},
		Status: "running",
		Type:   kind,
	}

	s := stdapi.New("api", "api")
	s.Route("POST", "/resources", func(c *stdapi.Context) error {
		require.Equal(t, kind, c.Form("kind"))
		require.Equal(t, r.Name, c.Form("name"))
		require.Equal(t, "hello=world&then=what", c.Form("parameters"))
		return c.RenderJSON(r)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.SystemResourceCreateClassic(kind, structs.ResourceCreateOptions{
			Name:       &r.Name,
			Parameters: r.Parameters,
		})
		require.NoError(t, err)
		require.Equal(t, r, got)
	})
}

func TestSystemResourceDeleteClassic(t *testing.T) {
	name := "r1"
	s := stdapi.New("api", "api")
	s.Route("DELETE", fmt.Sprintf("/resources/%s", name), func(c *stdapi.Context) error {
		return c.RenderOK()
	})

	testServer(t, s, func(c *sdk.Client) {
		err := c.SystemResourceDeleteClassic(name)
		require.NoError(t, err)
	})
}

func TestSystemResourceGetClassic(t *testing.T) {
	r := &structs.Resource{
		Name: "r1",
		Parameters: map[string]string{
			"hello": "world",
			"then":  "what",
		},
		Status: "running",
		Type:   "postgres",
	}
	s := stdapi.New("api", "api")
	s.Route("GET", fmt.Sprintf("/resources/%s", r.Name), func(c *stdapi.Context) error {
		return c.RenderJSON(r)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.SystemResourceGetClassic(r.Name)
		require.NoError(t, err)
		require.Equal(t, r, got)
	})
}

func TestSystemResourceLinkClassic(t *testing.T) {
	app := "app1"
	r := &structs.Resource{
		Name: "r1",
		Parameters: map[string]string{
			"hello": "world",
			"then":  "what",
		},
		Status: "running",
		Type:   "postgres",
	}
	s := stdapi.New("api", "api")
	s.Route("POST", fmt.Sprintf("/resources/%s/links", r.Name), func(c *stdapi.Context) error {
		require.Equal(t, app, c.Form("app"))
		return c.RenderJSON(r)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.SystemResourceLinkClassic(r.Name, app)
		require.NoError(t, err)
		require.Equal(t, r, got)
	})
}

func TestSystemResourceListClassic(t *testing.T) {
	rs := structs.Resources{
		{
			Name: "r1",
			Parameters: map[string]string{
				"hello": "world",
				"then":  "what",
			},
			Status: "running",
			Type:   "postgres",
		},
		{
			Name: "r2",
			Parameters: map[string]string{
				"hello": "world",
			},
			Status: "running",
			Type:   "postgres",
		},
	}
	s := stdapi.New("api", "api")
	s.Route("GET", "/resources", func(c *stdapi.Context) error {
		return c.RenderJSON(rs)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.SystemResourceListClassic()
		require.NoError(t, err)
		require.Equal(t, rs, got)
	})
}

func TestSystemResourceTypesClassic(t *testing.T) {
	rs := structs.ResourceTypes{
		{
			Name: "r1",
			Parameters: structs.ResourceParameters{
				{
					Default:     "1",
					Description: "d",
					Name:        "p1",
				},
			},
		},
		{
			Name: "r2",
			Parameters: structs.ResourceParameters{
				{
					Default:     "test",
					Description: "d2",
					Name:        "p2",
				},
			},
		},
	}
	s := stdapi.New("api", "api")
	s.Route("OPTIONS", "/resources", func(c *stdapi.Context) error {
		return c.RenderJSON(rs)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.SystemResourceTypesClassic()
		require.NoError(t, err)
		require.Equal(t, rs, got)
	})
}

func TestSystemResourceUnlinkClassic(t *testing.T) {
	app := "app1"
	r := &structs.Resource{
		Name: "r1",
		Parameters: map[string]string{
			"hello": "world",
			"then":  "what",
		},
		Status: "running",
		Type:   "postgres",
	}
	s := stdapi.New("api", "api")
	s.Route("DELETE", fmt.Sprintf("/resources/%s/links/%s", r.Name, app), func(c *stdapi.Context) error {
		return c.RenderJSON(r)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.SystemResourceUnlinkClassic(r.Name, app)
		require.NoError(t, err)
		require.Equal(t, r, got)
	})
}

func TestSystemResourceUpdateClassic(t *testing.T) {
	r := &structs.Resource{
		Name:       "r1",
		Parameters: map[string]string{"hello": "world"},
		Status:     "running",
		Type:       "postgres",
	}

	s := stdapi.New("api", "api")
	s.Route("PUT", fmt.Sprintf("/resources/%s", r.Name), func(c *stdapi.Context) error {
		require.Equal(t, "hello=world", c.Form("parameters"))
		return c.RenderJSON(r)
	})

	testServer(t, s, func(c *sdk.Client) {
		got, err := c.SystemResourceUpdateClassic(r.Name, structs.ResourceUpdateOptions{
			Parameters: r.Parameters,
		})
		require.NoError(t, err)
		require.Equal(t, r, got)
	})
}

func TestProcessRunAttached(t *testing.T) {
	app := "app1"
	service := "service"
	s := stdapi.New("api", "api")
	s.Route("SOCKET", fmt.Sprintf("/apps/%s/processes/%s/run", app, service), func(c *stdapi.Context) error {
		require.Equal(t, "test", c.Header("Command"))
		require.Equal(t, "1", c.Header("Height"))
		require.Equal(t, "2", c.Header("Width"))
		require.Equal(t, "10", c.Header("Timeout"))
		_, b, err := c.Websocket().ReadMessage()
		require.NoError(t, err)
		require.Equal(t, "cmd", string(b))
		return c.Websocket().WriteMessage(websocket.TextMessage, []byte("data"+statusCodePrefix+"0"))
	})

	testServer(t, s, func(c *sdk.Client) {
		rw := bufio.NewReadWriter(bufio.NewReader(bytes.NewReader([]byte("cmd"))), bufio.NewWriter(&bytes.Buffer{}))
		got, err := c.ProcessRunAttached(app, service, rw, 10, structs.ProcessRunOptions{
			Command: options.String("test"),
			Height:  options.Int(1),
			Width:   options.Int(2),
		})
		require.NoError(t, err)
		require.Equal(t, 0, got)
	})
}
