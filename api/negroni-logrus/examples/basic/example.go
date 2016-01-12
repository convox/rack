package main

import (
	"fmt"
	"net/http"

	"github.com/convox/rack/api/negroni-logrus"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/codegangsta/negroni"
)

func main() {
	r := http.NewServeMux()
	r.HandleFunc(`/`, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "success!\n")
	})

	n := negroni.New()
	n.Use(negronilogrus.NewMiddleware())
	n.UseHandler(r)

	n.Run(":9999")
}
