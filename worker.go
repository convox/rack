package main

import (
	"os"

	"github.com/convox/kernel/formation"
)

func startWorker() {
	if os.Getenv("WORKER") == "true" {
		formation.Listen()
	}
}
