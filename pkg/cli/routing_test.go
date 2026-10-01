package cli_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/require"
)

type fakeRequest struct {
	Path     string
	Rack     string
	Password string
}

type fakeHost struct {
	Host string

	lock     sync.Mutex
	requests []fakeRequest
}

func newFakeHost(t *testing.T, fn http.HandlerFunc) *fakeHost {
	f := &fakeHost{}

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pw, _ := r.BasicAuth()

		f.lock.Lock()
		f.requests = append(f.requests, fakeRequest{Path: r.URL.Path, Rack: r.Header.Get("Rack"), Password: pw})
		f.lock.Unlock()

		fn(w, r)
	}))
	t.Cleanup(ts.Close)

	u, err := url.Parse(ts.URL)
	require.NoError(t, err)

	f.Host = u.Host

	return f
}

func (f *fakeHost) Requests() []fakeRequest {
	f.lock.Lock()
	defer f.lock.Unlock()

	return append([]fakeRequest{}, f.requests...)
}

func (f *fakeHost) RackRequests() []fakeRequest {
	rs := []fakeRequest{}

	for _, r := range f.Requests() {
		if r.Rack != "" {
			rs = append(rs, r)
		}
	}

	return rs
}

func fakeRack(t *testing.T, name string) *fakeHost {
	return newFakeHost(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/system" {
			http.NotFound(w, r)
			return
		}

		writeJSON(w, structs.System{Name: name, Provider: "aws", Region: "us-east-2", Status: "running", Version: "20260929232050"})
	})
}

func fakeConsole(t *testing.T, racks ...string) *fakeHost {
	return newFakeHost(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/racks":
			list := []map[string]interface{}{}
			for _, rack := range racks {
				parts := strings.SplitN(rack, "/", 2)
				list = append(list, map[string]interface{}{"name": parts[1], "organization": map[string]string{"name": parts[0]}, "status": "running"})
			}
			writeJSON(w, list)
		case "/system":
			rack := r.Header.Get("Rack")
			for _, name := range racks {
				if name == rack {
					writeJSON(w, structs.System{Name: strings.SplitN(name, "/", 2)[1], Provider: "aws", Status: "running", Version: "20260929232050"})
					return
				}
			}
			w.WriteHeader(403)
			writeJSON(w, map[string]string{"error": "no such rack: " + rack})
		default:
			http.NotFound(w, r)
		}
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	_ = json.NewEncoder(w).Encode(v)
}

func writeSettings(t *testing.T, e *cli.Engine, files map[string]interface{}) {
	for name, v := range files {
		data, ok := v.(string)
		if !ok {
			raw, err := json.Marshal(v)
			require.NoError(t, err)
			data = string(raw)
		}

		require.NoError(t, os.WriteFile(filepath.Join(e.Settings, name), []byte(data), 0600))
	}
}

func readSettingsJSON(t *testing.T, e *cli.Engine, name string) map[string]string {
	data, err := os.ReadFile(filepath.Join(e.Settings, name))
	require.NoError(t, err)

	var kv map[string]string
	require.NoError(t, json.Unmarshal(data, &kv))

	return kv
}

func testRouting(t *testing.T, fn func(e *cli.Engine, console, rack *fakeHost)) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		e.Client = nil

		t.Setenv("CONVOX_LOCAL", "disable")

		console := fakeConsole(t, "test/foo")
		rack := fakeRack(t, "v2-scoped-rc")

		writeSettings(t, e, map[string]interface{}{
			"host":         console.Host,
			"auth":         map[string]string{console.Host: "consolepw", rack.Host: "rackpw"},
			"self-managed": map[string]string{"v2-scoped-rc": rack.Host},
		})

		fn(e, console, rack)
	})
}

var fxRackOutput = []string{
	"Name      v2-scoped-rc",
	"Provider  aws",
	"Region    us-east-2",
	"Status    running",
	"Version   20260929232050",
}

func TestRackFlagSelfManaged(t *testing.T) {
	testRouting(t, func(e *cli.Engine, console, rack *fakeHost) {
		res, err := testExecute(e, "rack -r v2-scoped-rc", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, fxRackOutput)

		require.Equal(t, []fakeRequest{{Path: "/system", Rack: "v2-scoped-rc", Password: "rackpw"}}, rack.Requests())
		require.Empty(t, console.Requests())
	})
}

func TestRackFlagConsoleName(t *testing.T) {
	testRouting(t, func(e *cli.Engine, console, rack *fakeHost) {
		res, err := testExecute(e, "rack -r test/foo", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)

		require.Equal(t, []fakeRequest{{Path: "/system", Rack: "test/foo", Password: "consolepw"}}, console.Requests())
		require.Empty(t, rack.Requests())
	})
}

