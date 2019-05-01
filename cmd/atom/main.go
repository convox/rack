package main

import (
	"fmt"
	"os"

	"github.com/convox/rack/pkg/atom"
	"k8s.io/client-go/rest"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}
}

func run() error {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	ac, err := atom.NewController(cfg)
	if err != nil {
		return err
	}

	ac.Run()

	return nil
}
