package main

import (
	"os"

	"github.com/convox/rack/pkg/cli"
)

var (
	version = "dev"
)

func main() {
	c := cli.New("convox2", version)

	os.Exit(c.Execute(os.Args[1:]))
}
