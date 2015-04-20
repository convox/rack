package main

import (
	"os"

	"github.com/convox/cli/stdcli"
)

func main() {
	app := stdcli.New()
	app.Usage = "command-line application management"
	app.Run(os.Args)
}
