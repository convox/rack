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

var fxApp = structs.App{
	Name:       "app1",
	Generation: "2",
	Parameters: fxAppParameters,
	Release:    "release1",
	Status:     "running",
}

var fxAppParameters = map[string]string{
	"ParamFoo":   "value1",
	"ParamOther": "value2",
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
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"APP   STATUS    GEN  RELEASE ",
			"app1  running   2    release1",
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
		i.On("AppParametersGet", "app1").Return(fxAppParameters, nil)

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
