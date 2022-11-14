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
	mockstdcli "github.com/convox/rack/pkg/mock/stdcli"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestRacks(t *testing.T) {
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

		me := &mockstdcli.Executor{}
		me.On("Execute", "kubectl", "get", "ns", "--selector=system=convox,type=rack", "--output=name").Return([]byte("namespace/dev\n"), nil)
		e.Executor = me

		res, err := testExecute(e, "racks", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"NAME        STATUS",
			"local/dev   running",
			"test/foo    running",
			"test/other  updating",
		})

		me.AssertExpectations(t)
	})
}

func TestRacksLocalDisable(t *testing.T) {
	orig := os.Getenv("CONVOX_LOCAL")
	os.Setenv("CONVOX_LOCAL", "disable")
	defer os.Setenv("CONVOX_LOCAL", orig)

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

		me := &mockstdcli.Executor{}
		e.Executor = me

		res, err := testExecute(e, "racks", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"NAME        STATUS",
			"test/foo    running",
			"test/other  updating",
		})

		me.AssertExpectations(t)
	})
}

func TestRacksError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		r := mux.NewRouter()

		r.HandleFunc("/racks", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			w.Write([]byte("test"))
		}).Methods("GET")

		ts := httptest.NewTLSServer(r)

		tsu, err := url.Parse(ts.URL)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(e.Settings, "host"), []byte(tsu.Host), 0644)
		require.NoError(t, err)

		me := &mockstdcli.Executor{}
		me.On("Execute", "kubectl", "get", "ns", "--selector=system=convox,type=rack", "--output=name").Return(nil, fmt.Errorf("err1"))
		e.Executor = me

		res, err := testExecute(e, "racks", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"NAME  STATUS",
		})
	})
}
