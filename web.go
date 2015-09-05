package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/codegangsta/negroni"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/nlogger"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/stvp/rollbar"
	"github.com/convox/kernel/Godeps/_workspace/src/golang.org/x/net/websocket"
	"github.com/convox/kernel/controllers"
	"github.com/convox/kernel/helpers"
)

func authRequired(rw http.ResponseWriter, message string) error {
	rw.Header().Set("WWW-Authenticate", `Basic realm="Convox Rack"`)
	rw.WriteHeader(401)
	rw.Write([]byte(message))
	return fmt.Errorf(message)
}

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

func development(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if os.Getenv("DEVELOPMENT") == "true" {
		rw.Header().Set("Access-Control-Allow-Origin", "*")
		rw.Header().Set("Access-Control-Allow-Headers", "*")
		rw.Header().Set("Access-Control-Allow-Methods", "*")
	}

	next(rw, r)
}

func recovery(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	defer recoverWith(func(err error) {
		log := logger.New("ns=kernel").At("panic")
		helpers.Error(log, err)
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	})

	next(rw, r)
}

type ApiHandlerFunc func(http.ResponseWriter, *http.Request) error
type ApiWebsocketFunc func(*websocket.Conn) error

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
			controllers.RenderError(rw, err)
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

var port string = "5000"

func startWeb() {
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	router := mux.NewRouter()

	router.HandleFunc("/apps", api("app.list", controllers.AppList)).Methods("GET")
	router.HandleFunc("/apps", api("app.create", controllers.AppCreate)).Methods("POST")
	router.HandleFunc("/apps/{app}", api("app.get", controllers.AppShow)).Methods("GET")
	router.HandleFunc("/apps/{app}", api("app.delete", controllers.AppDelete)).Methods("DELETE")
	router.HandleFunc("/apps/{app}/builds", api("build.list", controllers.BuildList)).Methods("GET")
	router.HandleFunc("/apps/{app}/build", api("build.create", controllers.BuildCreate)).Methods("POST")
	router.HandleFunc("/apps/{app}/builds/{build}", api("build.get", controllers.BuildGet)).Methods("GET")
	router.HandleFunc("/apps/{app}/environment", api("environment.list", controllers.EnvironmentList)).Methods("GET")
	router.HandleFunc("/apps/{app}/environment", api("environment.set", controllers.EnvironmentSet)).Methods("POST")
	router.HandleFunc("/apps/{app}/environment/{name}", api("environment.delete", controllers.EnvironmentDelete)).Methods("DELETE")
	router.HandleFunc("/apps/{app}/processes", api("process.list", controllers.ProcessList)).Methods("GET")
	router.HandleFunc("/apps/{app}/processes/{process}", api("process.get", controllers.ProcessShow)).Methods("GET")
	router.HandleFunc("/apps/{app}/processes/{process}/run", api("process.run.detach", controllers.ProcessRunDetached)).Methods("POST")
	router.HandleFunc("/apps/{app}/processes/{process}/scale", api("process.scale", controllers.ProcessScale)).Methods("POST")
	router.HandleFunc("/apps/{app}/releases", api("release.list", controllers.ReleaseList)).Methods("GET")
	router.HandleFunc("/apps/{app}/releases/{release}", api("release.get", controllers.ReleaseShow)).Methods("GET")
	router.HandleFunc("/apps/{app}/releases/{release}/promote", api("release.promote", controllers.ReleasePromote)).Methods("POST")
	router.HandleFunc("/rack", api("rack.show", controllers.RackShow)).Methods("GET")
	router.HandleFunc("/rack", api("rack.update", controllers.RackUpdate)).Methods("PUT")
	router.HandleFunc("/services", api("service.list", controllers.ServiceList)).Methods("GET")
	router.HandleFunc("/services", api("service.create", controllers.ServiceCreate)).Methods("POST")
	router.HandleFunc("/services/{service}", api("service.show", controllers.ServiceShow)).Methods("GET")
	router.HandleFunc("/services/{service}", api("service.delete", controllers.ServiceDelete)).Methods("DELETE")

	// websockets
	router.Handle("/apps/{app}/logs", ws("app.logs", controllers.AppLogs)).Methods("GET")
	router.Handle("/apps/{app}/builds/{build}/logs", ws("build.logs", controllers.BuildLogs)).Methods("GET")
	router.Handle("/apps/{app}/processes/{process}/run", ws("process.run.attach", controllers.ProcessRunAttached)).Methods("GET")
	router.Handle("/services/{service}/logs", ws("service.logs", controllers.ServiceLogs)).Methods("GET")

	// utility
	router.HandleFunc("/boom", controllers.UtilityBoom).Methods("GET")
	router.HandleFunc("/check", controllers.UtilityCheck).Methods("GET")

	// limbo
	// auth.HandleFunc("/apps/{app}/debug", controllers.AppDebug).Methods("GET")
	// auth.HandleFunc("/apps/{app}/processes/{id}", controllers.ProcessStop).Methods("DELETE")
	// auth.HandleFunc("/apps/{app}/processes/{id}/top", controllers.ProcessTop).Methods("GET")
	// auth.HandleFunc("/top/{metric}", controllers.ClusterTop).Methods("GET")

	n := negroni.New()

	n.Use(negroni.HandlerFunc(recovery))
	n.Use(negroni.HandlerFunc(development))
	n.Use(nlogger.New("ns=kernel", nil))

	n.UseHandler(router)

	n.Run(fmt.Sprintf(":%s", port))
}
