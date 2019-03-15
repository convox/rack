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
	"github.com/stretchr/testify/require"
)

func TestLogin(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/auth", r.URL.Path)
			user, pass, _ := r.BasicAuth()
			require.Equal(t, "convox", user)
			require.Equal(t, "password", pass)
		}))

		tsu, err := url.Parse(ts.URL)
		require.NoError(t, err)

		res, err := testExecute(e, fmt.Sprintf("login %s -p password", tsu.Host), nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{fmt.Sprintf("Authenticating with %s... OK", tsu.Host)})

		data, err := ioutil.ReadFile(filepath.Join(e.Settings, "auth"))
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("{\n  \"%s\": \"password\"\n}", tsu.Host), string(data))

		data, err = ioutil.ReadFile(filepath.Join(e.Settings, "host"))
		require.NoError(t, err)
		require.Equal(t, tsu.Host, string(data))
	})
}

func TestLoginError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
		}))

		tsu, err := url.Parse(ts.URL)
		require.NoError(t, err)

		res, err := testExecute(e, fmt.Sprintf("login %s -p password", tsu.Host), nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: invalid login"})
		res.RequireStdout(t, []string{fmt.Sprintf("Authenticating with %s... ", tsu.Host)})
	})
}
