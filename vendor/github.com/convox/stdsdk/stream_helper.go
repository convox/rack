package stdsdk

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// some caveats apply: https://github.com/gorilla/websocket/issues/441
type AdapterWs struct {
	conn       *websocket.Conn
	readMutex  sync.Mutex
	writeMutex sync.Mutex
	reader     io.Reader
}

func NewAdapterWs(conn *websocket.Conn) *AdapterWs {
	return &AdapterWs{
		conn:       conn,
		readMutex:  sync.Mutex{},
		writeMutex: sync.Mutex{},
	}
}

func (a *AdapterWs) Read(b []byte) (int, error) {
	// Read() can be called concurrently, and we mutate some internal state here
	a.readMutex.Lock()
	defer a.readMutex.Unlock()

	if a.reader == nil {
		messageType, reader, err := a.conn.NextReader()
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			return 0, io.EOF
		}
		if err != nil {
			return 0, err
		}

		if messageType == websocket.BinaryMessage {
			return 0, io.EOF
		}

		if messageType != websocket.TextMessage {
			return 0, nil
		}

		a.reader = reader
	}

	bytesRead, err := a.reader.Read(b)
	if err != nil {
		a.reader = nil

		// EOF for the current Websocket frame, more will probably come so..
		if err == io.EOF {
			// .. we must hide this from the caller since our semantics are a
			// stream of bytes across many frames
			err = nil
		}
	}

	return bytesRead, err
}

func (a *AdapterWs) ReadMessage() (int, []byte, error) {
	a.readMutex.Lock()
	defer a.readMutex.Unlock()
	return a.conn.ReadMessage()
}

func (a *AdapterWs) Write(b []byte) (int, error) {
	a.writeMutex.Lock()
	defer a.writeMutex.Unlock()

	nextWriter, err := a.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return 0, err
	}

	bytesWritten, err := nextWriter.Write(b)
	nextWriter.Close()

	return bytesWritten, err
}

func (a *AdapterWs) WriteMessage(messageType int, data []byte) error {
	a.writeMutex.Lock()
	defer a.writeMutex.Unlock()
	return a.conn.WriteMessage(messageType, data)
}

func (a *AdapterWs) Close() error {
	return a.conn.Close()
}

func (a *AdapterWs) LocalAddr() net.Addr {
	return a.conn.LocalAddr()
}

func (a *AdapterWs) RemoteAddr() net.Addr {
	return a.conn.RemoteAddr()
}

func (a *AdapterWs) SetDeadline(t time.Time) error {
	if err := a.SetReadDeadline(t); err != nil {
		return err
	}

	return a.SetWriteDeadline(t)
}

func (a *AdapterWs) SetReadDeadline(t time.Time) error {
	return a.conn.SetReadDeadline(t)
}

func (a *AdapterWs) SetWriteDeadline(t time.Time) error {
	return a.conn.SetWriteDeadline(t)
}

func chanFromReader(r io.Reader) (chan []byte, chan error) {
	c := make(chan []byte)
	errCh := make(chan error)

	go func() {
		b := make([]byte, 1024)

		for {
			n, err := r.Read(b)
			if n > 0 {
				res := make([]byte, n)
				// Copy the buffer so it doesn't get changed while read by the recipient.
				copy(res, b[:n])
				c <- res
			}
			if err != nil {
				if err != io.EOF {
					errCh <- err
				}
				c <- nil
				return
			}
		}
	}()

	return c, errCh
}

// CopyFromToWsTcp accepts a websocket connection and TCP connection and copies data between them
func CopyFromToWsTcp(wsConn *AdapterWs, tcpConn net.Conn) error {
	wsChan, wsErrChan := chanFromReader(wsConn)
	tcpChan, tcpErrChan := chanFromReader(tcpConn)

	defer wsConn.Close()
	defer tcpConn.Close()
	for {
		select {
		case wsData := <-wsChan:
			if wsData == nil {
				return fmt.Errorf("TCP connection closed: D: %s, S: %s", tcpConn.LocalAddr().String(), wsConn.RemoteAddr().String())
			} else {
				_, err := tcpConn.Write(wsData)
				if err == io.ErrClosedPipe {
					return fmt.Errorf("TCP connection closed: D: %s, S: %s", tcpConn.LocalAddr().String(), wsConn.RemoteAddr().String())
				}
			}
		case tcpData := <-tcpChan:
			if tcpData == nil {
				return fmt.Errorf("TCP connection closed: D: %s, S: %s", tcpConn.LocalAddr().String(), wsConn.LocalAddr().String())
			} else {
				_, err := wsConn.Write(tcpData)
				if err != nil {
					return fmt.Errorf("TCP connection closed: D: %s, S: %s", tcpConn.LocalAddr().String(), wsConn.LocalAddr().String())
				}
			}
		case err := <-wsErrChan:
			return err
		case err := <-tcpErrChan:
			return err
		}
	}
}

func CopyStreamToEachOther(fromConn io.ReadWriter, toConn io.ReadWriter) error {
	fromChan, fromErrChan := chanFromReader(fromConn)
	toChan, toErrChan := chanFromReader(toConn)

	if xc, ok := toConn.(io.Closer); ok {
		defer xc.Close()
	}

	if yc, ok := fromConn.(io.Closer); ok {
		defer yc.Close()
	}

	for {
		select {
		case toData := <-toChan:
			if toData == nil {
				return fmt.Errorf("TCP connection closed from destination")
			} else {
				_, err := fromConn.Write(toData)
				if err != nil {
					if err == io.ErrClosedPipe {
						return nil
					}
					return err
				}
			}
		case fromData := <-fromChan:
			if fromData == nil {
				return fmt.Errorf("TCP connection closed from source")
			} else {
				_, err := toConn.Write(fromData)
				if err != nil {
					if err == io.ErrClosedPipe {
						return nil
					}
					return err
				}
			}
		case err := <-toErrChan:
			return err
		case err := <-fromErrChan:
			return err
		}
	}
}

func WsKeepAlivePing(ctx context.Context, ws *AdapterWs) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			ws.WriteMessage(websocket.PingMessage, []byte{})
		}
	}
}

func copyToWS(ctx context.Context, ws *AdapterWs, r io.Reader) error {
	if r == nil {
		return nil
	}
	rChan, rErrChan := chanFromReader(r)

	// used as eof
	defer ws.WriteMessage(websocket.BinaryMessage, []byte{})

	for {
		select {
		case <-ctx.Done():
			return nil
		case data := <-rChan:
			if data == nil {
				return fmt.Errorf("TCP connection closed from destination")
			} else {
				_, err := ws.Write(data)
				if err != nil {
					if err == io.ErrClosedPipe {
						return nil
					}
					return err
				}
			}
		case err := <-rErrChan:
			return err
		}
	}
}

func copyFromWS(ctx context.Context, ws *AdapterWs, w io.WriteCloser) error {
	wsChan, wsErrChan := chanFromReader(ws)

	defer w.Close()

	for {
		select {
		case <-ctx.Done():
			return nil
		case data := <-wsChan:
			if data == nil {
				return fmt.Errorf("TCP connection closed from destination")
			} else {
				_, err := w.Write(data)
				if err != nil {
					if err == io.ErrClosedPipe {
						return nil
					}
					return err
				}
			}
		case err := <-wsErrChan:
			return err
		}
	}
}
