package workers

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/api/models"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/ddollar/logger"
)

func StartCluster() {
	var log = logger.New("ns=cluster_monitor")

	defer recoverWith(func(err error) {
		helpers.Error(log, err)
	})

	for _ = range time.Tick(5 * time.Minute) {
		log.Log("tick")

		instanceCount, err := getRackInstanceCount()

		if err != nil {
			log.Error(err)
			continue
		}

		cInstanceIds, cInstanceConnections, err := describeClusterInstances()

		if err != nil {
			log.Error(err)
			continue
		}

		// Get and Describe Rack ASG Resource
		resources, err := models.ListResources(os.Getenv("RACK"))

		ares, err := models.AutoScaling().DescribeAutoScalingGroups(
			&autoscaling.DescribeAutoScalingGroupsInput{
				AutoScalingGroupNames: []*string{
					aws.String(resources["Instances"].Id),
				},
			},
		)

		if err != nil {
			log.Error(err)
			continue
		}

		// Test if ASG Instance is registered and connected in ECS cluster

		aInstanceIds := []string{}
		uInstanceIds := []string{}

		for _, i := range ares.AutoScalingGroups[0].Instances {
			if connected, exists := cInstanceConnections[*i.InstanceId]; connected && exists {
				aInstanceIds = append(aInstanceIds, *i.InstanceId)
			} else {
				// Not registered or not connected => set Unhealthy
				if *i.LifecycleState == "InService" {
					_, err := models.AutoScaling().SetInstanceHealth(
						&autoscaling.SetInstanceHealthInput{
							HealthStatus:             aws.String("Unhealthy"),
							InstanceId:               aws.String(*i.InstanceId),
							ShouldRespectGracePeriod: aws.Bool(true),
						},
					)

					if err != nil {
						log.Error(err)
						continue
					}

					uInstanceIds = append(uInstanceIds, *i.InstanceId)
				}
			}
		}

		sort.Strings(aInstanceIds)
		sort.Strings(cInstanceIds)
		sort.Strings(uInstanceIds)

		log.Log("InstanceCount=%v connected='%v' healthy='%v' marked='%s'", instanceCount, strings.Join(cInstanceIds, ","), strings.Join(aInstanceIds, ","), strings.Join(uInstanceIds, ","))
	}
}

func describeClusterInstances() ([]string, map[string]bool, error) {
	ids := make([]string, 0)
	conns := make(map[string]bool)

	// List and Describe ECS Container Instances
	ires, err := models.ECS().ListContainerInstances(
		&ecs.ListContainerInstancesInput{
			Cluster: aws.String(os.Getenv("CLUSTER")),
		},
	)

	if err != nil {
		return ids, conns, err
	}

	dres, err := models.ECS().DescribeContainerInstances(
		&ecs.DescribeContainerInstancesInput{
			Cluster:            aws.String(os.Getenv("CLUSTER")),
			ContainerInstances: ires.ContainerInstanceArns,
		},
	)

	if err != nil {
		return ids, conns, err
	}

	for _, i := range dres.ContainerInstances {
		conns[*i.Ec2InstanceId] = *i.AgentConnected

		if *i.AgentConnected {
			ids = append(ids, *i.Ec2InstanceId)
		}
	}

	return ids, conns, nil
}

func getRackInstanceCount() (int, error) {
	name := os.Getenv("RACK")

	res, err := models.CloudFormation().DescribeStacks(
		&cloudformation.DescribeStacksInput{
			StackName: aws.String(name),
		},
	)

	if err != nil {
		return 0, err
	}

	for _, p := range res.Stacks[0].Parameters {
		if *p.ParameterKey == "InstanceCount" {
			c, err := strconv.Atoi(*p.ParameterValue)

			if err != nil {
				return 0, err
			}

			return c, nil
		}
	}

	return 0, fmt.Errorf("Stack %s InstanceCount parameter missing", name)
}
