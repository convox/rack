package main

import (
	"fmt"
	"os"

	"github.com/convox/rack/router"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}
}

func run() error {
	r, err := router.New()
	if err != nil {
		return err
	}

	return r.Serve()
}
