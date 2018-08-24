package api_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestProxy(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		ro := stdsdk.RequestOptions{
			Body: strings.NewReader("in"),
		}
		p.On("Proxy", "host", 5000, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			w := args.Get(2).(io.Writer)
			w.Write([]byte("out"))
			r := args.Get(2).(io.Reader)
			data := make([]byte, 2)
			n, err := r.Read(data)
			require.NoError(t, err)
			require.Equal(t, 2, n)
			require.Equal(t, "in", string(data))
		})
		r, err := c.Websocket("/proxy/host/5000", ro)
		require.NoError(t, err)
		data := make([]byte, 3)
		n, err := r.Read(data)
		require.NoError(t, err)
		require.Equal(t, 3, n)
		require.Equal(t, "out", string(data))
	})
}

func TestProxyError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("Proxy", "host", 5000, mock.Anything).Return(fmt.Errorf("err1"))
		r, err := c.Websocket("/proxy/host/5000", stdsdk.RequestOptions{})
		require.NoError(t, err)
		d, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, []byte("ERROR: err1\n"), d)
	})
}
