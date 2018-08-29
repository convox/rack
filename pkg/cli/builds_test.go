package cli_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var fxBuild = structs.Build{
	Id:          "build1",
	Description: "desc",
	Ended:       fxStarted.Add(2 * time.Minute),
	Release:     "release1",
	Started:     fxStarted,
	Status:      "complete",
}

var fxBuildCreated = structs.Build{
	Id:     "build2",
	Status: "running",
}

var fxBuildFailed = structs.Build{
	Id:      "build3",
	Started: fxStarted,
	Status:  "failed",
}

var fxBuildRunning = structs.Build{
	Id:      "build4",
	Started: fxStarted,
	Status:  "running",
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
		i.On("BuildGet", "app1", "build4").Return(&fxBuild, nil)

		res, err := testExecute(e, "build ./testdata/httpd -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
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
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{
			"Packaging source... OK",
			"Uploading source... OK",
			"Starting build... ",
		})
	})
}

func TestBuildClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystemClassic, nil)
		i.On("BuildCreateUpload", "app1", mock.Anything, structs.BuildCreateOptions{}).Return(&fxBuild, nil)
		i.On("BuildLogs", "app1", "build1", structs.LogsOptions{}).Return(testLogs(fxLogs), nil)
		i.On("BuildGet", "app1", "build1").Return(&fxBuildRunning, nil).Twice()
		i.On("BuildGet", "app1", "build4").Return(&fxBuild, nil)

		res, err := testExecute(e, "build ./testdata/httpd -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
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

func TestBuilds(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		b1 := structs.Builds{
			fxBuild,
			fxBuildRunning,
			fxBuildFailed,
		}
		i.On("BuildList", "app1", structs.BuildListOptions{}).Return(b1, nil)

		res, err := testExecute(e, "builds -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"ID      STATUS    RELEASE   STARTED     ELAPSED  DESCRIPTION",
			"build1  complete  release1  2 days ago  2m0s     desc       ",
			"build4  running             2 days ago                      ",
			"build3  failed              2 days ago                      ",
		})
	})
}

func TestBuildsError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("BuildList", "app1", structs.BuildListOptions{}).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "builds -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestBuildsExport(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		data, err := ioutil.ReadFile("testdata/build.tgz")
		require.NoError(t, err)
		i.On("BuildExport", "app1", "build1", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			args.Get(2).(io.Writer).Write(data)
		})
		tmpd, err := ioutil.TempDir("", "")
		require.NoError(t, err)
		tmpf := filepath.Join(tmpd, "export.tgz")

		res, err := testExecute(e, fmt.Sprintf("builds export build1 -a app1 -f %s", tmpf), nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Exporting build... OK"})
		tdata, err := ioutil.ReadFile(tmpf)
		require.NoError(t, err)
		require.Equal(t, data, tdata)
	})
}

func TestBuildsExportStdout(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		data, err := ioutil.ReadFile("testdata/build.tgz")
		require.NoError(t, err)
		i.On("BuildExport", "app1", "build1", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			args.Get(2).(io.Writer).Write(data)
		})

		res, err := testExecute(e, "builds export build1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{"Exporting build... OK"})
		require.Equal(t, data, []byte(res.Stdout))
	})
}

func TestBuildsExportError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("BuildExport", "app1", "build1", mock.Anything).Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "builds export build1 -a app1 -f /dev/null", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Exporting build... "})
	})
}

func TestBuildsImport(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		data, err := ioutil.ReadFile("testdata/build.tgz")
		require.NoError(t, err)
		i.On("SystemGet").Return(&fxSystem, nil)
		i.On("BuildImport", "app1", mock.Anything).Return(&fxBuild, nil).Run(func(args mock.Arguments) {
			rdata, err := ioutil.ReadAll(args.Get(1).(io.Reader))
			require.NoError(t, err)
			require.Equal(t, data, rdata)
		})

		res, err := testExecute(e, "builds import -a app1 -f testdata/build.tgz", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Importing build... OK, release1"})
	})
}

func TestBuildsImportError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystem, nil)
		i.On("BuildImport", "app1", mock.Anything).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "builds import -a app1 -f testdata/build.tgz", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Importing build... "})
	})
}

func TestBuildsImportClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		data, err := ioutil.ReadFile("testdata/build.tgz")
		require.NoError(t, err)
		i.On("SystemGet").Return(&fxSystemClassic, nil)
		i.On("BuildImportMultipart", "app1", mock.Anything).Return(&fxBuild, nil).Run(func(args mock.Arguments) {
			rdata, err := ioutil.ReadAll(args.Get(1).(io.Reader))
			require.NoError(t, err)
			require.Equal(t, data, rdata)
		})

		res, err := testExecute(e, "builds import -a app1 -f testdata/build.tgz", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Importing build... OK, release1"})
	})
}

func TestBuildsInfo(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("BuildGet", "app1", "build1").Return(&fxBuild, nil)

		res, err := testExecute(e, "builds info build1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Id           build1",
			"Status       complete",
			"Release      release1",
			"Description  desc",
			"Started      2 days ago",
			"Elapsed      2m0s",
		})
	})
}

func TestBuildsInfoError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("BuildGet", "app1", "build1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "builds info build1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestBuildsLogs(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.LogsOptions{}
		i.On("BuildLogs", "app1", "build1", opts).Return(testLogs(fxLogs), nil)

		res, err := testExecute(e, "builds logs build1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			fxLogs[0],
			fxLogs[1],
		})
	})
}

func TestBuildsLogsError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.LogsOptions{}
		i.On("BuildLogs", "app1", "build1", opts).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "builds logs build1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}
