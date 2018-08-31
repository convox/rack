package cli_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/require"
)

func TestScale(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystem, nil)
		i.On("ServiceList", "app1").Return(structs.Services{fxService, fxService}, nil)
		i.On("ProcessList", "app1", structs.ProcessListOptions{}).Return(structs.Processes{fxProcess, fxProcess}, nil)

		res, err := testExecute(e, "scale -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"SERVICE   DESIRED  RUNNING  CPU  MEMORY",
			"service1  1        0        2    3     ",
			"service1  1        0        2    3     ",
		})
	})
}

func TestScaleError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystem, nil)
		i.On("ServiceList", "app1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "scale -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestScaleClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystemClassic, nil)
		i.On("FormationGet", "app1").Return(structs.Services{fxService, fxService}, nil)
		i.On("ProcessList", "app1", structs.ProcessListOptions{}).Return(structs.Processes{fxProcess, fxProcess}, nil)

		res, err := testExecute(e, "scale -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"SERVICE   DESIRED  RUNNING  CPU  MEMORY",
			"service1  1        0        2    3     ",
			"service1  1        0        2    3     ",
		})
	})
}

func TestScaleUpdate(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystem, nil)
		i.On("ServiceUpdate", "app1", "web", structs.ServiceUpdateOptions{Count: options.Int(3), Cpu: options.Int(5), Memory: options.Int(10)}).Return(nil)

		res, err := testExecute(e, "scale web --cpu 5 --memory 10 --count 3 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Scaling web... OK"})
	})
}

func TestScaleUpdateError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystem, nil)
		i.On("ServiceUpdate", "app1", "web", structs.ServiceUpdateOptions{Count: options.Int(3), Cpu: options.Int(5), Memory: options.Int(10)}).Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "scale web --cpu 5 --memory 10 --count 3 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Scaling web... "})
	})
}

func TestScaleUpdateClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystemClassic, nil)
		i.On("FormationUpdate", "app1", "web", structs.ServiceUpdateOptions{Count: options.Int(3), Cpu: options.Int(5), Memory: options.Int(10)}).Return(nil)

		res, err := testExecute(e, "scale web --cpu 5 --memory 10 --count 3 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Scaling web... OK"})
	})
}
