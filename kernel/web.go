package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/convox/convox/kernel/controllers"
	"github.com/convox/convox/kernel/controllers/apps"
	"github.com/convox/convox/kernel/controllers/clusters"

	"github.com/convox/convox/Godeps/_workspace/src/github.com/codegangsta/negroni"
	"github.com/convox/convox/Godeps/_workspace/src/github.com/gorilla/mux"
)

var port string = "3000"

func redirect(path string) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		http.Redirect(rw, r, "/clusters", http.StatusFound)
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

	router.HandleFunc("/", redirect("/clusters")).Methods("GET")
	router.HandleFunc("/clusters", clusters.List).Methods("GET")
	router.HandleFunc("/clusters", clusters.Create).Methods("POST")
	router.HandleFunc("/clusters/{cluster}", clusters.Show).Methods("GET")
	router.HandleFunc("/clusters/{cluster}/delete", clusters.Delete).Methods("POST")
	router.HandleFunc("/clusters/{cluster}/apps", apps.Create).Methods("POST")
	router.HandleFunc("/clusters/{cluster}/apps/{app}", apps.Show).Methods("GET")
	router.HandleFunc("/clusters/{cluster}/apps/{app}/delete", apps.Delete).Methods("POST")
	router.HandleFunc("/clusters/{cluster}/apps/{app}/processes/{process}", controllers.ProcessShow).Methods("GET")
	router.HandleFunc("/settings", controllers.Settings).Methods("GET")

	router.HandleFunc("/setup", func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(rw, "Needs setup")
	}).Methods("GET")

	n := negroni.Classic()
	n.Use(negroni.HandlerFunc(parseForm))
	n.UseHandler(router)
	n.Run(fmt.Sprintf(":%s", port))
}
