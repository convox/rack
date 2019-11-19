package cli_test

import (
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

func TestTest(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ObjectStore", "app1", mock.AnythingOfType("string"), mock.Anything, structs.ObjectStoreOptions{}).Return(&fxObject, nil).Run(func(args mock.Arguments) {
			require.Regexp(t, `tmp/[0-9a-f]{30}\.tgz`, args.Get(1).(string))
		})
		i.On("BuildCreate", "app1", "object://test", structs.BuildCreateOptions{Development: options.Bool(true), Description: options.String("foo")}).Return(fxBuild(), nil)
		i.On("BuildLogs", "app1", "build1", structs.LogsOptions{}).Return(testLogs(fxLogs()), nil)
		i.On("BuildGet", "app1", "build1").Return(fxBuildRunning(), nil).Once()
		i.On("BuildGet", "app1", "build4").Return(fxBuild(), nil)
		i.On("ReleaseGet", "app1", "release1").Return(fxRelease(), nil)
		i.On("ProcessRun", "app1", "web", structs.ProcessRunOptions{Command: options.String("sleep 7200"), Release: options.String("release1")}).Return(fxProcess(), nil)
		i.On("ProcessGet", "app1", "pid1").Return(fxProcessPending(), nil).Twice()
		i.On("ProcessGet", "app1", "pid1").Return(fxProcess(), nil)
		opts := structs.ProcessExecOptions{Entrypoint: options.Bool(true), Tty: options.Bool(false)}
		i.On("ProcessExec", "app1", "pid1", "make test", mock.Anything, opts).Return(0, nil).Run(func(args mock.Arguments) {
			data, err := ioutil.ReadAll(args.Get(3).(io.Reader))
			require.NoError(t, err)
			require.Equal(t, "in", string(data))
			args.Get(3).(io.Writer).Write([]byte("out"))
		})
		i.On("ProcessStop", "app1", "pid1").Return(nil)

		res, err := testExecute(e, "test ./testdata/httpd -a app1 -d foo -t 7200", strings.NewReader("in"))
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Packaging source... OK",
			"Uploading source... OK",
			"Starting build... OK",
			"log1",
			"log2",
			"Running make test on web",
			"out",
		})
	})
}

func TestTestFail(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		i.On("ObjectStore", "app1", mock.AnythingOfType("string"), mock.Anything, structs.ObjectStoreOptions{}).Return(&fxObject, nil).Run(func(args mock.Arguments) {
			require.Regexp(t, `tmp/[0-9a-f]{30}\.tgz`, args.Get(1).(string))
		})
		i.On("BuildCreate", "app1", "object://test", structs.BuildCreateOptions{Development: options.Bool(true), Description: options.String("foo")}).Return(fxBuild(), nil)
		i.On("BuildLogs", "app1", "build1", structs.LogsOptions{}).Return(testLogs(fxLogs()), nil)
		i.On("BuildGet", "app1", "build1").Return(fxBuildRunning(), nil).Once()
		i.On("BuildGet", "app1", "build4").Return(fxBuild(), nil)
		i.On("ReleaseGet", "app1", "release1").Return(fxRelease(), nil)
		i.On("ProcessRun", "app1", "web", structs.ProcessRunOptions{Command: options.String("sleep 7200"), Release: options.String("release1")}).Return(fxProcess(), nil)
		i.On("ProcessGet", "app1", "pid1").Return(fxProcessPending(), nil).Twice()
		i.On("ProcessGet", "app1", "pid1").Return(fxProcess(), nil)
		opts := structs.ProcessExecOptions{Entrypoint: options.Bool(true), Tty: options.Bool(false)}
		i.On("ProcessExec", "app1", "pid1", "make test", mock.Anything, opts).Return(4, nil).Run(func(args mock.Arguments) {
			data, err := ioutil.ReadAll(args.Get(3).(io.Reader))
			require.NoError(t, err)
			require.Equal(t, "in", string(data))
			args.Get(3).(io.Writer).Write([]byte("out"))
		})
		i.On("ProcessStop", "app1", "pid1").Return(nil)

		res, err := testExecute(e, "test ./testdata/httpd -a app1 -d foo -t 7200", strings.NewReader("in"))
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: exit 4"})
		res.RequireStdout(t, []string{
			"Packaging source... OK",
			"Uploading source... OK",
			"Starting build... OK",
			"log1",
			"log2",
			"Running make test on web",
			"out",
		})
	})
}
