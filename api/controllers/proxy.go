package controllers

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/convox/rack/api/httperr"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

func Proxy(ws *websocket.Conn) *httperr.Error {
	vars := mux.Vars(ws.Request())
	host := vars["host"]
	port := vars["port"]

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", host, port), 3*time.Second)

	if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
		return httperr.Errorf(403, "timeout")
	}

	if err != nil {
		return httperr.Server(err)
	}

	var wg sync.WaitGroup

	wg.Add(2)
	go copyAsync(ws, conn, &wg)
	go copyAsync(conn, ws, &wg)
	wg.Wait()

	return nil
}

func copyAsync(dst io.Writer, src io.Reader, wg *sync.WaitGroup) {
	defer wg.Done()
	io.Copy(dst, src)
}
