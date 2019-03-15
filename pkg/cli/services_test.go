package cli_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/require"
)

func TestServices(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ServiceList", "app1").Return(structs.Services{*fxService(), *fxService()}, nil)

		res, err := testExecute(e, "services -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"SERVICE   DOMAIN  PORTS  ",
			"service1  domain  1:2 1:2",
			"service1  domain  1:2 1:2",
		})
	})
}

func TestServicesError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ServiceList", "app1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "services -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestServicesClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemClassic(), nil)
		i.On("FormationGet", "app1").Return(structs.Services{*fxService(), *fxService()}, nil)

		res, err := testExecute(e, "services -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"SERVICE   DOMAIN  PORTS  ",
			"service1  domain  1:2 1:2",
			"service1  domain  1:2 1:2",
		})
	})
}
