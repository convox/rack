package api

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/convox/stdapi"
)

func renderStatusCode(w io.Writer, code int) error {
	_, err := fmt.Fprintf(w, "F1E49A85-0AD7-4AEF-A618-C249C6E6568D:%d\n", code)
	return err
}

func copyLogs(c *stdapi.Context, r io.ReadCloser, cancel context.CancelFunc) error {
	ws := c.Websocket()

	gone := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer close(done)

		ws.SetPingHandler(func(string) error { return nil })

		for {
			if _, _, err := ws.NextReader(); err != nil {
				close(gone)
				cancel()
				_ = r.Close()
				return
			}
		}
	}()

	_, err := io.Copy(c, r)

	select {
	case <-gone:
		err = nil
	default:
	}

	_ = ws.UnderlyingConn().SetReadDeadline(time.Now())
	<-done

	return err
}
