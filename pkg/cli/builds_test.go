package cli_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var fxBuild = structs.Build{
	Id:      "build1",
	Release: "release1",
	Status:  "complete",
}

var fxBuildCreated = structs.Build{
	Id:     "build1",
	Status: "running",
}

var fxBuildFailed = structs.Build{
	Id:     "build1",
	Status: "failed",
}

var fxBuildRunning = structs.Build{
	Id:     "build1",
	Status: "running",
}

func TestBuild(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystem, nil)
		i.On("ObjectStore", "app1", mock.AnythingOfType("string"), mock.Anything, structs.ObjectStoreOptions{}).Return(&fxObject, nil).Run(func(args mock.Arguments) {
			require.Regexp(t, `tmp/[0-9a-f]{30}\.tgz`, args.Get(1).(string))
		})
		i.On("BuildCreate", "app1", "object://test", structs.BuildCreateOptions{}).Return(&fxBuild, nil)
		i.On("BuildLogs", "app1", "build1", structs.LogsOptions{}).Return(testLogs(fxLogs), nil)
		i.On("BuildGet", "app1", "build1").Return(&fxBuildRunning, nil).Twice()
		i.On("BuildGet", "app1", "build1").Return(&fxBuild, nil)

		res, err := testExecute(e, "build ./testdata/httpd -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Equal(t, "", res.Stderr)
		res.RequireStdout(t, []string{
			"Packaging source... OK",
			"Uploading source... OK",
			"Starting build... OK",
			"log1",
			"log2",
			"Build:   build1",
			"Release: release1",
		})
	})
}

func TestBuildError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystem, nil)
		i.On("ObjectStore", "app1", mock.AnythingOfType("string"), mock.Anything, structs.ObjectStoreOptions{}).Return(&fxObject, nil).Run(func(args mock.Arguments) {
			require.Regexp(t, `tmp/[0-9a-f]{30}\.tgz`, args.Get(1).(string))
		})
		i.On("BuildCreate", "app1", "object://test", structs.BuildCreateOptions{}).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "build ./testdata/httpd -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStdout(t, []string{
			"Packaging source... OK",
			"Uploading source... OK",
			"Starting build... ",
		})
		res.RequireStderr(t, []string{"ERROR: err1"})
	})
}

func TestBuildClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystemClassic, nil)
		i.On("BuildCreateUpload", "app1", mock.Anything, structs.BuildCreateOptions{}).Return(&fxBuild, nil)
		i.On("BuildLogs", "app1", "build1", structs.LogsOptions{}).Return(testLogs(fxLogs), nil)
		i.On("BuildGet", "app1", "build1").Return(&fxBuildRunning, nil).Twice()
		i.On("BuildGet", "app1", "build1").Return(&fxBuild, nil)

		res, err := testExecute(e, "build ./testdata/httpd -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Equal(t, "", res.Stderr)
		res.RequireStdout(t, []string{
			"Packaging source... OK",
			"Starting build... OK",
			"log1",
			"log2",
			"Build:   build1",
			"Release: release1",
		})
	})
}
