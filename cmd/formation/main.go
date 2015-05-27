package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

func die(err error) {
	fmt.Fprintf(os.Stderr, "error: %s\n", err)
	os.Exit(1)
}

func main() {
	req, err := ioutil.ReadAll(os.Stdin)

	if err != nil {
		die(err)
	}

	fmt.Println("Request:")
	fmt.Println(string(req))
}
