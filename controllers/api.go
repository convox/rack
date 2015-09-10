package controllers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/stvp/rollbar"
	"github.com/convox/kernel/Godeps/_workspace/src/golang.org/x/net/websocket"
)

type ApiHandlerFunc func(http.ResponseWriter, *http.Request) error
type ApiWebsocketFunc func(*websocket.Conn) error

func authenticate(rw http.ResponseWriter, r *http.Request) error {
	if os.Getenv("PASSWORD") == "" {
		return nil
	}

	auth := r.Header.Get("Authorization")

	if auth == "" {
		return authRequired(rw, "invalid authorization header")
	}

	if !strings.HasPrefix(auth, "Basic ") {
		return authRequired(rw, "no basic auth")
	}

	c, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))

	if err != nil {
		return err
	}

	parts := strings.SplitN(string(c), ":", 2)

	if len(parts) != 2 || parts[1] != os.Getenv("PASSWORD") {
		return authRequired(rw, "invalid password")
	}

	return nil
}

func authRequired(rw http.ResponseWriter, message string) error {
	rw.Header().Set("WWW-Authenticate", `Basic realm="Convox System"`)
	rw.WriteHeader(401)
	rw.Write([]byte(message))
	return fmt.Errorf(message)
}

func api(at string, handler ApiHandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		log := logger.New("ns=kernel").At(at).Start()

		err := authenticate(rw, r)

		if err != nil {
			log.Log("state=unauthorized")
			return
		}

		err = handler(rw, r)

		if err != nil {
			log.Error(err)
			rollbar.Error(rollbar.ERR, err)
			RenderError(rw, err)
			return
		}

		log.Log("state=success")
	}
}

func ws(at string, handler ApiWebsocketFunc) websocket.Handler {
	return websocket.Handler(func(ws *websocket.Conn) {
		log := logger.New("ns=kernel").At(at).Start()
		err := handler(ws)

		if err != nil {
			ws.Write([]byte(fmt.Sprintf("ERROR: %v\n", err)))
			log.Error(err)
			rollbar.Error(rollbar.ERR, err)
			return
		}

		log.Log("state=success")
	})
}
