package aws

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/logger"
)

var (
	spotTick = 60 * time.Second
)

// Main worker function
func (p *Provider) workerSpotReplace() {
	log := logger.New("ns=workers.spotreplace").At("spotReplace")

	if !p.SpotInstances {
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

func (p *Provider) spotReplace() error {
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

	ics, err := p.stackParameter(p.Rack, "InstanceCount")
	if err != nil {
		return err
	}

	ic, err := strconv.Atoi(ics)
	if err != nil {
		return err
	}

	odmin := p.OnDemandMinCount

	odc, err := p.asgResourceInstanceCount("Instances")
	if err != nil {
		return err
	}

	spc, err := p.asgResourceInstanceCountRunning("SpotInstances")
	if err != nil {
		return err
	}

	log.Logf("instanceCount=%d onDemandMin=%d onDemandCount=%d spotCount=%d", ic, odmin, odc, spc)

	spotDesired := ic - odmin

	if spc != spotDesired {
		log.Logf("stack=SpotInstances setDesiredCount=%d", spotDesired)

		if err := p.setAsgResourceDesiredCount("SpotInstances", spotDesired); err != nil {
			return err
		}
	}

	onDemandDesired := ic - spc

	if odc != onDemandDesired {
		log.Logf("stack=Instances setDesiredCount=%d", onDemandDesired)

		if err := p.setAsgResourceDesiredCount("Instances", onDemandDesired); err != nil {
			return err
		}
	}

	return nil
}

func (p *Provider) asgResourceInstanceCount(resource string) (int, error) {
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

	return int(*res.AutoScalingGroups[0].DesiredCapacity), nil
}

func (p *Provider) asgResourceInstanceCountRunning(resource string) (int, error) {
	asg, err := p.stackResource(p.Rack, resource)
	if err != nil {
		return 0, err
	}

	res, err := p.ec2().DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("instance-state-name"),
				Values: []*string{aws.String("running")},
			},
			&ec2.Filter{
				Name:   aws.String("tag:aws:autoscaling:groupName"),
				Values: []*string{asg.PhysicalResourceId},
			},
		},
	})
	if err != nil {
		return 0, err
	}

	count := 0

	for _, r := range res.Reservations {
		count += len(r.Instances)
	}

	return count, nil
}

func (p *Provider) setAsgResourceDesiredCount(resource string, count int) error {
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
