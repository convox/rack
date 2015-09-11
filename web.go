package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/codegangsta/negroni"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/nlogger"
	"github.com/convox/kernel/controllers"
	"github.com/convox/kernel/helpers"
)

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

const MinimumClientVersion = "20150911185302"

func versionCheck(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if r.URL.Path == "/system" {
		next(rw, r)
		return
	}

	if strings.HasPrefix(r.Header.Get("User-Agent"), "curl/") {
		next(rw, r)
		return
	}

	switch v := r.Header.Get("Version"); v {
	case "":
		rw.WriteHeader(403)
		rw.Write([]byte("client outdated, please update with `convox update`"))
	case "dev":
		next(rw, r)
	default:
		if v < MinimumClientVersion {
			controllers.RenderForbidden(rw, "client outdated, please update with `convox update`")
			return
		}

		next(rw, r)
	}
}

var port string = "5000"

func startWeb() {
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	n := negroni.New()

	n.Use(negroni.HandlerFunc(recovery))
	n.Use(negroni.HandlerFunc(development))
	n.Use(negroni.HandlerFunc(versionCheck))
	n.Use(nlogger.New("ns=kernel", nil))

	n.UseHandler(controllers.NewRouter())

	n.Run(fmt.Sprintf(":%s", port))
}
