package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/convox/rack/api/cmd/build/source"
)

func die(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	os.Exit(1)
}

var (
	app      string
	auth     string
	id       string
	manifest string
	method   string
	registry string
	src      string
)

func init() {
	flag.StringVar(&app, "app", "example", "app name")
	flag.StringVar(&auth, "auth", "", "docker auth data (base64 encoded)")
	flag.StringVar(&id, "id", "latest", "build id")
	flag.StringVar(&manifest, "manifest", "docker-compose.yml", "manifest file")
	flag.StringVar(&method, "method", "", "source method")
	flag.StringVar(&registry, "registry", "", "push to registry")
	flag.StringVar(&src, "source", "", "source location")
}

func main() {
	flag.Parse()

	dir, err := fetch()
	if err != nil {
		die(err)
	}

	defer os.RemoveAll(dir)

	if err := build(dir); err != nil {
		die(err)
	}
}

func fetch() (string, error) {
	var s source.Source

	switch method {
	case "tgz":
		s = &source.SourceTgz{src}
	default:
		die(fmt.Errorf("unknown method: %s", method))
	}

	return s.Fetch()
}

func build(dir string) error {
	dcy := filepath.Join(dir, "docker-compose.yml")

	if _, err := os.Stat(dcy); os.IsNotExist(err) {
		return fmt.Errorf("no docker-compose.yml found")
	}

	data, err := ioutil.ReadFile(dcy)
	if err != nil {
		return err
	}

	m, err := manifest.Load(data)
	if err != nil {
		return err
	}

	s := make(chan string)

	go func() {
		for l := range s {
			fmt.Println(l)
		}
	}()

	if err := m.Build(dir, app, s, true); err != nil {
		return err
	}

	if err := m.Push(s, app, registry, id, ""); err != nil {
		return err
	}

	return nil
}
