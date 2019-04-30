package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/convox/rack/provider/local"
)

type templater interface {
	SystemTemplate(string) ([]byte, error)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}
}

func run() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: system-template <provider> <version>")
	}

	switch os.Args[1] {
	case "local":
		p, err := local.FromEnv()
		if err != nil {
			return err
		}
		return template(p, os.Args[2])
	default:
		return fmt.Errorf("unknown provider: %s", os.Args[1])
	}

	return nil
}

func template(t templater, version string) error {
	data, err := t.SystemTemplate(version)
	if err != nil {
		return err
	}

	fmt.Println(strings.TrimSpace(string(data)))

	return nil
}
