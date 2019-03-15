package cli_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/require"
)

func TestRegistries(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("RegistryList").Return(structs.Registries{*fxRegistry(), *fxRegistry()}, nil)

		res, err := testExecute(e, "registries", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"SERVER     USERNAME",
			"registry1  username",
			"registry1  username",
		})
	})
}

func TestRegistriesError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("RegistryList").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "registries", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestRegistriesAdd(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("RegistryAdd", "foo", "bar", "baz").Return(fxRegistry(), nil)

		res, err := testExecute(e, "registries add foo bar baz", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Adding registry... OK"})
	})
}

func TestRegistriesAddError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("RegistryAdd", "foo", "bar", "baz").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "registries add foo bar baz", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Adding registry... "})
	})
}

func TestRegistriesRemove(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("RegistryRemove", "foo").Return(nil)

		res, err := testExecute(e, "registries remove foo", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Removing registry... OK"})
	})
}

func TestRegistriesRemoveError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("RegistryRemove", "foo").Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "registries remove foo", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Removing registry... "})
	})
}

func TestRegistriesRemoveClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemClassic(), nil)
		i.On("RegistryRemoveClassic", "foo").Return(nil)

		res, err := testExecute(e, "registries remove foo", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Removing registry... OK"})
	})
}
