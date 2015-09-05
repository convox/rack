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

var port string = "5000"

func redirect(path string) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		http.Redirect(rw, r, path, http.StatusFound)
	}
}

func parseForm(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// r.ParseMultipartForm(2048)
	next(rw, r)
}

func authRequired(rw http.ResponseWriter) {
	rw.Header().Set("WWW-Authenticate", `Basic realm="Convox"`)
	rw.WriteHeader(401)
	rw.Write([]byte("unauthorized"))
}

func basicAuthentication(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if r.RequestURI == "/check" {
		next(rw, r)
		return
	}

	if password := os.Getenv("PASSWORD"); password != "" {
		auth := r.Header.Get("Authorization")

		if auth == "" {
			authRequired(rw)
			return
		}

		if !strings.HasPrefix(auth, "Basic ") {
			authRequired(rw)
			return
		}

		c, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))

		if err != nil {
			return
		}

		parts := strings.SplitN(string(c), ":", 2)

		if len(parts) != 2 || parts[1] != password {
			authRequired(rw)
			return
		}
	}

	next(rw, r)
}

func development(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Access-Control-Allow-Headers", "*")
	rw.Header().Set("Access-Control-Allow-Methods", "*")

	next(rw, r)
}

type ApiHandlerFunc func(http.ResponseWriter, *http.Request) error
type ApiWebsocketFunc func(*websocket.Conn) error

func api(at string, handler ApiHandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		log := logger.New("ns=kernel").At(at).Start()
		err := handler(rw, r)

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

func check(rw http.ResponseWriter, r *http.Request) {
	rw.Write([]byte("ok"))
}

// This function returns a negroni middleware with a closure
// over the Logger instance
func NewPanicHandler(log *logger.Logger) func(http.ResponseWriter, *http.Request, http.HandlerFunc) {
	// Handler for any panics within an HTTP request
	return func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		defer recoverWith(func(err error) {
			helpers.Error(log, err)
			http.Error(rw, err.Error(), http.StatusInternalServerError)
		})

		next(rw, r)
	}
}

func startWeb() {
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	router := mux.NewRouter()

	// utility
	router.HandleFunc("/boom", controllers.Boom).Methods("GET")
	router.HandleFunc("/check", check).Methods("GET")

	// normalized
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
	router.HandleFunc("/services", api("service.list", controllers.ServiceList)).Methods("GET")
	router.HandleFunc("/services", api("service.create", controllers.ServiceCreate)).Methods("POST")
	router.HandleFunc("/services/{service}", api("service.show", controllers.ServiceShow)).Methods("GET")
	router.HandleFunc("/services/{service}", api("service.delete", controllers.ServiceDelete)).Methods("DELETE")

	// websockets
	router.Handle("/apps/{app}/logs", ws("app.logs", controllers.AppLogs)).Methods("GET")
	router.Handle("/apps/{app}/builds/{build}/logs", ws("build.logs", controllers.BuildLogs)).Methods("GET")
	router.Handle("/apps/{app}/processes/{process}/run", ws("process.run.attach", controllers.ProcessRunAttached)).Methods("GET")
	router.Handle("/services/{service}/logs", ws("service.logs", controllers.ServiceLogs)).Methods("GET")

	// limbo
	// router.HandleFunc("/apps/{app}/debug", controllers.AppDebug).Methods("GET")
	// router.HandleFunc("/apps/{app}/processes/{id}", controllers.ProcessStop).Methods("DELETE")
	// router.HandleFunc("/apps/{app}/processes/{id}/top", controllers.ProcessTop).Methods("GET")
	// router.HandleFunc("/top/{metric}", controllers.ClusterTop).Methods("GET")

	// todo
	router.HandleFunc("/system", controllers.SystemShow).Methods("GET")
	router.HandleFunc("/system", controllers.SystemUpdate).Methods("POST")
	router.HandleFunc("/version", controllers.VersionGet).Methods("GET")

	n := negroni.New(
		negroni.NewRecovery(),
		nlogger.New("ns=kernel", nil),
		negroni.NewStatic(http.Dir("public")),
	)

	if os.Getenv("DEVELOPMENT") == "true" {
		n.Use(negroni.HandlerFunc(development))
	}

	n.Use(negroni.HandlerFunc(NewPanicHandler(logger.New("ns=kernel"))))
	n.Use(negroni.HandlerFunc(parseForm))
	n.Use(negroni.HandlerFunc(basicAuthentication))
	n.UseHandler(router)

	n.Run(fmt.Sprintf(":%s", port))
}
