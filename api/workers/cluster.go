package workers

import (
	"os"
	"sort"
	"strings"
	"time"

	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/api/models"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/ddollar/logger"
)

type Instance struct {
	Id string

	ECS    bool
	ASG    bool
	Docker bool
	Run    bool

	Unhealthy bool
}

type Instances map[string]Instance

func StartCluster() {
	var log = logger.New("ns=cluster_monitor")

	defer recoverWith(func(err error) {
		helpers.Error(log, err)
	})

	for _ = range time.Tick(30 * time.Second) {
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
		for _, i := range instances {
			if i.ASG && !i.ECS {
				// Not registered or not connected => set Unhealthy
				_, err := models.AutoScaling().SetInstanceHealth(
					&autoscaling.SetInstanceHealthInput{
						HealthStatus:             aws.String("Unhealthy"),
						InstanceId:               aws.String(i.Id),
						ShouldRespectGracePeriod: aws.Bool(true),
					},
				)

				i.Unhealthy = true

				if err != nil {
					log.Error(err)
					continue
				}
			}
		}

		log.Log("InstanceCount=%v connected='%v' healthy='%v' marked='%s'", len(instances), strings.Join(instances.inECS(), ","), strings.Join(instances.inASG(), ","), strings.Join(instances.inUnhealthy(), ","))
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

func (instances Instances) inASG() []string {
	ids := []string{}

	for _, i := range instances {
		if i.ASG {
			ids = append(ids, i.Id)
		}
	}

	sort.Strings(ids)

	return ids
}

func (instances Instances) inECS() []string {
	ids := []string{}

	for _, i := range instances {
		if i.ECS {
			ids = append(ids, i.Id)
		}
	}

	sort.Strings(ids)

	return ids
}

func (instances Instances) inUnhealthy() []string {
	ids := []string{}

	for _, i := range instances {
		if i.Unhealthy {
			ids = append(ids, i.Id)
		}
	}

	sort.Strings(ids)

	return ids
}
