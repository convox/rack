package cli_test

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/convox/rack/pkg/cli"
	"github.com/convox/rack/pkg/helpers"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var fxApp = structs.App{
	Name:       "app1",
	Generation: "2",
	Parameters: fxParameters,
	Release:    "release1",
	Status:     "running",
}

var fxAppGeneration1 = structs.App{
	Name:       "app1",
	Generation: "1",
	Parameters: fxParameters,
	Release:    "release1",
	Status:     "running",
}

var fxAppUpdating = structs.App{
	Name:       "app1",
	Generation: "2",
	Parameters: fxParameters,
	Release:    "release1",
	Status:     "updating",
}

func TestApps(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		a1 := structs.Apps{
			fxApp,
			fxAppGeneration1,
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
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"APP   STATUS    GEN  RELEASE ",
			"app1  running   2    release1",
			"app1  running   1    release1",
			"app2  creating  1            ",
		})
	})
}

func TestAppsError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppList").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "apps", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestAppsCancel(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppCancel", "app1").Return(nil)

		res, err := testExecute(e, "apps cancel app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Cancelling app1... OK",
		})

		res, err = testExecute(e, "apps cancel -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Cancelling app1... OK",
		})
	})
}

func TestAppsCancelError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppCancel", "app1").Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "apps cancel app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{
			"Cancelling app1... ",
		})
	})
}

func TestAppsCreate(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.AppCreateOptions{}
		i.On("AppCreate", "app1", opts).Return(&fxApp, nil)

		res, err := testExecute(e, "apps create app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Creating app1... OK",
		})
	})
}

func TestAppsCreateError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.AppCreateOptions{}
		i.On("AppCreate", "app1", opts).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "apps create app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Creating app1... "})
	})
}

func TestAppsCreateGeneration1(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.AppCreateOptions{
			Generation: options.String("1"),
		}
		i.On("AppCreate", "app1", opts).Return(&fxApp, nil)

		res, err := testExecute(e, "apps create app1 -g 1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Creating app1... OK",
		})
	})
}

func TestAppsCreateWait(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.AppCreateOptions{}
		i.On("AppCreate", "app1", opts).Return(&fxApp, nil)
		i.On("AppGet", "app1").Return(&structs.App{Status: "creating"}, nil).Twice()
		i.On("AppGet", "app1").Return(&fxApp, nil)

		res, err := testExecute(e, "apps create app1 --wait", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Creating app1... OK",
		})
	})
}

func TestAppsDelete(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppDelete", "app1").Return(nil)

		res, err := testExecute(e, "apps delete app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Deleting app1... OK",
		})
	})
}

func TestAppsDeleteError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppDelete", "app1").Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "apps delete app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Deleting app1... "})
	})
}

func TestAppsDeleteWait(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppDelete", "app1").Return(nil)
		i.On("AppGet", "app1").Return(&structs.App{Status: "deleting"}, nil).Twice()
		i.On("AppGet", "app1").Return(nil, fmt.Errorf("no such app: app1"))

		res, err := testExecute(e, "apps delete app1 --wait", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Deleting app1... OK",
		})
	})
}

func TestAppsExport(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppGet", "app1").Return(&fxApp, nil)
		i.On("ReleaseGet", "app1", "release1").Return(&fxRelease, nil)
		bdata, err := ioutil.ReadFile("testdata/build.tgz")
		require.NoError(t, err)
		i.On("BuildExport", "app1", "build1", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			args.Get(2).(io.Writer).Write(bdata)
		})

		tmp, err := ioutil.TempDir("", "")
		require.NoError(t, err)
		defer os.RemoveAll(tmp)

		res, err := testExecute(e, fmt.Sprintf("apps export -a app1 -f %s/app.tgz", tmp), nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Exporting app app1... OK",
			"Exporting env... OK",
			"Exporting build build1... OK",
			"Packaging export... OK",
		})

		fd, err := os.Open(filepath.Join(tmp, "app.tgz"))
		require.NoError(t, err)
		defer fd.Close()

		gz, err := gzip.NewReader(fd)
		require.NoError(t, err)

		err = helpers.Unarchive(gz, tmp)
		require.NoError(t, err)

		data, err := ioutil.ReadFile(filepath.Join(tmp, "app.json"))
		require.NoError(t, err)
		require.Equal(t, "{\"generation\":\"2\",\"name\":\"app1\",\"release\":\"release1\",\"sleep\":false,\"status\":\"running\",\"parameters\":{\"ParamFoo\":\"value1\",\"ParamOther\":\"value2\"}}", string(data))

		data, err = ioutil.ReadFile(filepath.Join(tmp, "env"))
		require.NoError(t, err)
		require.Equal(t, "FOO=bar\nBAZ=quux", string(data))

		data, err = ioutil.ReadFile(filepath.Join(tmp, "build.tgz"))
		require.NoError(t, err)
		require.Equal(t, bdata, data)
	})
}

