package main

import (
	"fmt"
	"os"

	"github.com/convox/rack/api"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}
}

func run() error {
	a := api.New()

	a.Password = os.Getenv("PASSWORD")

	return a.Listen("https", ":5443")
}
