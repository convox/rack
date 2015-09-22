package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/convox/rack/api/models"
)

func die(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err.Error())
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: service <type>\n")
		os.Exit(1)
	}

	service := models.Service{Type: os.Args[1]}

	out, err := service.Formation()

	if err != nil && strings.HasSuffix(err.Error(), "not found") {
		die(fmt.Errorf("no such service type: %s", service.Type))
	}

	if err != nil {
		die(err)
	}

	fmt.Println(out)
}
