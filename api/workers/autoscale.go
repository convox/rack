package workers

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/convox/rack/api/models"
	"github.com/ddollar/logger"
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

	capacity, err := models.Provider().CapacityGet()
	if err != nil {
		log.Log("fn=models.GetSystemCapacity err=%q", err)
		return
	}

	log.Log("autoscale=%t", autoscale)

	if !autoscale {
		return
	}

	system, err := models.Provider().SystemGet()
	if err != nil {
		log.Log("fn=models.GetSystem err=%q", err)
		return
	}

	// calaculate instance requirements based on total process memory needed divided by the memory
	// on an individual instance
	instances := int(math.Ceil(float64(capacity.ProcessMemory) / float64(capacity.InstanceMemory)))

	// add one instance for some breathing room
	instances++

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

	err = models.Provider().SystemSave(*system)
	if err != nil {
		log.Log("fn=system.Save err=%q", err)
		return
	}

	// log for humans
	fmt.Printf("who=\"convox/monitor\" what=\"autoscaled instance count to %d\" why=\"a service wants %s processes behind a load balancer\"\n", system.Count)
}
