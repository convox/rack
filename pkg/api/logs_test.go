package api_test

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var logsOptions = structs.LogsOptions{Since: options.Duration(2 * time.Minute)}

var logsRoutes = []struct {
	name string
	path string
	on   func(p *structs.MockProvider, r io.ReadCloser)
}{
	{"AppLogs", "/apps/app1/logs", func(p *structs.MockProvider, r io.ReadCloser) {
		p.On("AppLogs", "app1", logsOptions).Return(r, nil)
	}},
	{"SystemLogs", "/system/logs", func(p *structs.MockProvider, r io.ReadCloser) {
		p.On("SystemLogs", logsOptions).Return(r, nil)
	}},
	{"BuildLogs", "/apps/app1/builds/build1/logs", func(p *structs.MockProvider, r io.ReadCloser) {
		p.On("BuildLogs", "app1", "build1", logsOptions).Return(r, nil)
	}},
}

type heldReader struct {
	*io.PipeReader
	closed chan struct{}
	once   sync.Once
}

func (r *heldReader) Close() error {
	r.once.Do(func() { close(r.closed) })
	return nil
}

type logsStream struct {
	conn   *websocket.Conn
	pw     *io.PipeWriter
	ctxs   chan context.Context
	closed chan struct{}
}

func testLogs(t *testing.T, fn func(t *testing.T, s *logsStream)) {
	for _, route := range logsRoutes {
		t.Run(route.name, func(t *testing.T) {
			testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
				s := &logsStream{
					ctxs:   make(chan context.Context, 1),
					closed: make(chan struct{}),
				}

				p.On("WithContext", mock.Anything).Unset()
				p.On("WithContext", mock.Anything).Return(func(ctx context.Context) structs.Provider {
					s.ctxs <- ctx
					return p
				})

				pr, pw := io.Pipe()
				defer func() { _ = pw.Close() }()

				s.pw = pw

				route.on(p, &heldReader{PipeReader: pr, closed: s.closed})

				conn, _, err := websocket.DefaultDialer.Dial("ws://"+c.Endpoint.Host+route.path, nil)
				require.NoError(t, err)
				defer func() { _ = conn.Close() }()

				s.conn = conn

				fn(t, s)
			})
		})
	}
}

func (s *logsStream) context(t *testing.T) context.Context {
	t.Helper()

	select {
	case ctx := <-s.ctxs:
		return ctx
	case <-time.After(2 * time.Second):
		t.Fatal("the provider was not called")
	}

	return nil
}

func requireGone(t *testing.T, s *logsStream) {
	t.Helper()

	ctx := s.context(t)

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("the provider context was not cancelled")
	}

	select {
	case <-s.closed:
	case <-time.After(2 * time.Second):
		t.Fatal("the provider reader was not closed")
	}
}

func TestLogsClientTCPClose(t *testing.T) {
	testLogs(t, func(t *testing.T, s *logsStream) {
		_, err := s.pw.Write([]byte("line\n"))
		require.NoError(t, err)

		_, data, err := s.conn.ReadMessage()
		require.NoError(t, err)
		require.Equal(t, "line\n", string(data))

		require.NoError(t, s.conn.UnderlyingConn().Close())

		requireGone(t, s)
	})
}

func TestLogsClientCloseFrame(t *testing.T) {
	testLogs(t, func(t *testing.T, s *logsStream) {
		err := s.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
		require.NoError(t, err)

		requireGone(t, s)
	})
}

func TestLogsClientFramesIgnored(t *testing.T) {
	testLogs(t, func(t *testing.T, s *logsStream) {
		pongs := 0
		s.conn.SetPongHandler(func(string) error {
			pongs++
			return nil
		})

		require.NoError(t, s.conn.WriteMessage(websocket.BinaryMessage, []byte{}))
		require.NoError(t, s.conn.WriteMessage(websocket.TextMessage, []byte("input")))
		require.NoError(t, s.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(time.Second)))

		time.Sleep(200 * time.Millisecond)

		_, err := s.pw.Write([]byte("line\n"))
		require.NoError(t, err)

		_, data, err := s.conn.ReadMessage()
		require.NoError(t, err)
		require.Equal(t, "line\n", string(data))
		require.Zero(t, pongs)

		require.NoError(t, s.context(t).Err())

		select {
		case <-s.closed:
			t.Fatal("the provider reader was closed")
		default:
		}
	})
}

func TestLogsFiniteStreamClientStaysOpen(t *testing.T) {
	testLogs(t, func(t *testing.T, s *logsStream) {
		_, err := s.pw.Write([]byte("test"))
		require.NoError(t, err)
		require.NoError(t, s.pw.Close())

		require.NoError(t, s.conn.SetReadDeadline(time.Now().Add(2*time.Second)))

		mt, data, err := s.conn.ReadMessage()
		require.NoError(t, err)
		require.Equal(t, websocket.TextMessage, mt)
		require.Equal(t, "test", string(data))

		mt, data, err = s.conn.ReadMessage()
		require.NoError(t, err)
		require.Equal(t, websocket.BinaryMessage, mt)
		require.Empty(t, data)
	})
}

func TestLogsCopyErrorReachesClient(t *testing.T) {
	testLogs(t, func(t *testing.T, s *logsStream) {
		require.NoError(t, s.pw.CloseWithError(errors.New("err2")))

		require.NoError(t, s.conn.SetReadDeadline(time.Now().Add(2*time.Second)))

		_, data, err := s.conn.ReadMessage()
		require.NoError(t, err)
		require.Equal(t, "ERROR: err2\n", string(data))
	})
}
