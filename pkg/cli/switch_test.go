package cli_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
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

		tmp, err := ioutil.TempDir("", "")
		require.NoError(t, err)
		e.Settings = tmp

		err = ioutil.WriteFile(filepath.Join(tmp, "host"), []byte(tsu.Host), 0644)
		require.NoError(t, err)

		res, err := testExecute(e, "switch foo", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Switched to test/foo"})

		data, err := ioutil.ReadFile(filepath.Join(tmp, "racks"))
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

		tmp, err := ioutil.TempDir("", "")
		require.NoError(t, err)
		e.Settings = tmp

		err = ioutil.WriteFile(filepath.Join(tmp, "host"), []byte(tsu.Host), 0644)
		require.NoError(t, err)

		res, err := testExecute(e, "switch rack1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: could not find rack: rack1"})
		res.RequireStdout(t, []string{""})
	})
}
