package workers

import (
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
	log := logger.New("ns=workers.autoscale").At("autoscaleRack")

	capacity, err := models.Provider().CapacityGet()
	if err != nil {
		log.Error(err)
		return
	}

	log.Log("autoscale=%t", autoscale)
	if !autoscale {
		return
	}

	system, err := models.Provider().SystemGet()
	if err != nil {
		log.Error(err)
		return
	}

	log.Log("status=%q", system.Status)
	if system.Status != "running" {
		return
	}

	// calculate instance requirements based on total process memory needed divided by the memory
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

	// if no change then exit
	if system.Count == instances {
		return
	}

	log.Log("process.memory=%d instance.memory=%d instances=%d change=%d", capacity.ProcessMemory, capacity.InstanceMemory, instances, (instances - system.Count))

	system.Count = instances

	err = models.Provider().SystemSave(*system)
	if err != nil {
		log.Error(err)
		return
	}
}
