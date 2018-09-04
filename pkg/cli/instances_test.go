package cli_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInstances(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("InstanceList").Return(structs.Instances{*fxInstance(), *fxInstance()}, nil)

		res, err := testExecute(e, "instances", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"ID         STATUS  STARTED     PS  CPU     MEM     PUBLIC  PRIVATE",
			"instance1  status  2 days ago  3   42.30%  71.80%  public  private",
			"instance1  status  2 days ago  3   42.30%  71.80%  public  private",
		})
	})
}

func TestInstancesError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("InstanceList").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "instances", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestInstancesKeyroll(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("InstanceKeyroll").Return(nil)

		res, err := testExecute(e, "instances keyroll", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Rolling instance key... OK"})
	})
}

func TestInstancesKeyrollError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("InstanceKeyroll").Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "instances keyroll", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Rolling instance key... "})
	})
}

func TestInstancesSsh(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		opts := structs.InstanceShellOptions{}
		i.On("InstanceShell", "instance1", mock.Anything, opts).Return(4, nil).Run(func(args mock.Arguments) {
			data, err := ioutil.ReadAll(args.Get(1).(io.Reader))
			require.NoError(t, err)
			require.Equal(t, "in", string(data))
			args.Get(1).(io.Writer).Write([]byte("out"))
		})

		res, err := testExecute(e, "instances ssh instance1", strings.NewReader("in"))
		require.NoError(t, err)
		require.Equal(t, 4, res.Code)
		res.RequireStderr(t, []string{""})
		require.Equal(t, "out", res.Stdout)
	})
}

func TestInstancesSshError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		opts := structs.InstanceShellOptions{}
		i.On("InstanceShell", "instance1", mock.Anything, opts).Return(0, fmt.Errorf("err1"))

		res, err := testExecute(e, "instances ssh instance1", strings.NewReader("in"))
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestInstancesSshClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemClassic(), nil)
		opts := structs.InstanceShellOptions{}
		i.On("InstanceShellClassic", "instance1", mock.Anything, opts).Return(4, nil).Run(func(args mock.Arguments) {
			data, err := ioutil.ReadAll(args.Get(1).(io.Reader))
			require.NoError(t, err)
			require.Equal(t, "in", string(data))
			args.Get(1).(io.Writer).Write([]byte("out"))
		})

		res, err := testExecute(e, "instances ssh instance1", strings.NewReader("in"))
		require.NoError(t, err)
		require.Equal(t, 4, res.Code)
		res.RequireStderr(t, []string{""})
		require.Equal(t, "out", res.Stdout)
	})
}

func TestInstancesTerminate(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("InstanceTerminate", "instance1").Return(nil)

		res, err := testExecute(e, "instances terminate instance1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Terminating instance... OK"})
	})
}

func TestInstancesTerminateError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("InstanceTerminate", "instance1").Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "instances terminate instance1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Terminating instance... "})
	})
}
