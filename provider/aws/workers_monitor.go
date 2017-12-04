package aws

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/convox/rack/helpers"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/convox/logger"
)

type instance struct {
	Id string

	ASG    bool
	Check  bool
	Docker bool
	ECS    bool

	Unhealthy bool
}

type instances map[string]instance

var lastASGActivity = time.Now()

func (p *AWSProvider) workerMonitor() {
	var log = logger.New("ns=workers.monitor")

	defer recoverWith(func(err error) {
		helpers.Error(log, err)
	})

	disconnectedInstances := map[string]struct{}{}
	for range time.Tick(5 * time.Minute) {
		log.Logf("tick")

		ii := instances{}

		if err := p.describeASG(&ii); err != nil {
			log.Error(err)
			continue
		}

		if err := p.describeECS(&ii); err != nil {
			log.Error(err)
			continue
		}

		// Test if ASG Instance is registered and connected in ECS cluster
		for k, i := range ii {
			if !i.ASG {
				// TODO: Rogue instance?! Terminate?
				continue
			}

			if !i.ECS {
				_, seenBefore := disconnectedInstances[i.Id]

				if !seenBefore {
					disconnectedInstances[i.Id] = struct{}{}
					fmt.Printf("who=\"convox/monitor\" what=\"instance %s missed it's first heartbeat\" why=\"ECS reported agent disconnected\"\n", i.Id)
					continue
				}
				// Not registered or not connected => set Unhealthy
				_, err := p.autoscaling().SetInstanceHealth(
					&autoscaling.SetInstanceHealthInput{
						HealthStatus:             aws.String("Unhealthy"),
						InstanceId:               aws.String(i.Id),
						ShouldRespectGracePeriod: aws.Bool(true),
					},
				)

				i.Unhealthy = true
				ii[k] = i

				if err != nil {
					log.Error(err)
					continue
				}

				// log for humans
				fmt.Printf("who=\"convox/monitor\" what=\"marked instance %s unhealthy\" why=\"ECS reported agent disconnected\"\n", i.Id)
			}
			delete(disconnectedInstances, i.Id)
		}

		log.Logf(ii.log())
	}
}

func (p *AWSProvider) describeASG(ii *instances) error {
	ires, err := p.rackResource("Instances")
	if err != nil {
		return err
	}

	res, err := p.autoscaling().DescribeAutoScalingGroups(
		&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{aws.String(ires)},
		},
	)
	if err != nil {
		return err
	}

	for _, i := range res.AutoScalingGroups[0].Instances {
		instance := (*ii)[*i.InstanceId]

		instance.Id = *i.InstanceId
		instance.ASG = *i.LifecycleState == "InService"

		(*ii)[*i.InstanceId] = instance
	}

	// describe and log every recent ASG activity
	dres, err := p.autoscaling().DescribeScalingActivities(
		&autoscaling.DescribeScalingActivitiesInput{
			AutoScalingGroupName: aws.String(ires),
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

func (p *AWSProvider) describeECS(ii *instances) error {
	dres, err := p.listAndDescribeContainerInstances()
	if err != nil {
		return err
	}

	for _, i := range dres.ContainerInstances {
		instance := (*ii)[*i.Ec2InstanceId]

		instance.Id = *i.Ec2InstanceId
		instance.ECS = *i.AgentConnected

		(*ii)[*i.Ec2InstanceId] = instance
	}

	return nil
}

func (ii instances) log() string {
	var asgIds, ecsIds, unhealthyIds []string

	for _, i := range ii {
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
		len(ii),
		strings.Join(ecsIds, ","),
		strings.Join(asgIds, ","),
		strings.Join(unhealthyIds, ","),
	)
}
