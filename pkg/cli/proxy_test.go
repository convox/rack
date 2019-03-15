package cli_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestProxy(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.ProxyOptions{
			TLS: options.Bool(false),
		}
		i.On("Proxy", "test.example.org", 5000, mock.Anything, opts).Return(nil).Run(func(args mock.Arguments) {
			buf := make([]byte, 2)
			rwc := args.Get(2).(io.ReadWriteCloser)
			n, err := rwc.Read(buf)
			require.NoError(t, err)
			require.Equal(t, 2, n)
			require.Equal(t, "in", string(buf))
			n, err = rwc.Write([]byte("out"))
			require.NoError(t, err)
			require.Equal(t, 3, n)
			rwc.Close()
		})

		port := rand.Intn(30000) + 10000

		ch := make(chan *result)

		go func() {
			res, _ := testExecute(e, fmt.Sprintf("proxy %d:test.example.org:5000", port), nil)
			ch <- res
		}()

		time.Sleep(50 * time.Millisecond)

		cn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		require.NoError(t, err)

		cn.Write([]byte("in"))

		data, err := ioutil.ReadAll(cn)
		require.NoError(t, err)
		require.Equal(t, "out", string(data))

		cli.ProxyCloser <- nil

		res := <-ch

		require.NotNil(t, res)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			fmt.Sprintf("proxying localhost:%d to test.example.org:5000", port),
			fmt.Sprintf("connect: %d", port),
		})
	})
}

func TestProxyError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		opts := structs.ProxyOptions{
			TLS: options.Bool(false),
		}
		i.On("Proxy", "test.example.org", 5000, mock.Anything, opts).Return(fmt.Errorf("err1"))

		port := rand.Intn(30000) + 10000

		ch := make(chan *result)

		go func() {
			res, _ := testExecute(e, fmt.Sprintf("proxy %d:test.example.org:5000", port), nil)
			ch <- res
		}()

		time.Sleep(100 * time.Millisecond)

		cn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		require.NoError(t, err)

		cn.Write([]byte("in"))

		data, err := ioutil.ReadAll(cn)
		require.Error(t, err)
		require.Len(t, data, 0)

		cli.ProxyCloser <- nil

		res := <-ch

		require.NotNil(t, res)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{
			fmt.Sprintf("proxying localhost:%d to test.example.org:5000", port),
			fmt.Sprintf("connect: %d", port),
		})
	})
}
