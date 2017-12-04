package main

import "github.com/convox/rack/provider"

var (
	Provider = provider.FromEnv()
)

func main() {
	go Provider.Workers()

	select {}
}
