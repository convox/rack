package aws

import (
	"math"
	"os"
	"time"

	"github.com/convox/logger"
)

var (
	autoscale = (os.Getenv("AUTOSCALE") == "true")
	tick      = 1 * time.Minute
)

func (p *AWSProvider) workerAutoscale() {
	p.autoscaleRack()

	for range time.Tick(tick) {
		p.autoscaleRack()
	}
}

func (p *AWSProvider) autoscaleRack() {
	log := logger.New("ns=workers.autoscale").At("autoscaleRack")

	// do nothing unless autoscaling is on
	if !autoscale {
		return
	}

	capacity, err := p.CapacityGet()
	if err != nil {
		log.Error(err)
		return
	}

	system, err := p.SystemGet()
	if err != nil {
		log.Error(err)
		return
	}

	log.Logf("status=%q", system.Status)

	// only allow running and converging status through
	switch system.Status {
	case "running", "converging":
	default:
		return
	}

	// start with the current count
	desired := 0

	// calculate instances required to statisfy cpu reservations plus one for breathing room
	if c := int(math.Ceil(float64(capacity.ProcessCPU)/float64(capacity.InstanceCPU))) + 1; c > desired {
		log = log.Replace("reason", "cpu")
		desired = c
	}

	// calculate instances required to statisfy memory reservations plus one for breathing room
	if c := int(math.Ceil(float64(capacity.ProcessMemory)/float64(capacity.InstanceMemory))) + 1; c > desired {
		log = log.Replace("reason", "memory")
		desired = c
	}

	// instance count cant be less than 2
	if desired < 2 {
		log = log.Replace("reason", "minimum")
		desired = 2
	}

	// instance count must be at least maxconcurrency+1
	if c := int(capacity.ProcessWidth) + 1; c > desired {
		log = log.Replace("reason", "width")
		desired = c
	}

	// if no change then exit
	if system.Count == desired {
		return
	}

	oldCount := system.Count

	// ok to start multiple instances in one pass
	// when shutting down go one at a time but only if current status is "running"
	if desired < system.Count {
		if system.Status == "running" {
			system.Count--
		}
	} else {
		system.Count = desired
	}

	log.Logf("change=%d", (system.Count - oldCount))
	// nothing changed, return
	if system.Count == oldCount {
		return
	}

	err = p.SystemSave(*system)
	if err != nil {
		log.Error(err)
		return
	}
}