func TestAppsImport(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppCreate", "app1", structs.AppCreateOptions{Generation: options.String("2")}).Return(&fxApp, nil)
		i.On("AppGet", "app1").Return(&structs.App{Status: "creating"}, nil).Twice()
		i.On("AppGet", "app1").Return(&fxApp, nil).Twice()
		bdata, err := ioutil.ReadFile("testdata/build.tgz")
		require.NoError(t, err)
		i.On("BuildImport", "app1", mock.Anything).Return(&fxBuild, nil).Run(func(args mock.Arguments) {
			rdata, err := ioutil.ReadAll(args.Get(1).(io.Reader))
			require.NoError(t, err)
			require.Equal(t, bdata, rdata)
		})
		i.On("ReleaseCreate", "app1", structs.ReleaseCreateOptions{Env: options.String("ALPHA=one\nBRAVO=two\n")}).Return(&fxRelease, nil)
		i.On("ReleasePromote", "app1", "release1").Return(nil)
		i.On("AppGet", "app1").Return(&structs.App{Status: "creating"}, nil).Twice()
		i.On("AppGet", "app1").Return(&fxApp, nil).Twice()
		i.On("AppUpdate", "app1", structs.AppUpdateOptions{Parameters: map[string]string{"Foo": "bar", "Baz": "qux"}}).Return(nil)
		i.On("AppGet", "app1").Return(&structs.App{Status: "creating"}, nil).Twice()
		i.On("AppGet", "app1").Return(&fxApp, nil).Twice()

		res, err := testExecute(e, "apps import -a app1 -f testdata/app.tgz", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Creating app app1... OK",
			"Importing build... OK, release1",
			"Importing env... OK, release1",
			"Promoting release1... OK",
			"Updating parameters... OK",
		})
	})
}

func TestAppsImportNoBuild(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppCreate", "app1", structs.AppCreateOptions{Generation: options.String("2")}).Return(&fxApp, nil)
		i.On("AppGet", "app1").Return(&structs.App{Status: "creating"}, nil).Twice()
		i.On("AppGet", "app1").Return(&fxApp, nil).Twice()
		i.On("AppUpdate", "app1", structs.AppUpdateOptions{Parameters: map[string]string{"Foo": "bar", "Baz": "qux"}}).Return(nil)
		i.On("AppGet", "app1").Return(&structs.App{Status: "creating"}, nil).Twice()
		i.On("AppGet", "app1").Return(&fxApp, nil).Twice()

		res, err := testExecute(e, "apps import -a app1 -f testdata/app.nobuild.tgz", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Creating app app1... OK",
			"Updating parameters... OK",
		})
	})
}

func TestAppsImportNoParams(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppCreate", "app1", structs.AppCreateOptions{Generation: options.String("2")}).Return(&fxApp, nil)
		i.On("AppGet", "app1").Return(&structs.App{Status: "creating"}, nil).Twice()
		i.On("AppGet", "app1").Return(&fxApp, nil).Twice()
		bdata, err := ioutil.ReadFile("testdata/build.tgz")
		require.NoError(t, err)
		i.On("BuildImport", "app1", mock.Anything).Return(&fxBuild, nil).Run(func(args mock.Arguments) {
			rdata, err := ioutil.ReadAll(args.Get(1).(io.Reader))
			require.NoError(t, err)
			require.Equal(t, bdata, rdata)
		})
		i.On("ReleaseCreate", "app1", structs.ReleaseCreateOptions{Env: options.String("ALPHA=one\nBRAVO=two\n")}).Return(&fxRelease, nil)
		i.On("ReleasePromote", "app1", "release1").Return(nil)
		i.On("AppGet", "app1").Return(&structs.App{Status: "creating"}, nil).Twice()
		i.On("AppGet", "app1").Return(&fxApp, nil).Twice()

		res, err := testExecute(e, "apps import -a app1 -f testdata/app.noparams.tgz", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Creating app app1... OK",
			"Importing build... OK, release1",
			"Importing env... OK, release1",
			"Promoting release1... OK",
		})
	})
}

func TestAppsImportSameParams(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppCreate", "app1", structs.AppCreateOptions{Generation: options.String("2")}).Return(&fxApp, nil)
		i.On("AppGet", "app1").Return(&fxApp, nil).Twice()
		bdata, err := ioutil.ReadFile("testdata/build.tgz")
		require.NoError(t, err)
		i.On("BuildImport", "app1", mock.Anything).Return(&fxBuild, nil).Run(func(args mock.Arguments) {
			rdata, err := ioutil.ReadAll(args.Get(1).(io.Reader))
			require.NoError(t, err)
			require.Equal(t, bdata, rdata)
		})
		i.On("ReleaseCreate", "app1", structs.ReleaseCreateOptions{Env: options.String("ALPHA=one\nBRAVO=two\n")}).Return(&fxRelease, nil)
		i.On("ReleasePromote", "app1", "release1").Return(nil)
		i.On("AppGet", "app1").Return(&fxApp, nil).Twice()
		i.On("AppUpdate", "app1", structs.AppUpdateOptions{Parameters: map[string]string{"Foo": "bar", "Baz": "qux"}}).Return(nil)
		i.On("AppGet", "app1").Return(&fxApp, nil).Twice()

		res, err := testExecute(e, "apps import -a app1 -f testdata/app.sameparams.tgz", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Creating app app1... OK",
			"Importing build... OK, release1",
			"Importing env... OK, release1",
			"Promoting release1... OK",
		})
	})
}

