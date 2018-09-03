package cli_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	mockstdcli "github.com/convox/rack/pkg/mock/stdcli"
	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		me := &mockstdcli.Executor{}
		me.On("Execute", "docker", "ps", "--filter", "label=convox.type=rack", "--format", "{{.Names}}").Return([]byte("classic\n"), nil)
		me.On("Execute", "kubectl", "get", "ns", "--selector=system=convox,type=rack", "--output=name").Return([]byte("namespace/dev\n"), nil)
		e.Executor = me

		err := ioutil.WriteFile(filepath.Join(e.Settings, "host"), []byte("host1"), 0644)
		require.NoError(t, err)

		i.On("SystemGet").Return(&fxSystem, nil)

		res, err := testExecute(e, "version", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"client: test",
			"server: 20180901000000 (host1)",
		})
	})
}

func TestVersionError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		me := &mockstdcli.Executor{}
		me.On("Execute", "docker", "ps", "--filter", "label=convox.type=rack", "--format", "{{.Names}}").Return([]byte("classic\n"), nil)
		me.On("Execute", "kubectl", "get", "ns", "--selector=system=convox,type=rack", "--output=name").Return([]byte("namespace/dev\n"), nil)
		e.Executor = me

		err := ioutil.WriteFile(filepath.Join(e.Settings, "host"), []byte("host1"), 0644)
		require.NoError(t, err)

		i.On("SystemGet").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "version", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{
			"client: test",
		})
	})
}

func TestVersionNoSystem(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		me := &mockstdcli.Executor{}
		me.On("Execute", "docker", "ps", "--filter", "label=convox.type=rack", "--format", "{{.Names}}").Return([]byte(""), nil)
		me.On("Execute", "kubectl", "get", "ns", "--selector=system=convox,type=rack", "--output=name").Return([]byte(""), nil)
		e.Executor = me

		res, err := testExecute(e, "version", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"client: test",
			"server: none",
		})
	})
}

func TestVersionNoSystemMultipleLocal(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		me := &mockstdcli.Executor{}
		me.On("Execute", "docker", "ps", "--filter", "label=convox.type=rack", "--format", "{{.Names}}").Return([]byte("classic\n"), nil)
		me.On("Execute", "kubectl", "get", "ns", "--selector=system=convox,type=rack", "--output=name").Return([]byte("namespace/dev\n"), nil)
		e.Executor = me

		res, err := testExecute(e, "version", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"client: test",
			"server: none",
		})
	})
}

func TestVersionNoSystemSingleLocal(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		me := &mockstdcli.Executor{}
		me.On("Execute", "docker", "ps", "--filter", "label=convox.type=rack", "--format", "{{.Names}}").Return([]byte(""), nil)
		me.On("Execute", "kubectl", "get", "ns", "--selector=system=convox,type=rack", "--output=name").Return([]byte("namespace/dev\n"), nil)
		e.Executor = me

		i.On("SystemGet").Return(&fxSystemLocal, nil)

		res, err := testExecute(e, "version", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"client: test",
			"server: dev (rack.dev)",
		})
	})
}
