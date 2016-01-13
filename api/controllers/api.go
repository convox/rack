package controllers

import (
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/convox/rack/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/convox/rack/Godeps/_workspace/src/golang.org/x/net/websocket"
	"github.com/convox/rack/api/httperr"
)

var RequestTimeout time.Duration = 3600 * time.Second

type ApiHandlerFunc func(http.ResponseWriter, *http.Request) *httperr.Error
type ApiWebsocketFunc func(*websocket.Conn) *httperr.Error

func api(at string, handler ApiHandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		start := time.Now()

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
			rw.WriteHeader(err.Code())
			RenderError(rw, err)
			logError(at, err)
			return
		}

		log.WithFields(log.Fields{
			"ns":                      "kernel",
			"at":                      at,
			"state":                   "success",
			"measure#handler.elapsed": fmt.Sprintf("%0.3fms", float64(time.Now().Sub(start).Nanoseconds())/1000000),
		}).Info()
	}
}

func logError(at string, err *httperr.Error) {
	l := log.WithFields(log.Fields{
		"ns":    "kernel",
		"at":    at,
		"state": "error",
	})

	if err.User() {
		l.WithField("count#error.user", 1).Warn(err.Error())
		return
	}

	err.Save()

	id := rand.Int31()

	l.WithFields(log.Fields{
		"id":          id,
		"count#error": 1,
	}).Warn(err.Error())

	for i, t := range err.Trace() {
		l.WithFields(log.Fields{
			"id":   id,
			"line": i,
		}).Warn(t)
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
		start := time.Now()

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
			logError(at, err)
			return
		}

		log.WithFields(log.Fields{
			"ns":    "kernel",
			"at":    at,
			"state": "success",
			"measure#websocket.handler.elapsed": fmt.Sprintf("%0.3fms", float64(time.Now().Sub(start).Nanoseconds())/1000000),
		}).Info()
	})
}

// Sends "true" to the done channel when either
// the websocket is closed or after a timeout
func signalWsClose(ws *websocket.Conn, done chan bool) {
	buf := make([]byte, 0)
	expires := time.Now().Add(RequestTimeout)
	for {
		_, err := ws.Read(buf)
		expired := time.Now().After(expires)
		if err == io.EOF || expired {
			done <- true
			return
		}
	}
}
