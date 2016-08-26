package controllers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/convox/logger"
	"github.com/convox/rack/api/httperr"
	"golang.org/x/net/websocket"
)

var RequestTimeout time.Duration = 3600 * time.Second

type ApiHandlerFunc func(http.ResponseWriter, *http.Request) *httperr.Error
type ApiWebsocketFunc func(*websocket.Conn) *httperr.Error

func api(at string, handler ApiHandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		log := logger.New("ns=api.controllers").At(at).Start()

		if !passwordCheck(r) {
			log.Errorf("invalid authorization")
			rw.Header().Set("WWW-Authenticate", `Basic realm="Convox System"`)
			rw.WriteHeader(401)
			rw.Write([]byte("invalid authorization"))
			return
		}

		if !versionCheck(r) {
			log.Errorf("invalid version")
			rw.WriteHeader(403)
			rw.Write([]byte("client outdated, please update with `convox update`"))
			return
		}

		err := handler(rw, r)

		if err != nil {
			log.Error(err)
			rw.WriteHeader(err.Code())
			RenderError(rw, err)
			return
		}

		log.Success()
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
		log := logger.New("ns=api.controllers").At(at).Start()

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
			log.Error(err)
			ws.Write([]byte(fmt.Sprintf("ERROR: %v\n", err)))
			return
		}

		log.Success()
	})
}
