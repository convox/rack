package start

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/convox/rack/pkg/structs"
)

type Interface interface {
	Start1(Options) error
	Start2(structs.Provider, Options) error
}

type Options struct {
	App     string
	Build   bool
	Cache   bool
	Command []string
	// Context  *cli.Context
	Id       string
	Manifest string
	Services []string
	Shift    int
	Sync     bool
}

type Start struct{}

func New() Interface {
	return &Start{}
}

func handleInterrupt(fn func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill)
	<-ch
	fmt.Println("")
	fn()
}
