package cli_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/require"
)

var fxApp = structs.App{
	Name:       "app1",
	Generation: "2",
	Release:    "release1",
	Status:     "running",
}

func TestApps(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		a1 := structs.Apps{
			fxApp,
			structs.App{
				Name:       "app2",
				Generation: "1",
				Status:     "creating",
			},
		}
		i.On("AppList").Return(a1, nil)

		res, err := testExecute(e, "apps", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 3, res.StdoutLines())
		require.Equal(t, "APP   STATUS    GEN  RELEASE ", res.StdoutLine(0))
		require.Equal(t, "app1  running   2    release1", res.StdoutLine(1))
		require.Equal(t, "app2  creating  1            ", res.StdoutLine(2))
	})
}

func TestAppsError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppList").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "apps", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		require.Equal(t, "ERROR: err1\n", res.Stderr)
		require.Equal(t, "", res.Stdout)
	})
}

func TestAppsCancel(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppCancel", "app1").Return(nil)

		res, err := testExecute(e, "apps cancel app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 1, res.StdoutLines())
		require.Equal(t, "Cancelling app1... OK", res.StdoutLine(0))

		res, err = testExecute(e, "apps cancel -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 1, res.StdoutLines())
		require.Equal(t, "Cancelling app1... OK", res.StdoutLine(0))
	})
}

func TestAppsCancelError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppCancel", "app1").Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "apps cancel app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		require.Equal(t, "ERROR: err1\n", res.Stderr)
		require.Equal(t, 1, res.StdoutLines())
		require.Equal(t, "Cancelling app1... ", res.StdoutLine(0))
	})
}
