package workers

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/logger"
	"github.com/convox/rack/api/models"
)

var (
	spotreplace = (os.Getenv("SPOT_INSTANCES") == "true")
)

// Main worker function
func StartSpotReplace() {
	spotReplace()

	for range time.Tick(tick) {
		spotReplace()
	}
}

func spotReplace() {
	log := logger.New("ns=workers.spotreplace").At("spotReplace")

	// do nothing unless autoscaling is on
	if !spotreplace {
		return
	}

	// get system
	system, err := models.Provider().SystemGet()
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

	resources, err := models.ListResources(os.Getenv("RACK"))
	if err != nil {
		return
	}

	// get on-demand ASG
	odres, err := models.AutoScaling().DescribeAutoScalingGroups(
		&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{
				aws.String(resources["Instances"].Id),
			},
		},
	)
	if err != nil {
		return
	}

	// count the Healthy, InService on-demand instances
	onDemandCount := 0
	for _, onDemandInstance := range odres.AutoScalingGroups[0].Instances {
		if (*onDemandInstance.HealthStatus == "Healthy") &&
			((*onDemandInstance.LifecycleState == "InService") || (*onDemandInstance.LifecycleState == "Pending")) {
			onDemandCount++
		}
	}

	onDemandDesiredCapacity := *odres.AutoScalingGroups[0].DesiredCapacity

	// get spot ASG
	sres, err := models.AutoScaling().DescribeAutoScalingGroups(
		&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{
				aws.String(resources["SpotInstances"].Id),
			},
		},
	)
	if err != nil {
		return
	}

	// count the Healthy, InService spot instances
	spotCount := 0
	for _, spotInstance := range sres.AutoScalingGroups[0].Instances {
		if (*spotInstance.HealthStatus == "Healthy") && (*spotInstance.LifecycleState == "InService") {
			spotCount++
		}
	}

	spotDesiredCapacity := *sres.AutoScalingGroups[0].DesiredCapacity
	totalInstances := onDemandCount + spotCount

	// get the count of DRAINING instances
	cres, err := models.ECS().ListContainerInstances(
		&ecs.ListContainerInstancesInput{
			Cluster: aws.String(resources["Cluster"].Id),
			Status:  aws.String("DRAINING"),
		},
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Subtract the count of draining instances from the total count
	totalInstances -= len(cres.ContainerInstanceArns)

	// if total instances > than InstanceCount and there are too many spot instances, reduce spot count by 1
	minOnDemandCount := 3
	if os.Getenv("ON_DEMAND_MIN_COUNT") != "" {
		minOnDemandCount, err = strconv.Atoi(os.Getenv("ON_DEMAND_MIN_COUNT"))
		if err != nil {
			fmt.Println(err)
		}
	}
	if (totalInstances > system.Count) && ((system.Count - spotCount) < minOnDemandCount) {
		_, err := models.AutoScaling().SetDesiredCapacity(
			&autoscaling.SetDesiredCapacityInput{
				AutoScalingGroupName: aws.String(resources["SpotInstances"].Id),
				DesiredCapacity:      aws.Int64(spotDesiredCapacity - 1),
			},
		)
		if err != nil {
			fmt.Println(err)
			return
		}
		return
	}

	// if total instances > than InstanceCount, reduce on-demand desired count by 1
	if totalInstances > system.Count {
		_, err := models.AutoScaling().SetDesiredCapacity(
			&autoscaling.SetDesiredCapacityInput{
				AutoScalingGroupName: aws.String(resources["Instances"].Id),
				DesiredCapacity:      aws.Int64(onDemandDesiredCapacity - 1),
			},
		)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	// if total instances < than InstanceCount, increase on-demand desired count by (InstanceCount - total instances)
	if totalInstances < system.Count {
		newInstancesNeeded := int64(system.Count - totalInstances)
		_, err := models.AutoScaling().SetDesiredCapacity(
			&autoscaling.SetDesiredCapacityInput{
				AutoScalingGroupName: aws.String(resources["Instances"].Id),
				DesiredCapacity:      aws.Int64(onDemandDesiredCapacity + newInstancesNeeded),
			},
		)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	return
}
