package main

import (
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/provider"
)

func main() {
	helpers.UseSharedCredentials()

	p, err := provider.FromEnv()
	if err != nil {
		panic(err)
	}

	p.Initialize(structs.ProviderOptions{})

	go p.Workers()

	select {}
}
