package aws

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/api/structs"
)

func (p *AWSProvider) InstanceList() (structs.Instances, error) {
	ihash := map[string]structs.Instance{}

	req := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("tag:Rack"), Values: []*string{aws.String(os.Getenv("RACK"))}},
			{Name: aws.String("tag:aws:cloudformation:logical-id"), Values: []*string{aws.String("Instances")}},
		},
	}

	err := p.ec2().DescribeInstancesPages(req, func(res *ec2.DescribeInstancesOutput, last bool) bool {
		for _, r := range res.Reservations {
			for _, i := range r.Instances {
				ihash[cs(i.InstanceId, "")] = structs.Instance{
					Id:        cs(i.InstanceId, ""),
					PrivateIp: cs(i.PrivateIpAddress, ""),
					PublicIp:  cs(i.PublicIpAddress, ""),
					Status:    "",
					Started:   ct(i.LaunchTime),
				}
			}
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	cis, err := p.describeContainerInstances()
	if err != nil {
		return nil, err
	}

	for _, cci := range cis.ContainerInstances {
		id := cs(cci.Ec2InstanceId, "")
		i := ihash[id]

		i.Agent = cb(cci.AgentConnected, false)
		i.Processes = int(ci(cci.RunningTasksCount, 0))
		i.Status = strings.ToLower(cs(cci.Status, "unknown"))

		var cpu, memory structs.InstanceResource

		for _, r := range cci.RegisteredResources {
			switch *r.Name {
			case "CPU":
				cpu.Total = int(ci(r.IntegerValue, 0))
			case "MEMORY":
				memory.Total = int(ci(r.IntegerValue, 0))
			}
		}

		for _, r := range cci.RemainingResources {
			switch *r.Name {
			case "CPU":
				cpu.Free = int(ci(r.IntegerValue, 0))
				cpu.Used = cpu.Total - cpu.Free
			case "MEMORY":
				memory.Free = int(ci(r.IntegerValue, 0))
				memory.Used = memory.Total - memory.Free
			}
		}

		i.Cpu = cpu.PercentUsed()
		i.Memory = memory.PercentUsed()

		ihash[id] = i
	}

	instances := structs.Instances{}

	for _, v := range ihash {
		instances = append(instances, v)
	}

	sort.Sort(instances)

	return instances, nil
}

func (p *AWSProvider) InstanceTerminate(id string) error {
	instances, err := p.InstanceList()
	if err != nil {
		return err
	}

	found := false

	for _, i := range instances {
		if i.Id == id {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("no such instance: %s", id)
	}

	_, err = p.autoscaling().TerminateInstanceInAutoScalingGroup(&autoscaling.TerminateInstanceInAutoScalingGroupInput{
		InstanceId:                     aws.String(id),
		ShouldDecrementDesiredCapacity: aws.Bool(false),
	})
	if err != nil {
		return err
	}

	return nil
}

// describeContainerInstances lists and describes all the ECS instances.
// It handles pagination for clusters > 100 instances.
func (p *AWSProvider) describeContainerInstances() (*ecs.DescribeContainerInstancesOutput, error) {
	instances := []*ecs.ContainerInstance{}
	var nextToken string

	for {
		res, err := p.listContainerInstances(&ecs.ListContainerInstancesInput{
			Cluster:   aws.String(p.Cluster),
			NextToken: &nextToken,
		})
		if ae, ok := err.(awserr.Error); ok && ae.Code() == "ClusterNotFoundException" {
			return nil, errorNotFound(fmt.Sprintf("cluster not found: %s", p.Cluster))
		}
		if err != nil {
			return nil, err
		}

		dres, err := p.ecs().DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
			Cluster:            aws.String(p.Cluster),
			ContainerInstances: res.ContainerInstanceArns,
		})
		if err != nil {
			return nil, err
		}

		instances = append(instances, dres.ContainerInstances...)

		// No more container results
		if res.NextToken == nil {
			break
		}

		// set the nextToken to be used for the next iteration
		nextToken = *res.NextToken
	}

	return &ecs.DescribeContainerInstancesOutput{
		ContainerInstances: instances,
	}, nil
}
