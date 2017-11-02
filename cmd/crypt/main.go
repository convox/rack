package main

import (
	"os"
)

func main() {
	app := NewCli()

	app.Usage = "env management"

	app.Run(os.Args)
}