func TestRackFlagPartialGoesToConsole(t *testing.T) {
	testRouting(t, func(e *cli.Engine, console, rack *fakeHost) {
		res, err := testExecute(e, "rack -r scoped", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: no such rack: scoped"})

		require.Equal(t, []fakeRequest{{Path: "/system", Rack: "scoped", Password: "consolepw"}}, console.Requests())
		require.Empty(t, rack.Requests())
	})
}

func TestRackFlagExactBeatsConsole(t *testing.T) {
	testRouting(t, func(e *cli.Engine, console, rack *fakeHost) {
		writeSettings(t, e, map[string]interface{}{
			"self-managed": map[string]string{"foo": rack.Host},
		})

		res, err := testExecute(e, "rack -r foo", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)

		require.Equal(t, []fakeRequest{{Path: "/system", Rack: "foo", Password: "rackpw"}}, rack.Requests())
		require.Empty(t, console.Requests())
	})
}

func TestRackFlagOtherCLIInstalledRack(t *testing.T) {
	testRouting(t, func(e *cli.Engine, console, rack *fakeHost) {
		a := fakeRack(t, "a")
		b := fakeRack(t, "b")

		writeSettings(t, e, map[string]interface{}{
			"host":         a.Host,
			"auth":         map[string]string{a.Host: "apw", b.Host: "bpw"},
			"self-managed": map[string]string{"a": a.Host, "b": b.Host},
		})

		res, err := testExecute(e, "rack -r b", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)

		require.Equal(t, []fakeRequest{{Path: "/system", Rack: "b", Password: "bpw"}}, b.Requests())
		require.Empty(t, a.Requests())
	})
}

func TestRackFlagConvoxHostPins(t *testing.T) {
	testRouting(t, func(e *cli.Engine, console, rack *fakeHost) {
		t.Setenv("CONVOX_HOST", console.Host)

		res, err := testExecute(e, "rack -r v2-scoped-rc", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: no such rack: v2-scoped-rc"})

		require.Equal(t, []fakeRequest{{Path: "/system", Rack: "v2-scoped-rc", Password: "consolepw"}}, console.Requests())
		require.Empty(t, rack.Requests())
	})
}

func TestRackFlagRackURLWins(t *testing.T) {
	testRouting(t, func(e *cli.Engine, console, rack *fakeHost) {
		t.Setenv("RACK_URL", "https://convox:urlpw@"+rack.Host)

		res, err := testExecute(e, "rack -r test/foo", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)

		res, err = testExecute(e, "rack -r v2-scoped-rc", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)

		require.Equal(t, []fakeRequest{
			{Path: "/system", Rack: "test/foo", Password: "urlpw"},
			{Path: "/system", Rack: "v2-scoped-rc", Password: "urlpw"},
		}, rack.Requests())
		require.Empty(t, console.Requests())
	})
}

func TestRackFlagConvoxRack(t *testing.T) {
	testRouting(t, func(e *cli.Engine, console, rack *fakeHost) {
		t.Setenv("CONVOX_RACK", "v2-scoped-rc")

		res, err := testExecute(e, "rack", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, fxRackOutput)

		require.Equal(t, []fakeRequest{{Path: "/system", Rack: "v2-scoped-rc", Password: "rackpw"}}, rack.Requests())
		require.Empty(t, console.Requests())
	})
}

func TestRackFlagLocalSetting(t *testing.T) {
	testRouting(t, func(e *cli.Engine, console, rack *fakeHost) {
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, ".convox"), 0700))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".convox", "rack"), []byte("v2-scoped-rc\n"), 0600))
		t.Chdir(dir)

		res, err := testExecute(e, "rack", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, fxRackOutput)

		require.Equal(t, []fakeRequest{{Path: "/system", Rack: "v2-scoped-rc", Password: "rackpw"}}, rack.Requests())
		require.Empty(t, console.Requests())
	})
}

func TestRackFlagConvoxPasswordStaysWithLoginHost(t *testing.T) {
	testRouting(t, func(e *cli.Engine, console, rack *fakeHost) {
		t.Setenv("CONVOX_PASSWORD", "override")

		res, err := testExecute(e, "rack -r v2-scoped-rc", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)

		require.Equal(t, []fakeRequest{{Path: "/system", Rack: "v2-scoped-rc", Password: "rackpw"}}, rack.Requests())
		require.Empty(t, console.Requests())
	})
}

func TestRackFlagConvoxPasswordLoginRack(t *testing.T) {
	testRouting(t, func(e *cli.Engine, console, rack *fakeHost) {
		writeSettings(t, e, map[string]interface{}{"host": rack.Host})
		t.Setenv("CONVOX_PASSWORD", "override")

		res, err := testExecute(e, "rack -r v2-scoped-rc", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)

		require.Equal(t, []fakeRequest{{Path: "/system", Rack: "v2-scoped-rc", Password: "override"}}, rack.Requests())
	})
}

func TestRackFlagNoLoginHost(t *testing.T) {
	testRouting(t, func(e *cli.Engine, console, rack *fakeHost) {
		require.NoError(t, os.Remove(filepath.Join(e.Settings, "host")))

		res, err := testExecute(e, "rack -r v2-scoped-rc", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, fxRackOutput)

		require.Equal(t, []fakeRequest{{Path: "/system", Rack: "v2-scoped-rc", Password: "rackpw"}}, rack.Requests())
		require.Empty(t, console.Requests())
	})
}
