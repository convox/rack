package cli_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	mockstart "github.com/convox/rack/pkg/mock/start"
	mockstdcli "github.com/convox/rack/pkg/mock/stdcli"
	"github.com/convox/rack/pkg/start"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var fxSystemLocal = structs.System{
	Name:     "convox",
	Provider: "local",
	Status:   "running",
	Version:  "dev",
}

func TestStart1(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		ms := &mockstart.Interface{}
		cli.Starter = ms

		opts := start.Options{
			App:   "app1",
			Build: true,
			Cache: true,
			Sync:  true,
		}

		ms.On("Start1", opts).Return(nil)

		i.On("SystemGet").Return(&fxSystemLocal, nil)

		res, err := testExecute(e, "start -g 1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{""})
	})
}

func TestStart1Error(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		ms := &mockstart.Interface{}
		cli.Starter = ms

		opts := start.Options{
			App:   "app1",
			Build: true,
			Cache: true,
			Sync:  true,
		}

		ms.On("Start1", opts).Return(fmt.Errorf("err1"))

		i.On("SystemGet").Return(&fxSystemLocal, nil)

		res, err := testExecute(e, "start -g 1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestStart1Options(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		ms := &mockstart.Interface{}
		cli.Starter = ms

		opts := start.Options{
			App:      "app1",
			Build:    false,
			Cache:    false,
			Command:  []string{"bin/command", "args"},
			Manifest: "manifest1",
			Services: []string{"service1"},
			Shift:    3000,
			Sync:     false,
		}

		ms.On("Start1", opts).Return(nil)

		i.On("SystemGet").Return(&fxSystemLocal, nil)

		res, err := testExecute(e, "start -g 1 -a app1 -m manifest1 --no-build --no-cache --no-sync -s 3000 service1 bin/command args", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{""})
	})
}

func TestStart2(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		me := &mockstdcli.Executor{}
		me.On("Execute", "docker", "ps", "--filter", "label=convox.type=rack", "--format", "{{.Names}}").Return([]byte("classic\n"), nil)
		me.On("Execute", "kubectl", "get", "ns", "--selector=system=convox,type=rack", "--output=name").Return([]byte("namespace/dev\n"), nil)
		e.Executor = me

		ms := &mockstart.Interface{}
		cli.Starter = ms

		opts := start.Options{
			App:   "app1",
			Build: true,
			Cache: true,
			Sync:  true,
		}

		ms.On("Start2", i, opts).Return(nil)

		i.On("SystemGet").Return(&fxSystemLocal, nil)

		res, err := testExecute(e, "start -g 2 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{""})
	})
}

func TestStart2Error(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		me := &mockstdcli.Executor{}
		me.On("Execute", "docker", "ps", "--filter", "label=convox.type=rack", "--format", "{{.Names}}").Return([]byte("classic\n"), nil)
		me.On("Execute", "kubectl", "get", "ns", "--selector=system=convox,type=rack", "--output=name").Return([]byte("namespace/dev\n"), nil)
		e.Executor = me

		ms := &mockstart.Interface{}
		cli.Starter = ms

		opts := start.Options{
			App:   "app1",
			Build: true,
			Cache: true,
			Sync:  true,
		}

		ms.On("Start2", i, opts).Return(fmt.Errorf("err1"))

		i.On("SystemGet").Return(&fxSystemLocal, nil)

		res, err := testExecute(e, "start -g 2 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestStart2Options(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		me := &mockstdcli.Executor{}
		me.On("Execute", "docker", "ps", "--filter", "label=convox.type=rack", "--format", "{{.Names}}").Return([]byte("classic\n"), nil)
		me.On("Execute", "kubectl", "get", "ns", "--selector=system=convox,type=rack", "--output=name").Return([]byte("namespace/dev\n"), nil)
		e.Executor = me

		ms := &mockstart.Interface{}
		cli.Starter = ms

		opts := start.Options{
			App:      "app1",
			Build:    false,
			Cache:    false,
			Manifest: "manifest1",
			Services: []string{"service1", "service2"},
			Sync:     false,
		}

		ms.On("Start2", i, opts).Return(nil)

		i.On("SystemGet").Return(&fxSystemLocal, nil)

		res, err := testExecute(e, "start -g 2 -a app1 -m manifest1 --no-build --no-cache --no-sync service1 service2", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{""})
	})
}

func TestStart2Remote(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		me := &mockstdcli.Executor{}
		me.On("Execute", "docker", "ps", "--filter", "label=convox.type=rack", "--format", "{{.Names}}").Return([]byte("classic\n"), nil)
		me.On("Execute", "kubectl", "get", "ns", "--selector=system=convox,type=rack", "--output=name").Return([]byte(""), nil)
		e.Executor = me

		ms := &mockstart.Interface{}
		cli.Starter = ms

		opts := start.Options{
			App:   "app1",
			Build: true,
			Cache: true,
			Sync:  true,
		}

		ms.On("Start2", mock.Anything, opts).Return(nil).Run(func(args mock.Arguments) {
			s := args.Get(0).(*sdk.Client)
			require.Equal(t, "https", s.Client.Endpoint.Scheme)
			require.Equal(t, "rack.classic", s.Client.Endpoint.Host)
		})

		i.On("SystemGet").Return(&fxSystem, nil)

		res, err := testExecute(e, "start -g 2 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{""})
	})
}

func TestStart2RemoteMultiple(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		me := &mockstdcli.Executor{}
		me.On("Execute", "docker", "ps", "--filter", "label=convox.type=rack", "--format", "{{.Names}}").Return([]byte("classic\n"), nil)
		me.On("Execute", "kubectl", "get", "ns", "--selector=system=convox,type=rack", "--output=name").Return([]byte("namespace/dev\n"), nil)
		e.Executor = me

		ms := &mockstart.Interface{}
		cli.Starter = ms

		opts := start.Options{
			App:   "app1",
			Build: true,
			Cache: true,
			Sync:  true,
		}

		ms.On("Start2", mock.Anything, opts).Return(nil).Run(func(args mock.Arguments) {
			s := args.Get(0).(*sdk.Client)
			require.Equal(t, "https", s.Client.Endpoint.Scheme)
			require.Equal(t, "rack.classic", s.Client.Endpoint.Host)
		})

		i.On("SystemGet").Return(&fxSystem, nil)

		res, err := testExecute(e, "start -g 2 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: multiple local racks detected, use `convox switch` to select one"})
		res.RequireStdout(t, []string{""})
	})
}