func TestAppsInfo(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppGet", "app1").Return(&fxApp, nil)

		res, err := testExecute(e, "apps info app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Name     app1",
			"Status   running",
			"Gen      2",
			"Release  release1",
		})

		res, err = testExecute(e, "apps info -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Name     app1",
			"Status   running",
			"Gen      2",
			"Release  release1",
		})
	})
}

func TestAppsInfoError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppGet", "app1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "apps info app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})

}

func TestAppsParams(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystem, nil)
		i.On("AppGet", "app1").Return(&fxApp, nil)

		res, err := testExecute(e, "apps params app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"ParamFoo    value1",
			"ParamOther  value2",
		})

		res, err = testExecute(e, "apps params -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"ParamFoo    value1",
			"ParamOther  value2",
		})
	})
}

func TestAppsParamsError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystem, nil)
		i.On("AppGet", "app1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "apps params app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestAppsParamsClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystemClassic, nil)
		i.On("AppParametersGet", "app1").Return(fxParameters, nil)

		res, err := testExecute(e, "apps params app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"ParamFoo    value1",
			"ParamOther  value2",
		})
	})
}

func TestAppsParamsSet(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystem, nil)
		opts := structs.AppUpdateOptions{
			Parameters: map[string]string{
				"Foo": "bar",
				"Baz": "qux",
			},
		}
		i.On("AppUpdate", "app1", opts).Return(nil)

		res, err := testExecute(e, "apps params set Foo=bar Baz=qux -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Updating parameters... OK"})
	})
}

func TestAppsParamsSetError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystem, nil)
		opts := structs.AppUpdateOptions{
			Parameters: map[string]string{
				"Foo": "bar",
				"Baz": "qux",
			},
		}
		i.On("AppUpdate", "app1", opts).Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "apps params set Foo=bar Baz=qux -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Updating parameters... "})
	})
}

func TestAppsParamsSetClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystemClassic, nil)
		i.On("AppParametersSet", "app1", map[string]string{"Foo": "bar", "Baz": "qux"}).Return(nil)

		res, err := testExecute(e, "apps params set Foo=bar Baz=qux -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Updating parameters... OK"})
	})
}

func TestAppsSleep(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystem, nil)
		opts := structs.AppUpdateOptions{
			Sleep: options.Bool(true),
		}
		i.On("AppUpdate", "app1", opts).Return(nil)

		res, err := testExecute(e, "apps sleep app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Sleeping app1... OK"})

		res, err = testExecute(e, "apps sleep -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Sleeping app1... OK"})
	})
}

func TestAppsSleepError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystem, nil)
		opts := structs.AppUpdateOptions{
			Sleep: options.Bool(true),
		}
		i.On("AppUpdate", "app1", opts).Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "apps sleep app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Sleeping app1... "})
	})
}

func TestAppsWake(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystem, nil)
		opts := structs.AppUpdateOptions{
			Sleep: options.Bool(false),
		}
		i.On("AppUpdate", "app1", opts).Return(nil)

		res, err := testExecute(e, "apps wake app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Waking app1... OK"})

		res, err = testExecute(e, "apps wake -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Waking app1... OK"})
	})
}

func TestAppsWakeError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystem, nil)
		opts := structs.AppUpdateOptions{
			Sleep: options.Bool(false),
		}
		i.On("AppUpdate", "app1", opts).Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "apps wake app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Waking app1... "})
	})
}

func TestAppsWait(t *testing.T) {
	testClientWait(t, 100*time.Millisecond, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.LogsOptions{
			Prefix: options.Bool(true),
			Since:  options.Duration(0),
		}
		i.On("AppGet", "app1").Return(&structs.App{Status: "creating"}, nil).Twice()
		i.On("AppGet", "app1").Return(&fxApp, nil)
		i.On("AppLogs", "app1", opts).Return(testLogs(fxLogsSystem), nil).Once()

		res, err := testExecute(e, "apps wait app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Waiting for app... ",
			fxLogsSystem[0],
			fxLogsSystem[1],
			"OK",
		})

		i.On("AppLogs", "app1", opts).Return(testLogs(fxLogsSystem), nil).Once()

		res, err = testExecute(e, "apps wait -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Waiting for app... ",
			fxLogsSystem[0],
			fxLogsSystem[1],
			"OK",
		})
	})
}

func TestAppsWaitError(t *testing.T) {
	testClientWait(t, 100*time.Millisecond, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.LogsOptions{
			Prefix: options.Bool(true),
			Since:  options.Duration(0),
		}
		i.On("AppGet", "app1").Return(nil, fmt.Errorf("err1"))
		i.On("AppLogs", "app1", opts).Return(nil, fmt.Errorf("err2"))

		res, err := testExecute(e, "apps wait app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Waiting for app... "})
	})
}
