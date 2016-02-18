package main

import (
	"fmt"

	"github.com/convox/rack/api/models"
)

func recoverWith(f func(err error)) {
	if r := recover(); r != nil {
		// coerce r to error type
		err, ok := r.(error)
		if !ok {
			err = fmt.Errorf("%v", r)
		}

		f(err)
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
