package main

import (
	"fmt"
	"os"

	"github.com/convox/changes"
)

func main() {
	ch := make(chan changes.Change)

	for _, watch := range os.Args[1:] {
		changes.Watch(watch, ch, changes.WatchOptions{})
	}

	for c := range ch {
		fmt.Printf("%s|%s|%s\n", c.Operation, c.Base, c.Path)
	}
}
