package workers

import (
	"math"
	"os"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/rack/api/provider"
)

var (
	autoscale = (os.Getenv("AUTOSCALE") == "true")
	tick      = 1 * time.Minute
)

func StartAutoscale() {
	autoscaleRack()

	for range time.Tick(tick) {
		autoscaleRack()
	}
}

func autoscaleRack() {
	log := logger.New("ns=workers.autoscale at=autoscaleRack")

	capacity, err := provider.CapacityGet()

	if err != nil {
		log.Log("fn=models.GetSystemCapacity err=%q", err)
		return
	}

	log.Log("autoscale=%t", autoscale)

	if !autoscale {
		return
	}

	system, err := provider.SystemGet()

	if err != nil {
		log.Log("fn=models.GetSystem err=%q", err)
		return
	}

	// calaculate instance requirements based on total process memory needed divided by the memory
	// on an individual instance
	instances := int(math.Ceil(float64(capacity.ProcessMemory) / float64(capacity.InstanceMemory)))

	// instance count cant be less than 2
	if instances < 2 {
		instances = 2
	}

	// instance count must be at least maxconcurrency+1
	if instances < (int(capacity.ProcessWidth) + 1) {
		instances = int(capacity.ProcessWidth) + 1
	}

	log.Log("process.memory=%d instance.memory=%d instances=%d change=%d", capacity.ProcessMemory, capacity.InstanceMemory, instances, (instances - system.Count))

	// if no change then exit
	if system.Count == instances {
		return
	}

	system.Count = instances

	err = provider.SystemSave(*system)

	if err != nil {
		log.Log("fn=system.Save err=%q", err)
		return
	}
}
