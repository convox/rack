package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/convox/rack/api/models"
)

func init() {
	models.ManifestRandomPorts = false

}

func main() {
	if len(os.Args) < 2 {
		die(fmt.Errorf("usage: fixture <docker-compose.yml>"))
	}

	data, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		die(err)
	}

	app := &models.App{
		Name: "httpd",
		Tags: map[string]string{
			"Name":   "httpd",
			"Type":   "app",
			"System": "convox",
			"Rack":   "convox",
		},
	}

	m, err := models.LoadManifest(string(data), app)
	if err != nil {
		die(err)
	}

	f, err := m.Formation()
	if err != nil {
		die(err)
	}

	fmt.Println(f)
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	os.Exit(1)
}
