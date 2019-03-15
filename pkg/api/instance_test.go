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

var fxInstance = structs.Instance{
	Agent:     true,
	Cpu:       1.0,
	Id:        "instance1",
	Memory:    2.0,
	PrivateIp: "private",
	Processes: 3,
	PublicIp:  "public",
	Status:    "status",
	Started:   time.Now().UTC(),
}

func TestInstanceKeyroll(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("InstanceKeyroll").Return(nil)
		err := c.Post("/instances/keyroll", stdsdk.RequestOptions{}, nil)
		require.NoError(t, err)
	})
}

func TestInstanceKeyrollError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("InstanceKeyroll").Return(fmt.Errorf("err1"))
		err := c.Post("/instances/keyroll", stdsdk.RequestOptions{}, nil)
		require.EqualError(t, err, "err1")
	})
}

func TestInstanceList(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		i1 := structs.Instances{fxInstance, fxInstance}
		i2 := structs.Instances{}
		p.On("InstanceList").Return(i1, nil)
		err := c.Get("/instances", stdsdk.RequestOptions{}, &i2)
		require.NoError(t, err)
		require.Equal(t, i1, i2)
	})
}

func TestInstanceListError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var i1 structs.Instances
		p.On("InstanceList").Return(nil, fmt.Errorf("err1"))
		err := c.Get("/instances", stdsdk.RequestOptions{}, &i1)
		require.EqualError(t, err, "err1")
		require.Nil(t, i1)
	})
}

func TestInstanceShell(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		opts := structs.InstanceShellOptions{
			Command: options.String("command"),
			Height:  options.Int(1),
			Width:   options.Int(2),
		}
		ro := stdsdk.RequestOptions{
			Body: strings.NewReader("in"),
			Headers: stdsdk.Headers{
				"Command": "command",
				"Height":  "1",
				"Width":   "2",
			},
		}
		p.On("InstanceShell", "instance1", mock.Anything, opts).Return(1, nil).Run(func(args mock.Arguments) {
			rw := args.Get(1).(io.ReadWriter)
			rw.Write([]byte("out"))
			data, err := ioutil.ReadAll(rw)
			require.NoError(t, err)
			require.Equal(t, "in", string(data))
		})
		r, err := c.Websocket("/instances/instance1/shell", ro)
		data, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, "outF1E49A85-0AD7-4AEF-A618-C249C6E6568D:1\n", string(data))
	})
}

func TestInstanceShellError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("InstanceShell", "instance1", mock.Anything, structs.InstanceShellOptions{}).Return(0, fmt.Errorf("err1"))
		r, err := c.Websocket("/instances/instance1/shell", stdsdk.RequestOptions{})
		require.NoError(t, err)
		d, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, []byte("ERROR: err1\n"), d)
	})
}

func TestInstanceTerminate(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("InstanceTerminate", "instance1").Return(nil)
		err := c.Delete("/instances/instance1", stdsdk.RequestOptions{}, nil)
		require.NoError(t, err)
	})
}

func TestInstanceTerminateError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("InstanceTerminate", "instance1").Return(fmt.Errorf("err1"))
		err := c.Delete("/instances/instance1", stdsdk.RequestOptions{}, nil)
		require.EqualError(t, err, "err1")
	})
}
