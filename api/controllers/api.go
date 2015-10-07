package controllers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ddollar/logger"
	"github.com/stvp/rollbar"
	"golang.org/x/net/websocket"
)

type ApiHandlerFunc func(http.ResponseWriter, *http.Request) error
type ApiWebsocketFunc func(*websocket.Conn) *Error

func api(at string, handler ApiHandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		log := logger.New("ns=kernel").At(at).Start()

		if !passwordCheck(r) {
			rw.Header().Set("WWW-Authenticate", `Basic realm="Convox System"`)
			rw.WriteHeader(401)
			rw.Write([]byte("invalid authorization"))
			return
		}

		if !versionCheck(r) {
			rw.WriteHeader(403)
			rw.Write([]byte("client outdated, please update with `convox update`"))
			return
		}

		err := handler(rw, r)

		if err != nil {
			log.Error(err)
			rollbar.Error(rollbar.ERR, err)
			RenderError(rw, err)
			return
		}

		log.Log("state=success")
	}
}

func passwordCheck(r *http.Request) bool {
	if os.Getenv("PASSWORD") == "" {
		return true
	}

	auth := r.Header.Get("Authorization")

	if auth == "" {
		return false
	}

	if !strings.HasPrefix(auth, "Basic ") {
		return false
	}

	c, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))

	if err != nil {
		return false
	}

	parts := strings.SplitN(string(c), ":", 2)

	if len(parts) != 2 || parts[1] != os.Getenv("PASSWORD") {
		return false
	}

	return true
}

const MinimumClientVersion = "20150911185301"

func versionCheck(r *http.Request) bool {
	if r.URL.Path == "/system" {
		return true
	}

	if strings.HasPrefix(r.Header.Get("User-Agent"), "curl/") {
		return true
	}

	switch v := r.Header.Get("Version"); v {
	case "":
		return false
	case "dev":
		return true
	default:
		return v >= MinimumClientVersion
	}

	return false
}

func ws(at string, handler ApiWebsocketFunc) websocket.Handler {
	return websocket.Handler(func(ws *websocket.Conn) {
		log := logger.New("ns=kernel").At(at).Start()

		if !passwordCheck(ws.Request()) {
			ws.Write([]byte("ERROR: invalid authorization\n"))
			return
		}

		if !versionCheck(ws.Request()) {
			ws.Write([]byte("client outdated, please update with `convox update`\n"))
			return
		}

		err := handler(ws)

		if err != nil {
			ws.Write([]byte(fmt.Sprintf("ERROR: %v\n", err)))

			if err.System() {
				err.Log(log)
			} else {
				log.Log("state=error type=user message=%q", err.Error())
			}

			return
		}

		log.Log("state=success")
	})
}
