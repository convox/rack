package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/convox/rack/api/models"
)

func die(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err.Error())
	os.Exit(1)
}

func main() {
	var manifest models.Manifest

	if stat, _ := os.Stdin.Stat(); stat.Mode()&os.ModeCharDevice == 0 {
		data, err := ioutil.ReadAll(os.Stdin)

		if err != nil {
			die(err)
		}

		manifest, err = models.LoadManifest(string(data), true)

		if err != nil {
			die(err)
		}
	}

	out, err := manifest.Formation()

	if err != nil {
		die(err)
	}

	fmt.Println(out)
}
