package main

import (
	"fmt"
	"os"

	"github.com/convox/rack/client"
)

func main() {
	fmt.Printf("konichiwa")

	// report back
	//   manifest
	// success or error
	//
	// or error status / reason

	fmt.Printf("Environ: %+v", os.Environ())

	c := client.New(os.Getenv("RACK_HOST"), os.Getenv("RACK_PASSWORD"), "build")
	_, err := c.UpdateBuild(os.Getenv("APP"), os.Getenv("BUILD"), "web:", "complete", "")
	if err != nil {
		panic(err)
	}
}
