package start

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/convox/rack/pkg/structs"
)

type Options struct {
	App     string
	Build   bool
	Cache   bool
	Command []string
	// Context  *cli.Context
	Id       string
	Manifest string
	Provider structs.Provider
	Services []string
	Shift    int
	Sync     bool
}

func handleInterrupt(fn func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill)
	<-ch
	fmt.Println("")
	fn()
}
