package main

import (
	"github.com/convox/rack/provider"
	"github.com/convox/rack/structs"
)

var (
	Provider structs.Provider
)

func init() {
	p, err := provider.FromEnv()
	if err != nil {
		panic(err)
	}
	Provider = p
}

func main() {
	Provider.Initialize(structs.ProviderOptions{})

	go Provider.Workers()

	select {}
}
