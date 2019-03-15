package main

import (
	"fmt"
	"os"

	"github.com/convox/rack/pkg/generate"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}
}

func run() error {
	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "controllers":
		data, err := generate.Controllers()
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	case "routes":
		data, err := generate.Routes()
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	case "sdk":
		data, err := generate.SDK()
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	default:
		usage()
	}

	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: generate <controllers|routes>\n")
	os.Exit(1)
}
