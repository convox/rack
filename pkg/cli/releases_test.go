package cli_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/require"
)

func TestReleases(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppGet", "app1").Return(fxApp(), nil)
		i.On("ReleaseList", "app1", structs.ReleaseListOptions{}).Return(structs.Releases{*fxRelease(), *fxRelease2()}, nil)

		res, err := testExecute(e, "releases -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"ID        STATUS  BUILD   CREATED   ",
			"release1  active  build1  2 days ago",
			"release2          build1  2 days ago",
		})
	})
}

func TestReleasesError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppGet", "app1").Return(fxApp(), nil)
		i.On("ReleaseList", "app1", structs.ReleaseListOptions{}).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "releases -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestReleasesInfo(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("ReleaseGet", "app1", "release1").Return(fxRelease(), nil)

		res, err := testExecute(e, "releases info release1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Id       release1",
			"Build    build1",
			fmt.Sprintf("Created  %s", fxRelease().Created.Format(time.RFC3339)),
			"Env      FOO=bar",
			"         BAZ=quux",
		})
	})
}

func TestReleasesInfoError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("ReleaseGet", "app1", "release1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "releases info release1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestReleasesManifest(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("ReleaseGet", "app1", "release1").Return(fxRelease(), nil)
		i.On("BuildGet", "app1", "build1").Return(fxBuild(), nil)

		res, err := testExecute(e, "releases manifest release1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"manifest1",
			"manifest2",
		})
	})
}

func TestReleasesManifestError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("ReleaseGet", "app1", "release1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "releases manifest release1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestReleasesPromote(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("ReleasePromote", "app1", "release1").Return(nil)

		res, err := testExecute(e, "releases promote release1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Promoting release1... OK"})
	})
}

func TestReleasesPromoteError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("ReleasePromote", "app1", "release1").Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "releases promote release1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Promoting release1... "})
	})
}

func TestReleasesRollback(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("ReleaseGet", "app1", "release2").Return(fxRelease2(), nil)
		i.On("ReleaseCreate", "app1", structs.ReleaseCreateOptions{Build: options.String(fxRelease2().Build), Env: options.String(fxRelease2().Env)}).Return(fxRelease3(), nil)
		i.On("ReleasePromote", "app1", "release3").Return(nil)

		res, err := testExecute(e, "releases rollback release2 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Rolling back to release2... OK, release3",
			"Promoting release3... OK",
		})
	})
}

func TestReleasesRollbackErrorCreate(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("ReleaseGet", "app1", "release2").Return(fxRelease2(), nil)
		i.On("ReleaseCreate", "app1", structs.ReleaseCreateOptions{Build: options.String(fxRelease2().Build), Env: options.String(fxRelease2().Env)}).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "releases rollback release2 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Rolling back to release2... "})
	})
}

func TestReleasesRollbackErrorPromote(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("ReleaseGet", "app1", "release2").Return(fxRelease2(), nil)
		i.On("ReleaseCreate", "app1", structs.ReleaseCreateOptions{Build: options.String(fxRelease2().Build), Env: options.String(fxRelease2().Env)}).Return(fxRelease3(), nil)
		i.On("ReleasePromote", "app1", "release3").Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "releases rollback release2 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{
			"Rolling back to release2... OK, release3",
			"Promoting release3... ",
		})
	})
}
