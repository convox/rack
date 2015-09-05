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
	"github.com/convox/kernel/Godeps/_workspace/src/golang.org/x/net/websocket"
	"github.com/convox/kernel/helpers"

	"github.com/convox/kernel/controllers"
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

	// router.HandleFunc("/", redirect("/apps")).Methods("GET")

	router.HandleFunc("/check", check).Methods("GET")

	router.HandleFunc("/apps", controllers.AppList).Methods("GET")
	router.HandleFunc("/apps", controllers.AppCreate).Methods("POST")
	router.HandleFunc("/apps/{app}", controllers.AppShow).Methods("GET")
	router.HandleFunc("/apps/{app}", controllers.AppUpdate).Methods("POST")
	router.HandleFunc("/apps/{app}", controllers.AppDelete).Methods("DELETE")
	router.HandleFunc("/apps/{app}/available", controllers.AppNameAvailable).Methods("GET")
	router.HandleFunc("/apps/{app}/build", controllers.BuildCreate).Methods("POST")
	router.HandleFunc("/apps/{app}/builds", controllers.BuildList).Methods("GET")
	router.HandleFunc("/apps/{app}/builds/{build}", controllers.BuildGet).Methods("GET")
	router.Handle("/apps/{app}/builds/{build}/logs", websocket.Handler(controllers.BuildLogs)).Methods("GET")
	router.HandleFunc("/apps/{app}/builds/{build}/status", controllers.BuildStatus).Methods("GET")
	router.HandleFunc("/apps/{app}/changes", controllers.AppChanges).Methods("GET")
	router.HandleFunc("/apps/{app}/debug", controllers.AppDebug).Methods("GET")
	router.HandleFunc("/apps/{app}/deployments", controllers.AppDeployments).Methods("GET")
	router.HandleFunc("/apps/{app}/environment", controllers.AppEnvironment).Methods("GET")
	router.HandleFunc("/apps/{app}/environment", controllers.EnvironmentSet).Methods("POST")
	router.HandleFunc("/apps/{app}/environment/{name}", controllers.EnvironmentCreate).Methods("POST")
	router.HandleFunc("/apps/{app}/environment/{name}", controllers.EnvironmentDelete).Methods("DELETE")
	router.HandleFunc("/apps/{app}/events", controllers.AppEvents).Methods("GET")
	router.HandleFunc("/apps/{app}/logs", controllers.AppLogs)
	router.HandleFunc("/apps/{app}/logs/stream", controllers.AppStream)
	router.HandleFunc("/apps/{app}/processes", controllers.ProcessList).Methods("GET")
	router.HandleFunc("/apps/{app}/processes/{process}", controllers.ProcessShow).Methods("GET")
	router.HandleFunc("/apps/{app}/processes/{id}", controllers.ProcessStop).Methods("DELETE")
	router.HandleFunc("/apps/{app}/processes/{id}/top", controllers.ProcessTop).Methods("GET")
	router.HandleFunc("/apps/{app}/processes/{process}/logs", controllers.ProcessLogs).Methods("GET")
	router.Handle("/apps/{app}/processes/{process}/run", websocket.Handler(controllers.ProcessRunAttached)).Methods("GET")
	router.HandleFunc("/apps/{app}/processes/{process}/run", controllers.ProcessRun).Methods("POST")
	router.HandleFunc("/apps/{app}/promote", controllers.AppPromote).Methods("POST")
	router.HandleFunc("/apps/{app}/releases", controllers.AppReleases).Methods("GET")
	router.HandleFunc("/apps/{app}/releases", controllers.ReleaseCreate).Methods("POST")
	router.HandleFunc("/apps/{app}/releases/{release}", controllers.ReleaseShow).Methods("GET")
	router.HandleFunc("/apps/{app}/releases/{release}/promote", controllers.ReleasePromote).Methods("POST")
	router.HandleFunc("/apps/{app}/resources", controllers.AppResources).Methods("GET")
	router.HandleFunc("/apps/{app}/services", controllers.ServiceLink).Methods("POST")
	router.HandleFunc("/apps/{app}/services/{name}", controllers.ServiceUnlink).Methods("DELETE")
	router.HandleFunc("/apps/{app}/status", controllers.AppStatus).Methods("GET")
	router.HandleFunc("/services", controllers.ServiceList).Methods("GET")
	router.HandleFunc("/services", controllers.ServiceCreate).Methods("POST")
	router.HandleFunc("/services/{service}", controllers.ServiceShow).Methods("GET")
	router.HandleFunc("/services/{service}/status", controllers.ServiceStatus).Methods("GET")
	router.HandleFunc("/services/{service}", controllers.ServiceDelete).Methods("DELETE")
	router.HandleFunc("/services/{service}/logs", controllers.ServiceLogs).Methods("GET")
	router.HandleFunc("/services/{service}/logs/stream", controllers.ServiceStream).Methods("GET")
	router.HandleFunc("/services/types/{type}", controllers.ServiceNameList).Methods("GET")
	router.HandleFunc("/settings", controllers.SettingsList).Methods("GET")
	router.HandleFunc("/settings", controllers.SettingsUpdate).Methods("POST")
	router.HandleFunc("/system", controllers.SystemShow).Methods("GET")
	router.HandleFunc("/system", controllers.SystemUpdate).Methods("POST")
	router.HandleFunc("/top/{metric}", controllers.ClusterTop).Methods("GET")
	router.HandleFunc("/version", controllers.VersionGet).Methods("GET")
	router.HandleFunc("/boom", controllers.Boom).Methods("GET")

	n := negroni.New(
		negroni.NewRecovery(),
		nlogger.New("ns=kernel", nil),
		negroni.NewStatic(http.Dir("public")),
	)

	n.Use(negroni.HandlerFunc(NewPanicHandler(logger.New("ns=kernel"))))
	n.Use(negroni.HandlerFunc(parseForm))
	n.Use(negroni.HandlerFunc(basicAuthentication))
	n.UseHandler(router)
	n.Run(fmt.Sprintf(":%s", port))
}
