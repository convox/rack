package main

import (
	"github.com/convox/rack/provider"
	"github.com/convox/rack/structs"
)

var (
	Provider = provider.FromEnv()
)

func main() {
	Provider.Initialize(structs.ProviderOptions{})

	go Provider.Workers()

	select {}
}
