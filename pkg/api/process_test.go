package api_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/mock"
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
	Started:  time.Now().UTC(),
	Status:   "status",
}

func TestProcessExec(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		a1 := fxApp
		p.On("AppGet", "app1").Return(&a1, nil)
		opts := structs.ProcessExecOptions{
			Entrypoint: options.Bool(true),
			Height:     options.Int(1),
			Tty:        options.Bool(true),
			Width:      options.Int(2),
		}
		ro := stdsdk.RequestOptions{
			Body: strings.NewReader("in"),
			Headers: stdsdk.Headers{
				"Command":    "command",
				"Entrypoint": "true",
				"Height":     "1",
				"Width":      "2",
			},
		}
		p.On("ProcessExec", "app1", "pid1", "command", mock.Anything, opts).Return(1, nil).Run(func(args mock.Arguments) {
			rw := args.Get(3).(io.ReadWriter)
			rw.Write([]byte("out"))
			data, err := ioutil.ReadAll(rw)
			require.NoError(t, err)
			require.Equal(t, "in", string(data))
		})
		r, err := c.Websocket("/apps/app1/processes/pid1/exec", ro)
		data, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, "outF1E49A85-0AD7-4AEF-A618-C249C6E6568D:1\n", string(data))
	})
}

func TestProcessExecError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		a1 := fxApp
		p.On("AppGet", "app1").Return(&a1, nil)
		opts := structs.ProcessExecOptions{
			Entrypoint: options.Bool(true),
			Height:     options.Int(1),
			Tty:        options.Bool(false),
			Width:      options.Int(2),
		}
		ro := stdsdk.RequestOptions{
			Body: strings.NewReader("in"),
			Headers: stdsdk.Headers{
				"Command":    "command",
				"Entrypoint": "true",
				"Height":     "1",
				"Tty":        "false",
				"Width":      "2",
			},
		}
		p.On("ProcessExec", "app1", "pid1", "command", mock.Anything, opts).Return(0, fmt.Errorf("err1"))
		r, err := c.Websocket("/apps/app1/processes/pid1/exec", ro)
		require.NoError(t, err)
		d, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, []byte("ERROR: err1\n"), d)
	})
}

func TestProcessExecValidate(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("AppGet", "app1").Return(nil, fmt.Errorf("no such app: app1"))
		r, err := c.Websocket("/apps/app1/processes/pid1/exec", stdsdk.RequestOptions{})
		data, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, "ERROR: no such app: app1\n", string(data))
	})
}

func TestProcessGet(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p1 := fxProcess
		p2 := structs.Process{}
		p.On("ProcessGet", "app1", "pid1").Return(&p1, nil)
		err := c.Get("/apps/app1/processes/pid1", stdsdk.RequestOptions{}, &p2)
		require.NoError(t, err)
		require.Equal(t, p1, p2)
	})
}

func TestProcessGetError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var p1 *structs.Process
		p.On("ProcessGet", "app1", "pid1").Return(nil, fmt.Errorf("err1"))
		err := c.Get("/apps/app1/processes/pid1", stdsdk.RequestOptions{}, p1)
		require.Nil(t, p1)
		require.EqualError(t, err, "err1")
	})
}

func TestProcessList(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p1 := structs.Processes{fxProcess, fxProcess}
		p2 := structs.Processes{}
		opts := structs.ProcessListOptions{
			Service: options.String("service"),
		}
		ro := stdsdk.RequestOptions{
			Query: stdsdk.Query{
				"service": "service",
			},
		}
		p.On("ProcessList", "app1", opts).Return(p1, nil)
		err := c.Get("/apps/app1/processes", ro, &p2)
		require.NoError(t, err)
		require.Equal(t, p1, p2)
	})
}

func TestProcessListError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var p1 structs.Processes
		p.On("ProcessList", "app1", structs.ProcessListOptions{}).Return(nil, fmt.Errorf("err1"))
		err := c.Get("/apps/app1/processes", stdsdk.RequestOptions{}, &p1)
		require.EqualError(t, err, "err1")
		require.Nil(t, p1)
	})
}

func TestProcessRun(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		a1 := fxApp
		p.On("AppGet", "app1").Return(&a1, nil)
		p1 := fxProcess
		p2 := structs.Process{}
		opts := structs.ProcessRunOptions{
			Command:     options.String("command"),
			Environment: map[string]string{"k1": "v1", "k2": "v2"},
			Height:      options.Int(1),
			Memory:      options.Int(2),
			Release:     options.String("release"),
			Width:       options.Int(3),
		}
		ro := stdsdk.RequestOptions{
			Body: strings.NewReader("in"),
			Headers: stdsdk.Headers{
				"Command":     "command",
				"Environment": "k1=v1&k2=v2",
				"Height":      "1",
				"Memory":      "2",
				"Release":     "release",
				"Width":       "3",
			},
		}
		p.On("ProcessRun", "app1", "service1", opts).Return(&p1, nil)
		err := c.Post("/apps/app1/services/service1/processes", ro, &p2)
		require.NoError(t, err)
		require.Equal(t, p1, p2)
	})
}

func TestProcessRunError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		a1 := fxApp
		p.On("AppGet", "app1").Return(&a1, nil)
		var p1 *structs.Process
		opts := structs.ProcessRunOptions{
			Command:     options.String("command"),
			Environment: map[string]string{"k1": "v1", "k2": "v2"},
			Height:      options.Int(1),
			Memory:      options.Int(2),
			Release:     options.String("release"),
			Width:       options.Int(3),
		}
		ro := stdsdk.RequestOptions{
			Body: strings.NewReader("in"),
			Headers: stdsdk.Headers{
				"Command":     "command",
				"Environment": "k1=v1&k2=v2",
				"Height":      "1",
				"Memory":      "2",
				"Release":     "release",
				"Width":       "3",
			},
		}
		p.On("ProcessRun", "app1", "service1", opts).Return(nil, fmt.Errorf("err1"))
		err := c.Post("/apps/app1/services/service1/processes", ro, &p1)
		require.EqualError(t, err, "err1")
		require.Nil(t, p1)
	})
}

func TestProcessRunValidate(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var p1 *structs.Process
		p.On("AppGet", "app1").Return(nil, fmt.Errorf("no such app: app1"))
		err := c.Post("/apps/app1/services/service1/processes", stdsdk.RequestOptions{}, p1)
		require.EqualError(t, err, "no such app: app1")
		require.Nil(t, p1)
	})
}

func TestProcessStop(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("ProcessStop", "app1", "pid1").Return(nil)
		err := c.Delete("/apps/app1/processes/pid1", stdsdk.RequestOptions{}, nil)
		require.NoError(t, err)
	})
}

func TestProcessStopError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("ProcessStop", "app1", "pid1").Return(fmt.Errorf("err1"))
		err := c.Delete("/apps/app1/processes/pid1", stdsdk.RequestOptions{}, nil)
		require.EqualError(t, err, "err1")
	})
}
