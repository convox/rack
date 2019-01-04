package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/convox/rack/pkg/router"
	"k8s.io/apimachinery/pkg/util/runtime"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}
}

func run() error {
	// hack to make glog stop complaining about flag parsing
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	_ = fs.Parse([]string{})
	flag.CommandLine = fs
	runtime.ErrorHandlers = []func(error){}

	r, err := router.New()
	if err != nil {
		return err
	}

	return r.Serve()
}
