package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/api/models"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/ddollar/logger"
)

func startClusterMonitor() {
	var log = logger.New("ns=cluster_monitor")

	defer recoverWith(func(err error) {
		helpers.Error(log, err)
	})

Tick:
	for _ = range time.Tick(5 * time.Minute) {
		log.Log("tick")

		// Ger Rack InstanceCount Parameter
		instanceCount := 0
		instanceType := "unknown"

		res, err := models.CloudFormation().DescribeStacks(
			&cloudformation.DescribeStacksInput{
				StackName: aws.String(os.Getenv("RACK")),
			},
		)

		if err != nil {
			log.Error(err)
			continue
		}

		for _, p := range res.Stacks[0].Parameters {
			if *p.ParameterKey == "InstanceCount" {
				c, err := strconv.Atoi(*p.ParameterValue)

				if err != nil {
					log.Error(err)
					break Tick
				}

				instanceCount = c
			}

			if *p.ParameterKey == "InstanceType" {
				instanceType = *p.ParameterValue
			}
		}

		helpers.SendMixpanelEvent("kernel-cluster-monitor", fmt.Sprintf("count=%d type=%s", instanceCount, instanceType))

		// List and Describe ECS Container Instances
		ires, err := models.ECS().ListContainerInstances(
			&ecs.ListContainerInstancesInput{
				Cluster: aws.String(os.Getenv("CLUSTER")),
			},
		)

		if err != nil {
			log.Error(err)
			continue
		}

		dres, err := models.ECS().DescribeContainerInstances(
			&ecs.DescribeContainerInstancesInput{
				Cluster:            aws.String(os.Getenv("CLUSTER")),
				ContainerInstances: ires.ContainerInstanceArns,
			},
		)

		if err != nil {
			log.Error(err)
			continue
		}

		cInstanceIds := make([]string, 0)
		cInstanceConnections := make(map[string]bool)

		for _, i := range dres.ContainerInstances {
			cInstanceConnections[*i.Ec2InstanceId] = *i.AgentConnected

			if *i.AgentConnected {
				cInstanceIds = append(cInstanceIds, *i.Ec2InstanceId)
			}
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

		if len(uInstanceIds) > 0 {
			helpers.SendMixpanelEvent("kernel-cluster-monitor-mark", strings.Join(uInstanceIds, ","))
		}

		log.Log("InstanceCount=%v connected='%v' healthy='%v' marked='%s'", instanceCount, strings.Join(cInstanceIds, ","), strings.Join(aInstanceIds, ","), strings.Join(uInstanceIds, ","))
	}
}
