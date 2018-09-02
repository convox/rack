package cli_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
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
