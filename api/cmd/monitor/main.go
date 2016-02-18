package main

import (
	"time"

	"github.com/convox/rack/api/workers"
)

func main() {
	go workers.StartAutoscale()
	go workers.StartCluster()
	go workers.StartHeartbeat()
	go workers.StartServicesCapacity()

	for {
		time.Sleep(1 * time.Hour)
	}
}
