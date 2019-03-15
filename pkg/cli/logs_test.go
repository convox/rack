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

func TestLogs(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppLogs", "app1", structs.LogsOptions{Prefix: options.Bool(true)}).Return(testLogs(fxLogs()), nil)

		res, err := testExecute(e, "logs -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			fxLogs()[0],
			fxLogs()[1],
		})
	})
}

func TestLogsError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppLogs", "app1", structs.LogsOptions{Prefix: options.Bool(true)}).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "logs -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}
