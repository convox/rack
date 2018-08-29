package cli_test

import (
	"fmt"
	"io/ioutil"
	"strings"
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

func TestAppsCreate(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.AppCreateOptions{}
		i.On("AppCreate", "app1", opts).Return(&fxApp, nil)

		res, err := testExecute(e, "apps create app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 1, res.StdoutLines())
		require.Equal(t, "Creating app1... OK", res.StdoutLine(0))
	})
}

func TestAppsCreateError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.AppCreateOptions{}
		i.On("AppCreate", "app1", opts).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "apps create app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		require.Equal(t, "ERROR: err1\n", res.Stderr)
		require.Equal(t, 1, res.StdoutLines())
		require.Equal(t, "Creating app1... ", res.StdoutLine(0))
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
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 1, res.StdoutLines())
		require.Equal(t, "Creating app1... OK", res.StdoutLine(0))
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
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 1, res.StdoutLines())
		require.Equal(t, "Creating app1... OK", res.StdoutLine(0))
	})
}

func TestAppsDelete(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppDelete", "app1").Return(nil)

		res, err := testExecute(e, "apps delete app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 1, res.StdoutLines())
		require.Equal(t, "Deleting app1... OK", res.StdoutLine(0))
	})
}

func TestAppsDeleteError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppDelete", "app1").Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "apps delete app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		require.Equal(t, "ERROR: err1\n", res.Stderr)
		require.Equal(t, 1, res.StdoutLines())
		require.Equal(t, "Deleting app1... ", res.StdoutLine(0))
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
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 1, res.StdoutLines())
		require.Equal(t, "Deleting app1... OK", res.StdoutLine(0))
	})
}

func TestAppsInfo(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppGet", "app1").Return(&fxApp, nil)

		res, err := testExecute(e, "apps info app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 4, res.StdoutLines())
		require.Equal(t, "Name     app1", res.StdoutLine(0))
		require.Equal(t, "Status   running", res.StdoutLine(1))
		require.Equal(t, "Gen      2", res.StdoutLine(2))
		require.Equal(t, "Release  release1", res.StdoutLine(3))

		res, err = testExecute(e, "apps info -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 4, res.StdoutLines())
		require.Equal(t, "Name     app1", res.StdoutLine(0))
		require.Equal(t, "Status   running", res.StdoutLine(1))
		require.Equal(t, "Gen      2", res.StdoutLine(2))
		require.Equal(t, "Release  release1", res.StdoutLine(3))
	})
}

func TestAppsInfoError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("AppGet", "app1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "apps info app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		require.Equal(t, "ERROR: err1\n", res.Stderr)
		require.Equal(t, 0, res.StdoutLines())
	})

}

func TestAppsParams(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystem, nil)
		i.On("AppGet", "app1").Return(&fxApp, nil)

		res, err := testExecute(e, "apps params app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 2, res.StdoutLines())
		require.Equal(t, "ParamFoo    value1", res.StdoutLine(0))
		require.Equal(t, "ParamOther  value2", res.StdoutLine(1))

		res, err = testExecute(e, "apps params -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 2, res.StdoutLines())
		require.Equal(t, "ParamFoo    value1", res.StdoutLine(0))
		require.Equal(t, "ParamOther  value2", res.StdoutLine(1))
	})
}

func TestAppsParamsError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystem, nil)
		i.On("AppGet", "app1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "apps params app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		require.Equal(t, "ERROR: err1\n", res.Stderr)
		require.Equal(t, 0, res.StdoutLines())
	})
}

func TestAppsParamsClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystemClassic, nil)
		i.On("AppParametersGet", "app1").Return(fxAppParameters, nil)

		res, err := testExecute(e, "apps params app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 2, res.StdoutLines())
		require.Equal(t, "ParamFoo    value1", res.StdoutLine(0))
		require.Equal(t, "ParamOther  value2", res.StdoutLine(1))
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
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 1, res.StdoutLines())
		require.Equal(t, "Updating parameters... OK", res.StdoutLine(0))
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
		require.Equal(t, "ERROR: err1\n", res.Stderr)
		require.Equal(t, 1, res.StdoutLines())
		require.Equal(t, "Updating parameters... ", res.StdoutLine(0))
	})
}

func TestAppsParamsSetClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(&fxSystemClassic, nil)
		i.On("AppParametersSet", "app1", map[string]string{"Foo": "bar", "Baz": "qux"}).Return(nil)

		res, err := testExecute(e, "apps params set Foo=bar Baz=qux -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 1, res.StdoutLines())
		require.Equal(t, "Updating parameters... OK", res.StdoutLine(0))
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
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 1, res.StdoutLines())
		require.Equal(t, "Sleeping app1... OK", res.StdoutLine(0))

		res, err = testExecute(e, "apps sleep -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 1, res.StdoutLines())
		require.Equal(t, "Sleeping app1... OK", res.StdoutLine(0))
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
		require.Equal(t, "ERROR: err1\n", res.Stderr)
		require.Equal(t, 1, res.StdoutLines())
		require.Equal(t, "Sleeping app1... ", res.StdoutLine(0))
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
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 1, res.StdoutLines())
		require.Equal(t, "Waking app1... OK", res.StdoutLine(0))

		res, err = testExecute(e, "apps wake -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 1, res.StdoutLines())
		require.Equal(t, "Waking app1... OK", res.StdoutLine(0))
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
		require.Equal(t, "ERROR: err1\n", res.Stderr)
		require.Equal(t, 1, res.StdoutLines())
		require.Equal(t, "Waking app1... ", res.StdoutLine(0))
	})
}

func TestAppsWait(t *testing.T) {
	testClientWait(t, 0*time.Second, func(e *cli.Engine, i *mocksdk.Interface) {
		logs := []string{
			"TIME system/aws/foo log1",
			"TIME system/aws/foo log2",
		}
		opts := structs.LogsOptions{
			Prefix: options.Bool(true),
			Since:  options.Duration(0),
		}
		i.On("AppGet", "app1").Return(&structs.App{Status: "creating"}, nil).Twice()
		i.On("AppGet", "app1").Return(&fxApp, nil)

		i.On("AppLogs", "app1", opts).Return(ioutil.NopCloser(strings.NewReader(strings.Join(logs, "\n"))), nil).Once()

		res, err := testExecute(e, "apps wait app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 4, res.StdoutLines())
		require.Equal(t, "Waiting for app... ", res.StdoutLine(0))
		require.Equal(t, logs[0], res.StdoutLine(1))
		require.Equal(t, logs[1], res.StdoutLine(2))
		require.Equal(t, "OK", res.StdoutLine(3))

		i.On("AppLogs", "app1", opts).Return(ioutil.NopCloser(strings.NewReader(strings.Join(logs, "\n"))), nil).Once()

		res, err = testExecute(e, "apps wait -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		require.Equal(t, "", res.Stderr)
		require.Equal(t, 4, res.StdoutLines())
		require.Equal(t, "Waiting for app... ", res.StdoutLine(0))
		require.Equal(t, logs[0], res.StdoutLine(1))
		require.Equal(t, logs[1], res.StdoutLine(2))
		require.Equal(t, "OK", res.StdoutLine(3))
	})
}

func TestAppsWaitError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.LogsOptions{
			Prefix: options.Bool(true),
			Since:  options.Duration(0),
		}
		i.On("AppGet", "app1").Return(nil, fmt.Errorf("err1"))
		i.On("AppLogs", "app1", opts).Return(nil, fmt.Errorf("err2"))

		res, err := testExecute(e, "apps wait app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		require.Equal(t, "ERROR: err1\n", res.Stderr)
		require.Equal(t, 1, res.StdoutLines())
		require.Equal(t, "Waiting for app... ", res.StdoutLine(0))
	})
}
