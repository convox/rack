package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/convox/rack/api/models"
	"github.com/convox/rack/manifest"
)

func main() {
	if len(os.Args) < 2 {
		die(fmt.Errorf("usage: fixture <docker-compose.yml>"))
	}

	data, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		die(err)
	}

	app := models.App{
		Name: "httpd",
		Tags: map[string]string{
			"Name":   "httpd",
			"Type":   "app",
			"System": "convox",
			"Rack":   "convox",
		},
	}

	m, err := manifest.Load(data)
	if err != nil {
		die(err)
	}

	f, err := app.Formation(*m)
	if err != nil {
		die(err)
	}

	fmt.Println(f)
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	os.Exit(1)
}
