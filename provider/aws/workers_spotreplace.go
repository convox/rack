package aws

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/convox/logger"
)

var (
	spotInstancesEnabled = (os.Getenv("SPOT_INSTANCES") == "true")
	spotTick             = 60 * time.Second
)

// Main worker function
func (p *AWSProvider) workerSpotReplace() {
	log := logger.New("ns=workers.spotreplace").At("spotReplace")

	if !spotInstancesEnabled {
		return
	}

	tick := time.Tick(spotTick)

	for range tick {
		if err := p.spotReplace(); err != nil {
			fmt.Printf("err = %+v\n", err)
			log.Error(err)
		}
	}
}

func (p *AWSProvider) spotReplace() error {
	log := logger.New("ns=workers.spotreplace").At("spotReplace")

	system, err := p.SystemGet()
	if err != nil {
		return err
	}

	log.Logf("status=%q", system.Status)

	// only modify spot instances when running or converging
	switch system.Status {
	case "running", "converging":
	default:
		return nil
	}

	ics, err := p.stackParameter(os.Getenv("RACK"), "InstanceCount")
	if err != nil {
		return err
	}

	ic, err := strconv.Atoi(ics)
	if err != nil {
		return err
	}

	odmin, err := strconv.Atoi(os.Getenv("ON_DEMAND_MIN_COUNT"))
	if err != nil {
		return err
	}

	odc, err := p.asgResourceInstanceCount("Instances")
	if err != nil {
		return err
	}

	spc, err := p.asgResourceInstanceCount("SpotInstances")
	if err != nil {
		return err
	}

	log.Logf("instanceCount=%d onDemandMin=%d onDemandCount=%d spotCount=%d", ic, odmin, odc, spc)

	spotDesired := ic - odmin
	onDemandDesired := ic - spc

	if spc != spotDesired {
		log.Logf("stack=SpotInstances setDesiredCount=%d", spotDesired)

		if err := p.setAsgResourceDesiredCount("SpotInstances", spotDesired); err != nil {
			return err
		}
	}

	if odc != onDemandDesired {
		log.Logf("stack=Instances setDesiredCount=%d", onDemandDesired)

		if err := p.setAsgResourceDesiredCount("Instances", onDemandDesired); err != nil {
			return err
		}
	}

	return nil
}

func (p *AWSProvider) asgResourceInstanceCount(resource string) (int, error) {
	asg, err := p.stackResource(p.Rack, resource)
	if err != nil {
		return 0, err
	}

	res, err := p.autoscaling().DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{asg.PhysicalResourceId},
	})
	if err != nil {
		return 0, err
	}
	if len(res.AutoScalingGroups) < 1 {
		return 0, fmt.Errorf("resource not found: %s", resource)
	}

	count := 0

	for _, ii := range res.AutoScalingGroups[0].Instances {
		if *ii.LifecycleState == "InService" && *ii.HealthStatus == "Healthy" {
			count++
		}
	}

	return count, nil
}

func (p *AWSProvider) setAsgResourceDesiredCount(resource string, count int) error {
	asg, err := p.stackResource(p.Rack, resource)
	if err != nil {
		return err
	}

	_, err = p.autoscaling().SetDesiredCapacity(&autoscaling.SetDesiredCapacityInput{
		AutoScalingGroupName: asg.PhysicalResourceId,
		DesiredCapacity:      aws.Int64(int64(count)),
	})
	if err != nil {
		return err
	}

	return nil
}
