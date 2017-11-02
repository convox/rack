package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/convox/rack/api/models"
	"github.com/convox/rack/manifest1"
)

func init() {
	manifest1.ManifestRandomPorts = false
}

func main() {
	if len(os.Args) < 2 {
		die(fmt.Errorf("usage: fixture <docker-compose.yml>"))
	}

	os.Setenv("REGION", "test")

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
			"Rack":   "convox-test",
		},
	}

	m, err := manifest1.Load(data)
	if err != nil {
		die(err)
	}

	f, err := app.Formation(*m)
	if err != nil {
		die(err)
	}

	pretty, err := models.PrettyJSON(f)
	if err != nil {
		die(err)
	}

	fmt.Println(pretty)
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	os.Exit(1)
}
