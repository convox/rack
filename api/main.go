package main

import (
	"fmt"

	"github.com/convox/rack/api/workers"
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
	go workers.StartCluster()
	go workers.StartHeartbeat()
	go workers.StartImages()
	go workers.StartServicesCapacity()

	startWeb()
}
