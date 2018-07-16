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
	a, err := api.New()
	if err != nil {
		return err
	}

	a.Password = os.Getenv("PASSWORD")

	return a.Listen("https", ":5443")
}
