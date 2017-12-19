package aws

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/convox/logger"
	"github.com/convox/rack/structs"
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

	// extra capacity for autoscale
	extra := 1
	if v := os.Getenv("AUTOSCALE_EXTRA"); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			log.Error(err)
			return
		}
		extra = i
	}
	log = log.Replace("extra", fmt.Sprintf("%d", extra))

	// need minimum 3 instances
	desired := 3

	// instance count must be at least max concurrency plus extra
	if c := int(capacity.ProcessWidth) + extra; c > desired {
		log = log.Replace("reason", "width")
		desired = c
	}

	// calculate instances required to statisfy cpu reservations plus extra
	if c := int(math.Ceil(float64(capacity.ProcessCPU)/float64(capacity.InstanceCPU))) + extra; c > desired {
		log = log.Replace("reason", "cpu")
		desired = c
	}

	// calculate instances required to statisfy memory reservations plus extra
	if c := int(math.Ceil(float64(capacity.ProcessMemory)/float64(capacity.InstanceMemory))) + extra; c > desired {
		log = log.Replace("reason", "memory")
		desired = c
	}

	target := desired

	// only stop one instance at a time
	if target < system.Count {
		target = system.Count - 1
	}

	log.Logf("desired=%d target=%d change=%d", desired, target, (target - system.Count))

	// nothing changed, return
	if target == system.Count {
		return
	}

	if err := p.SystemUpdate(structs.SystemUpdateOptions{InstanceCount: target}); err != nil {
		log.Error(err)
		return
	}
}
