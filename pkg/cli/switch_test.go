package cli_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestSwitch(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		r := mux.NewRouter()

		r.HandleFunc("/racks", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`[
				{"name":"foo","organization":{"name":"test"},"status":"running"},
				{"name":"other","organization":{"name":"test"},"status":"updating"}
			]`))
		}).Methods("GET")

		ts := httptest.NewTLSServer(r)

		tsu, err := url.Parse(ts.URL)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(e.Settings, "host"), []byte(tsu.Host), 0644)
		require.NoError(t, err)

		res, err := testExecute(e, "switch foo", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Switched to test/foo"})

		data, err := os.ReadFile(filepath.Join(e.Settings, "racks"))
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("{\n  \"%s\": \"test/foo\"\n}", tsu.Host), string(data))
	})
}

func TestSwitchUnknown(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		r := mux.NewRouter()

		r.HandleFunc("/racks", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`[
				{"name":"foo","organization":{"name":"test"},"status":"running"},
				{"name":"other","organization":{"name":"test"},"status":"updating"}
			]`))
		}).Methods("GET")

		ts := httptest.NewTLSServer(r)

		tsu, err := url.Parse(ts.URL)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(e.Settings, "host"), []byte(tsu.Host), 0644)
		require.NoError(t, err)

		res, err := testExecute(e, "switch rack1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: could not find rack: rack1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestSwitchCLIInstalledKeepsConsole(t *testing.T) {
	testRouting(t, func(e *cli.Engine, console, rack *fakeHost) {
		res, err := testExecute(e, "switch v2-scoped-rc", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Switched to v2-scoped-rc"})

		data, err := os.ReadFile(filepath.Join(e.Settings, "host"))
		require.NoError(t, err)
		require.Equal(t, console.Host, string(data))
		require.Equal(t, map[string]string{console.Host: "v2-scoped-rc"}, readSettingsJSON(t, e, "racks"))

		res, err = testExecute(e, "switch", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, []string{"v2-scoped-rc"})

		res, err = testExecute(e, "rack", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, fxRackOutput)
		require.Equal(t, []fakeRequest{{Path: "/system", Rack: "v2-scoped-rc", Password: "rackpw"}}, rack.RackRequests())

		res, err = testExecute(e, "racks", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, []string{
			"NAME          STATUS",
			"test/foo      running",
			"v2-scoped-rc  running",
		})
	})
}

func TestSwitchCLIInstalledThenConsole(t *testing.T) {
	testRouting(t, func(e *cli.Engine, console, rack *fakeHost) {
		res, err := testExecute(e, "switch v2-scoped-rc", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)

		res, err = testExecute(e, "switch test/foo", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, []string{"Switched to test/foo"})
		require.Equal(t, map[string]string{console.Host: "test/foo"}, readSettingsJSON(t, e, "racks"))

		res, err = testExecute(e, "rack", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Equal(t, []fakeRequest{{Path: "/system", Rack: "test/foo", Password: "consolepw"}}, console.RackRequests())
		require.Empty(t, rack.RackRequests())
	})
}

func TestSwitchCLIInstalledNotice(t *testing.T) {
	testRouting(t, func(e *cli.Engine, _, rack *fakeHost) {
		console := fakeConsole(t, "test/foo", "test/foo-app", "test/other")

		writeSettings(t, e, map[string]interface{}{
			"host":         console.Host,
			"auth":         map[string]string{console.Host: "consolepw", rack.Host: "rackpw"},
			"self-managed": map[string]string{"foo": rack.Host},
		})

		res, err := testExecute(e, "switch foo", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, []string{"Switched to foo"})
		res.RequireStderr(t, []string{"NOTICE: Console racks also match foo: test/foo, test/foo-app. To select one, use its full name from `convox racks`, for example `convox switch test/foo`"})

		data, err := os.ReadFile(filepath.Join(e.Settings, "host"))
		require.NoError(t, err)
		require.Equal(t, console.Host, string(data))
		require.Equal(t, map[string]string{console.Host: "foo"}, readSettingsJSON(t, e, "racks"))
	})
}

func TestSwitchCLIInstalledConvoxHost(t *testing.T) {
	testRouting(t, func(e *cli.Engine, console, rack *fakeHost) {
		require.NoError(t, os.Remove(filepath.Join(e.Settings, "host")))
		t.Setenv("CONVOX_HOST", console.Host)

		res, err := testExecute(e, "switch v2-scoped-rc", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: unset CONVOX_HOST to switch to v2-scoped-rc"})
		res.RequireStdout(t, []string{""})

		require.NoFileExists(t, filepath.Join(e.Settings, "host"))
		require.NoFileExists(t, filepath.Join(e.Settings, "racks"))

		res, err = testExecute(e, "switch test/foo", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, []string{"Switched to test/foo"})
		require.Equal(t, map[string]string{console.Host: "test/foo"}, readSettingsJSON(t, e, "racks"))
	})
}

func TestSwitchCLIInstalledNoConsole(t *testing.T) {
	testRouting(t, func(e *cli.Engine, console, rack *fakeHost) {
		a := fakeRack(t, "a")
		b := fakeRack(t, "b")

		writeSettings(t, e, map[string]interface{}{
			"host":         a.Host,
			"auth":         map[string]string{a.Host: "apw", b.Host: "bpw"},
			"self-managed": map[string]string{"a": a.Host, "b": b.Host},
		})

		res, err := testExecute(e, "switch b", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Switched to b"})

		data, err := os.ReadFile(filepath.Join(e.Settings, "host"))
		require.NoError(t, err)
		require.Equal(t, b.Host, string(data))
		require.Equal(t, map[string]string{b.Host: "b"}, readSettingsJSON(t, e, "racks"))

		res, err = testExecute(e, "switch", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStdout(t, []string{"b"})

		res, err = testExecute(e, "rack", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Equal(t, []fakeRequest{{Path: "/system", Rack: "b", Password: "bpw"}}, b.RackRequests())
		require.Empty(t, a.RackRequests())
	})
}
