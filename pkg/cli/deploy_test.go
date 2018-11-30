package cli_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDeploy(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ObjectStore", "app1", mock.AnythingOfType("string"), mock.Anything, structs.ObjectStoreOptions{}).Return(&fxObject, nil).Run(func(args mock.Arguments) {
			require.Regexp(t, `tmp/[0-9a-f]{30}\.tgz`, args.Get(1).(string))
		})
		i.On("BuildCreate", "app1", "object://test", structs.BuildCreateOptions{}).Return(fxBuild(), nil)
		i.On("BuildLogs", "app1", "build1", structs.LogsOptions{}).Return(testLogs(fxLogs()), nil)
		i.On("BuildGet", "app1", "build1").Return(fxBuildRunning(), nil).Once()
		i.On("BuildGet", "app1", "build4").Return(fxBuild(), nil)
		i.On("AppGet", "app1").Return(fxApp(), nil)
		i.On("ReleaseGet", "app1", "release1").Return(fxRelease(), nil).Once()
		i.On("ReleasePromote", "app1", "release1", structs.ReleasePromoteOptions{}).Return(nil)
		i.On("AppLogs", "app1", structs.LogsOptions{Prefix: options.Bool(true), Since: options.Duration(1)}).Return(testLogs(fxLogsSystem()), nil).Once()

		res, err := testExecute(e, "deploy ./testdata/httpd -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Packaging source... OK",
			"Uploading source... OK",
			"Starting build... OK",
			"log1",
			"log2",
			"Running hooks: before-promote",
			"Promoting release1... ",
			"TIME system/service/component log1",
			"TIME system/service/component log2",
			"OK",
			"Running hooks: after-promote",
		})
	})
}

func TestDeployError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ObjectStore", "app1", mock.AnythingOfType("string"), mock.Anything, structs.ObjectStoreOptions{}).Return(&fxObject, nil).Run(func(args mock.Arguments) {
			require.Regexp(t, `tmp/[0-9a-f]{30}\.tgz`, args.Get(1).(string))
		})
		i.On("BuildCreate", "app1", "object://test", structs.BuildCreateOptions{}).Return(fxBuild(), nil)
		i.On("BuildLogs", "app1", "build1", structs.LogsOptions{}).Return(testLogs(fxLogs()), nil)
		i.On("BuildGet", "app1", "build1").Return(fxBuildRunning(), nil).Once()
		i.On("BuildGet", "app1", "build4").Return(fxBuild(), nil)
		i.On("AppGet", "app1").Return(fxApp(), nil)
		i.On("ReleaseGet", "app1", "release1").Return(fxRelease(), nil).Once()
		i.On("ReleasePromote", "app1", "release1", structs.ReleasePromoteOptions{}).Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "deploy ./testdata/httpd -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{
			"Packaging source... OK",
			"Uploading source... OK",
			"Starting build... OK",
			"log1",
			"log2",
			"Running hooks: before-promote",
			"Promoting release1... ",
		})
	})
}
