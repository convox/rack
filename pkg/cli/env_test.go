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

func TestEnv(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.ReleaseListOptions{Limit: options.Int(1)}
		i.On("ReleaseList", "app1", opts).Return(structs.Releases{*fxRelease()}, nil)
		i.On("ReleaseGet", "app1", "release1").Return(fxRelease(), nil)

		res, err := testExecute(e, "env -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"BAZ=quux",
			"FOO=bar",
		})
	})
}

func TestEnvError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.ReleaseListOptions{Limit: options.Int(1)}
		i.On("ReleaseList", "app1", opts).Return(structs.Releases{*fxRelease()}, nil)
		i.On("ReleaseGet", "app1", "release1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "env -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestEnvGet(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.ReleaseListOptions{Limit: options.Int(1)}
		i.On("ReleaseList", "app1", opts).Return(structs.Releases{*fxRelease()}, nil)
		i.On("ReleaseGet", "app1", "release1").Return(fxRelease(), nil)

		res, err := testExecute(e, "env get FOO -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"bar"})
	})
}

func TestEnvGetError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.ReleaseListOptions{Limit: options.Int(1)}
		i.On("ReleaseList", "app1", opts).Return(structs.Releases{*fxRelease()}, nil)
		i.On("ReleaseGet", "app1", "release1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "env get FOO -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestEnvGetMissing(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.ReleaseListOptions{Limit: options.Int(1)}
		i.On("ReleaseList", "app1", opts).Return(structs.Releases{*fxRelease()}, nil)
		i.On("ReleaseGet", "app1", "release1").Return(fxRelease(), nil)

		res, err := testExecute(e, "env get FOOO -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: env not found: FOOO"})
		res.RequireStdout(t, []string{""})
	})
}

func TestEnvSet(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		opts := structs.ReleaseListOptions{Limit: options.Int(1)}
		i.On("ReleaseList", "app1", opts).Return(structs.Releases{*fxRelease()}, nil)
		i.On("ReleaseGet", "app1", "release1").Return(fxRelease(), nil)
		ropts := structs.ReleaseCreateOptions{Env: options.String("AAA=bbb\nBAZ=quux\nCCC=ddd\nFOO=bar")}
		i.On("ReleaseCreate", "app1", ropts).Return(fxRelease(), nil)

		res, err := testExecute(e, "env set AAA=bbb CCC=ddd -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Setting AAA, CCC... OK",
			"Release: release1",
		})
	})
}

func TestEnvSetError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		opts := structs.ReleaseListOptions{Limit: options.Int(1)}
		i.On("ReleaseList", "app1", opts).Return(structs.Releases{*fxRelease()}, nil)
		i.On("ReleaseGet", "app1", "release1").Return(fxRelease(), nil)
		ropts := structs.ReleaseCreateOptions{Env: options.String("AAA=bbb\nBAZ=quux\nCCC=ddd\nFOO=bar")}
		i.On("ReleaseCreate", "app1", ropts).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "env set AAA=bbb CCC=ddd -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Setting AAA, CCC... "})
	})
}

func TestEnvSetClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemClassic(), nil)
		opts := structs.ReleaseListOptions{Limit: options.Int(1)}
		i.On("ReleaseList", "app1", opts).Return(structs.Releases{*fxRelease()}, nil)
		i.On("ReleaseGet", "app1", "release1").Return(fxRelease(), nil)
		i.On("EnvironmentSet", "app1", []byte("AAA=bbb\nBAZ=quux\nCCC=ddd\nFOO=bar")).Return(fxRelease(), nil)

		res, err := testExecute(e, "env set AAA=bbb CCC=ddd -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Setting AAA, CCC... OK",
			"Release: release1",
		})
	})
}

func TestEnvSetReplace(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		opts := structs.ReleaseListOptions{Limit: options.Int(1)}
		i.On("ReleaseList", "app1", opts).Return(structs.Releases{*fxRelease()}, nil)
		i.On("ReleaseGet", "app1", "release1").Return(fxRelease(), nil)
		ropts := structs.ReleaseCreateOptions{Env: options.String("AAA=bbb\nCCC=ddd")}
		i.On("ReleaseCreate", "app1", ropts).Return(fxRelease(), nil)

		res, err := testExecute(e, "env set AAA=bbb CCC=ddd -a app1 --replace", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Setting AAA, CCC... OK",
			"Release: release1",
		})
	})
}

func TestEnvSetReplaceError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		opts := structs.ReleaseListOptions{Limit: options.Int(1)}
		i.On("ReleaseList", "app1", opts).Return(structs.Releases{*fxRelease()}, nil)
		i.On("ReleaseGet", "app1", "release1").Return(fxRelease(), nil)
		ropts := structs.ReleaseCreateOptions{Env: options.String("AAA=bbb\nCCC=ddd")}
		i.On("ReleaseCreate", "app1", ropts).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "env set AAA=bbb CCC=ddd -a app1 --replace", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Setting AAA, CCC... "})
	})
}

func TestEnvUnset(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		opts := structs.ReleaseListOptions{Limit: options.Int(1)}
		i.On("ReleaseList", "app1", opts).Return(structs.Releases{*fxRelease()}, nil)
		i.On("ReleaseGet", "app1", "release1").Return(fxRelease(), nil)
		ropts := structs.ReleaseCreateOptions{Env: options.String("BAZ=quux")}
		i.On("ReleaseCreate", "app1", ropts).Return(fxRelease(), nil)

		res, err := testExecute(e, "env unset FOO -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Unsetting FOO... OK",
			"Release: release1",
		})
	})
}

func TestEnvUnsetError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		opts := structs.ReleaseListOptions{Limit: options.Int(1)}
		i.On("ReleaseList", "app1", opts).Return(structs.Releases{*fxRelease()}, nil)
		i.On("ReleaseGet", "app1", "release1").Return(fxRelease(), nil)
		ropts := structs.ReleaseCreateOptions{Env: options.String("BAZ=quux")}
		i.On("ReleaseCreate", "app1", ropts).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "env unset FOO -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Unsetting FOO... "})
	})
}

func TestEnvUnsetClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemClassic(), nil)
		opts := structs.ReleaseListOptions{Limit: options.Int(1)}
		i.On("ReleaseList", "app1", opts).Return(structs.Releases{*fxRelease()}, nil)
		i.On("ReleaseGet", "app1", "release1").Return(fxRelease(), nil)
		i.On("EnvironmentUnset", "app1", "FOO").Return(fxRelease(), nil)

		res, err := testExecute(e, "env unset FOO -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Unsetting FOO... OK",
			"Release: release1",
		})
	})
}
