package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/codegangsta/negroni"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/web/controllers"
)

var port string = "5000"

func redirect(path string) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		http.Redirect(rw, r, path, http.StatusFound)
	}
}

func parseForm(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	r.ParseMultipartForm(2048)
	next(rw, r)
}

func main() {
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	router := mux.NewRouter()

	router.HandleFunc("/", redirect("/apps")).Methods("GET")

	router.HandleFunc("/apps", controllers.AppList).Methods("GET")
	router.HandleFunc("/apps", controllers.AppCreate).Methods("POST")
	router.HandleFunc("/apps/{app}", controllers.AppShow).Methods("GET")
	router.HandleFunc("/apps/{app}", controllers.AppDelete).Methods("DELETE")
	router.HandleFunc("/apps/{app}/build", controllers.AppBuild).Methods("POST")
	router.HandleFunc("/apps/{app}/promote", controllers.AppPromote).Methods("POST")

	n := negroni.Classic()
	n.Use(negroni.HandlerFunc(parseForm))
	n.UseHandler(router)
	n.Run(fmt.Sprintf(":%s", port))
}
