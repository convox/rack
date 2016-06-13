package workers

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/api/models"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/ddollar/logger"
)

type Instance struct {
	Id string

	ASG    bool
	Check  bool
	Docker bool
	ECS    bool

	Unhealthy bool
}

type Instances map[string]Instance

var lastASGActivity = time.Now()

func StartCluster() {
	var log = logger.New("ns=cluster_monitor")

	defer recoverWith(func(err error) {
		helpers.Error(log, err)
	})

	for _ = range time.Tick(5 * time.Minute) {
		log.Log("tick")

		instances := Instances{}

		err := instances.describeASG()
		if err != nil {
			log.Error(err)
			continue
		}

		err = instances.describeECS()
		if err != nil {
			log.Error(err)
			continue
		}

		// Test if ASG Instance is registered and connected in ECS cluster
		for k, i := range instances {
			if !i.ASG {
				// TODO: Rogue instance?! Terminate?
				continue
			}

			if !i.ECS {
				// Not registered or not connected => set Unhealthy
				_, err := models.AutoScaling().SetInstanceHealth(
					&autoscaling.SetInstanceHealthInput{
						HealthStatus:             aws.String("Unhealthy"),
						InstanceId:               aws.String(i.Id),
						ShouldRespectGracePeriod: aws.Bool(true),
					},
				)

				i.Unhealthy = true
				instances[k] = i

				if err != nil {
					log.Error(err)
					continue
				}

				// log for humans
				fmt.Printf("who=\"convox/monitor\" what=\"marked instance %s unhealthy\" why=\"ECS reported agent disconnected\"\n", i.Id)
			}
		}

		log.Log(instances.log())
	}
}

func (instances Instances) describeASG() error {
	resources, err := models.ListResources(os.Getenv("RACK"))

	res, err := models.AutoScaling().DescribeAutoScalingGroups(
		&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{
				aws.String(resources["Instances"].Id),
			},
		},
	)
	if err != nil {
		return err
	}

	for _, i := range res.AutoScalingGroups[0].Instances {
		instance := instances[*i.InstanceId]

		instance.Id = *i.InstanceId
		instance.ASG = *i.LifecycleState == "InService"

		instances[*i.InstanceId] = instance
	}

	// describe and log every recent ASG activity
	dres, err := models.AutoScaling().DescribeScalingActivities(
		&autoscaling.DescribeScalingActivitiesInput{
			AutoScalingGroupName: aws.String(resources["Instances"].Id),
		},
	)
	if err != nil {
		return nil
	}

	for _, a := range dres.Activities {
		if lastASGActivity.Before(*a.StartTime) {
			fmt.Printf("who=\"EC2/ASG\" what=%q why=%q\n", *a.Description, *a.Cause)
			lastASGActivity = *a.StartTime
		}
	}

	return nil
}

func (instances Instances) describeECS() error {
	res, err := models.ECS().ListContainerInstances(
		&ecs.ListContainerInstancesInput{
			Cluster: aws.String(os.Getenv("CLUSTER")),
		},
	)

	if err != nil {
		return err
	}

	dres, err := models.ECS().DescribeContainerInstances(
		&ecs.DescribeContainerInstancesInput{
			Cluster:            aws.String(os.Getenv("CLUSTER")),
			ContainerInstances: res.ContainerInstanceArns,
		},
	)

	if err != nil {
		return err
	}

	for _, i := range dres.ContainerInstances {
		instance := instances[*i.Ec2InstanceId]

		instance.Id = *i.Ec2InstanceId
		instance.ECS = *i.AgentConnected

		instances[*i.Ec2InstanceId] = instance
	}

	return nil
}

func (instances Instances) log() string {
	var asgIds, ecsIds, unhealthyIds []string

	for _, i := range instances {
		if i.ASG {
			asgIds = append(asgIds, i.Id)
		}

		if i.ECS {
			ecsIds = append(ecsIds, i.Id)
		}

		if i.Unhealthy {
			unhealthyIds = append(unhealthyIds, i.Id)
		}
	}

	sort.Strings(asgIds)
	sort.Strings(ecsIds)
	sort.Strings(unhealthyIds)

	return fmt.Sprintf("count=%v connected='%v' healthy='%v' marked='%s'",
		len(instances),
		strings.Join(ecsIds, ","),
		strings.Join(asgIds, ","),
		strings.Join(unhealthyIds, ","),
	)
}
