package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/codegangsta/negroni"
	"github.com/convox/logger"
	"github.com/convox/nlogger"
	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/api/helpers"
)

var port string = "5000"

func main() {
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	n := negroni.New()

	n.Use(negroni.HandlerFunc(recovery))
	n.Use(negroni.HandlerFunc(development))
	n.Use(nlogger.New("ns=api.web", nil))

	n.UseHandler(controllers.NewRouter())

	n.Run(fmt.Sprintf(":%s", port))
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

func recoverWith(f func(err error)) {
	if r := recover(); r != nil {
		// coerce r to error type
		err, ok := r.(error)
		if !ok {
			err = fmt.Errorf("%v", r)
		}

		f(err)
	}
}
