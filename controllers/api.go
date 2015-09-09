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
	"github.com/gorilla/mux"
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

func NewRouter() (router *mux.Router) {
	router = mux.NewRouter()

	router.HandleFunc("/apps", api("app.list", AppList)).Methods("GET")
	router.HandleFunc("/apps", api("app.create", AppCreate)).Methods("POST")
	router.HandleFunc("/apps/{app}", api("app.get", AppShow)).Methods("GET")
	router.HandleFunc("/apps/{app}", api("app.delete", AppDelete)).Methods("DELETE")
	router.HandleFunc("/apps/{app}/builds", api("build.list", BuildList)).Methods("GET")
	router.HandleFunc("/apps/{app}/builds", api("build.create", BuildCreate)).Methods("POST")
	router.HandleFunc("/apps/{app}/builds/{build}", api("build.get", BuildGet)).Methods("GET")
	router.HandleFunc("/apps/{app}/environment", api("environment.list", EnvironmentList)).Methods("GET")
	router.HandleFunc("/apps/{app}/environment", api("environment.set", EnvironmentSet)).Methods("POST")
	router.HandleFunc("/apps/{app}/environment/{name}", api("environment.delete", EnvironmentDelete)).Methods("DELETE")
	router.HandleFunc("/apps/{app}/formation", api("formation.list", FormationList)).Methods("GET")
	router.HandleFunc("/apps/{app}/formation/{process}", api("formation.set", FormationSet)).Methods("POST")
	router.HandleFunc("/apps/{app}/processes", api("process.list", ProcessList)).Methods("GET")
	router.HandleFunc("/apps/{app}/processes/{process}", api("process.get", ProcessShow)).Methods("GET")
	router.HandleFunc("/apps/{app}/processes/{process}", api("process.stop", ProcessStop)).Methods("DELETE")
	router.HandleFunc("/apps/{app}/processes/{process}/run", api("process.run.detach", ProcessRunDetached)).Methods("POST")
	router.HandleFunc("/apps/{app}/releases", api("release.list", ReleaseList)).Methods("GET")
	router.HandleFunc("/apps/{app}/releases/{release}", api("release.get", ReleaseShow)).Methods("GET")
	router.HandleFunc("/apps/{app}/releases/{release}/promote", api("release.promote", ReleasePromote)).Methods("POST")
	router.HandleFunc("/services", api("service.list", ServiceList)).Methods("GET")
	router.HandleFunc("/services", api("service.create", ServiceCreate)).Methods("POST")
	router.HandleFunc("/services/{service}", api("service.show", ServiceShow)).Methods("GET")
	router.HandleFunc("/services/{service}", api("service.delete", ServiceDelete)).Methods("DELETE")
	router.HandleFunc("/system", api("system.show", SystemShow)).Methods("GET")
	router.HandleFunc("/system", api("system.update", SystemUpdate)).Methods("PUT")

	// websockets
	router.Handle("/apps/{app}/logs", ws("app.logs", AppLogs)).Methods("GET")
	router.Handle("/apps/{app}/builds/{build}/logs", ws("build.logs", BuildLogs)).Methods("GET")
	router.Handle("/apps/{app}/processes/{process}/run", ws("process.run.attach", ProcessRunAttached)).Methods("GET")
	router.Handle("/services/{service}/logs", ws("service.logs", ServiceLogs)).Methods("GET")

	// utility
	router.HandleFunc("/boom", UtilityBoom).Methods("GET")
	router.HandleFunc("/check", UtilityCheck).Methods("GET")

	// limbo
	// auth.HandleFunc("/apps/{app}/debug", controllers.AppDebug).Methods("GET")
	// auth.HandleFunc("/apps/{app}/processes/{id}", controllers.ProcessStop).Methods("DELETE")
	// auth.HandleFunc("/apps/{app}/processes/{id}/top", controllers.ProcessTop).Methods("GET")
	// auth.HandleFunc("/top/{metric}", controllers.ClusterTop).Methods("GET")

	return
}
