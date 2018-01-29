package main

import (
	"fmt"
	"os"

	"github.com/convox/rack/server"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}
}

func run() error {
	return server.Listen(os.Getenv("PROVIDER"), ":5443")
}
