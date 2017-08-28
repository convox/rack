package workers

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/cloudflare/cfssl/log"
	"github.com/convox/logger"
	"github.com/convox/rack/api/models"
)

var (
	spotInstancesEnabled = (os.Getenv("SPOT_INSTANCES") == "true")
	spotTick             = 5 * time.Second
)

// Main worker function
func StartSpotReplace() {
	if !spotInstancesEnabled {
		return
	}

	tick := time.Tick(spotTick)

	for range tick {
		if err := spotReplace(); err != nil {
			fmt.Printf("err = %+v\n", err)
			log.Error(err)
		}
	}
}

func spotReplace() error {
	log := logger.New("ns=workers.spotreplace").At("spotReplace")

	system, err := models.Provider().SystemGet()
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

	ics, err := stackParameter(os.Getenv("RACK"), "InstanceCount")
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

	spc, err := asgResourceInstanceCount("SpotInstances")
	if err != nil {
		return err
	}

	spotDesired := ic - odmin
	onDemandDesired := ic - spc

	log.Logf("stack=SpotInstances setDesiredCount=%d", spotDesired)

	if err := setAsgResourceDesiredCount("SpotInstances", spotDesired); err != nil {
		return err
	}

	log.Logf("stack=Instances setDesiredCount=%d", onDemandDesired)

	if err := setAsgResourceDesiredCount("Instances", onDemandDesired); err != nil {
		return err
	}

	return nil
}

func asgResourceInstanceCount(resource string) (int, error) {
	rs, err := models.ListResources(os.Getenv("RACK"))
	if err != nil {
		return 0, err
	}

	res, err := models.AutoScaling().DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{aws.String(rs[resource].Id)},
	})
	if err != nil {
		return 0, err
	}
	if len(res.AutoScalingGroups) < 1 {
		return 0, fmt.Errorf("no such autoscaling resource: %s", resource)
	}

	count := 0

	for _, ii := range res.AutoScalingGroups[0].Instances {
		if *ii.LifecycleState == "InService" && *ii.HealthStatus == "Healthy" {
			count++
		}
	}

	return count, nil
}

func setAsgResourceDesiredCount(resource string, count int) error {
	rs, err := models.ListResources(os.Getenv("RACK"))
	if err != nil {
		return err
	}

	_, err = models.AutoScaling().SetDesiredCapacity(&autoscaling.SetDesiredCapacityInput{
		AutoScalingGroupName: aws.String(rs[resource].Id),
		DesiredCapacity:      aws.Int64(int64(count)),
	})
	if err != nil {
		return err
	}

	return nil
}

func stackParameter(stack, param string) (string, error) {
	res, err := models.DescribeStack(stack)
	if err != nil {
		return "", err
	}
	if len(res.Stacks) < 1 {
		return "", fmt.Errorf("no such stack: %s", stack)
	}

	for _, p := range res.Stacks[0].Parameters {
		if *p.ParameterKey == param {
			return *p.ParameterValue, nil
		}
	}

	return "", fmt.Errorf("no such parameter %s for stack: %s", param, stack)
}
