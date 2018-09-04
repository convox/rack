package cli_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ProcessRun", "app1", "web", structs.ProcessRunOptions{Command: options.String("sleep 7200")}).Return(fxProcess(), nil)
		i.On("ProcessGet", "app1", "pid1").Return(fxProcessPending(), nil).Twice()
		i.On("ProcessGet", "app1", "pid1").Return(fxProcess(), nil)
		opts := structs.ProcessExecOptions{Entrypoint: options.Bool(true), Tty: options.Bool(false)}
		i.On("ProcessExec", "app1", "pid1", "bash", mock.Anything, opts).Return(4, nil).Run(func(args mock.Arguments) {
			data, err := ioutil.ReadAll(args.Get(3).(io.Reader))
			require.NoError(t, err)
			require.Equal(t, "in", string(data))
			args.Get(3).(io.Writer).Write([]byte("out"))
		})
		i.On("ProcessStop", "app1", "pid1").Return(nil)

		res, err := testExecute(e, "run web bash -a app1 -t 7200", strings.NewReader("in"))
		require.NoError(t, err)
		require.Equal(t, 4, res.Code)
		res.RequireStderr(t, []string{""})
		require.Equal(t, "out", res.Stdout)
	})
}

func TestRunError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ProcessRun", "app1", "web", structs.ProcessRunOptions{Command: options.String("sleep 7200")}).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "run web bash -a app1 -t 7200", strings.NewReader("in"))
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestRunClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemClassic(), nil)
		i.On("ProcessRunAttached", "app1", "web", mock.Anything, 7200, structs.ProcessRunOptions{Command: options.String("bash")}).Return(4, nil).Run(func(args mock.Arguments) {
			data, err := ioutil.ReadAll(args.Get(2).(io.Reader))
			require.NoError(t, err)
			require.Equal(t, "in", string(data))
			args.Get(2).(io.Writer).Write([]byte("out"))
		})

		res, err := testExecute(e, "run web bash -a app1 -t 7200", strings.NewReader("in"))
		require.NoError(t, err)
		require.Equal(t, 4, res.Code)
		res.RequireStderr(t, []string{""})
		require.Equal(t, "out", res.Stdout)
	})
}

func TestRunClassicDetached(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemClassic(), nil)
		i.On("ProcessRunDetached", "app1", "web", structs.ProcessRunOptions{Command: options.String("bash")}).Return("pid1", nil)

		res, err := testExecute(e, "run web bash -a app1 -d", strings.NewReader("in"))
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Running detached process... OK, pid1"})
	})
}

func TestRunDetached(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ProcessRun", "app1", "web", structs.ProcessRunOptions{Command: options.String("bash")}).Return(fxProcess(), nil)

		res, err := testExecute(e, "run web bash -a app1 -d", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Running detached process... OK, pid1"})
	})
}
