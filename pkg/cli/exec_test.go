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

func TestExec(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.ProcessExecOptions{Tty: options.Bool(false)}
		i.On("ProcessExec", "app1", "0123456789", "bash", mock.Anything, opts).Return(4, nil).Run(func(args mock.Arguments) {
			data, err := ioutil.ReadAll(args.Get(3).(io.Reader))
			require.NoError(t, err)
			require.Equal(t, "in", string(data))
			args.Get(3).(io.Writer).Write([]byte("out"))
		})

		res, err := testExecute(e, "exec 0123456789 bash -a app1", strings.NewReader("in"))
		require.NoError(t, err)
		require.Equal(t, 4, res.Code)
		res.RequireStderr(t, []string{""})
		require.Equal(t, "out", res.Stdout)
	})
}

func TestExecError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.ProcessExecOptions{Tty: options.Bool(false)}
		i.On("ProcessExec", "app1", "0123456789", "bash", mock.Anything, opts).Return(0, fmt.Errorf("err1"))

		res, err := testExecute(e, "exec 0123456789 bash -a app1", strings.NewReader("in"))
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}
