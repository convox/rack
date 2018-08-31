package cli_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/require"
)

var fxProcess = structs.Process{
	Id:       "pid1",
	App:      "app1",
	Command:  "command",
	Cpu:      1.0,
	Host:     "host",
	Image:    "image",
	Instance: "instance",
	Memory:   2.0,
	Name:     "name",
	Ports:    []string{"1000", "2000"},
	Release:  "release1",
	Started:  time.Now().UTC().Add(-49 * time.Hour),
	Status:   "running",
}

var fxProcessPending = structs.Process{
	Id:       "pid1",
	App:      "app1",
	Command:  "command",
	Cpu:      1.0,
	Host:     "host",
	Image:    "image",
	Instance: "instance",
	Memory:   2.0,
	Name:     "name",
	Ports:    []string{"1000", "2000"},
	Release:  "release1",
	Started:  time.Now().UTC().Add(-49 * time.Hour),
	Status:   "pending",
}

func TestPs(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("ProcessList", "app1", structs.ProcessListOptions{}).Return(structs.Processes{fxProcess, fxProcessPending}, nil)

		res, err := testExecute(e, "ps -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"ID    SERVICE  STATUS   RELEASE   STARTED     COMMAND",
			"pid1  name     running  release1  2 days ago  command",
			"pid1  name     pending  release1  2 days ago  command",
		})
	})
}

func TestPsError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("ProcessList", "app1", structs.ProcessListOptions{}).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "ps -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestPsInfo(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("ProcessGet", "app1", "pid1").Return(&fxProcess, nil)

		res, err := testExecute(e, "ps info pid1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Id        pid1",
			"App       app1",
			"Command   command",
			"Instance  instance",
			"Release   release1",
			"Service   name",
			"Started   2 days ago",
			"Status    running",
		})
	})
}

func TestPsInfoError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("ProcessGet", "app1", "pid1").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "ps info pid1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestPsStop(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("ProcessStop", "app1", "pid1").Return(nil)

		res, err := testExecute(e, "ps stop pid1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Stopping pid1... OK"})
	})
}

func TestPsStopError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("ProcessStop", "app1", "pid1").Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "ps stop pid1 -a app1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Stopping pid1... "})
	})
}
