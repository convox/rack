package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/convox/kernel/builder/Godeps/_workspace/src/github.com/codegangsta/negroni"
	"github.com/convox/kernel/builder/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/builder/controllers"
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

	rand.Seed(time.Now().UTC().UnixNano())

	router := mux.NewRouter()

	router.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("Hello, World!"))
	}).Methods("GET")

	router.HandleFunc("/apps/{app}/build", controllers.Build).Methods("POST")

	n := negroni.Classic()
	n.Use(negroni.HandlerFunc(parseForm))
	n.UseHandler(router)
	n.Run(fmt.Sprintf(":%s", port))
}
