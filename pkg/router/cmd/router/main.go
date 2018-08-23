package main

import (
	"fmt"
	"os"

	"github.com/convox/rack/pkg/router"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}
}

func run() error {
	r, err := router.New("vlan2", "10.42.0.0/16", "dev")
	if err != nil {
		return err
	}

	return r.Serve()
}
