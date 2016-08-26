package main

import (
	"fmt"

	"github.com/convox/rack/api/models"
)

func recoverWith(f func(err error)) {
	if r := recover(); r != nil {
		if err, ok := r.(error); ok {
			f(err)
		} else {
			f(fmt.Errorf("%v", r))
		}
	}
}

func main() {
	// prime the API and instance for builds by logging into private registries
	// and pulling down latest app images
	go func() {
		models.LoginPrivateRegistries()
		models.PullAppImages()
	}()

	startWeb()
}
