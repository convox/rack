package cli_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/require"
)

func TestRestart(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("ServiceList", "app1").Return(structs.Services{*fxService(), *fxService()}, nil)
		i.On("ServiceRestart", "app1", "service1").Return(nil).Twice()

		res, err := testExecute(e, "restart -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Restarting service1... OK",
			"Restarting service1... OK",
		})
	})
}

func TestRestartErrorList(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("ServiceList", "app1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "restart -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestRestartErrorRestart(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("ServiceList", "app1").Return(structs.Services{*fxService(), *fxService()}, nil)
		i.On("ServiceRestart", "app1", "service1").Return(fmt.Errorf("err1")).Once()

		res, err := testExecute(e, "restart -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Restarting service1... "})
	})
}
