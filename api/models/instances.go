package models

import (
	"os"
	"strings"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/client"
)

type Instance client.Instance

type InstanceResource struct {
	Total int `json:"total"`
	Free  int `json:"free"`
	Used  int `json:"used"`
}

func (ir InstanceResource) PercentUsed() float64 {
	return float64(ir.Used) / float64(ir.Total)
}

func (s *System) GetInstances() ([]*Instance, error) {
	res, err := ECS().ListContainerInstances(
		&ecs.ListContainerInstancesInput{
			Cluster: aws.String(os.Getenv("CLUSTER")),
		},
	)

	if err != nil {
		return nil, err
	}

	ecsRes, err := ECS().DescribeContainerInstances(
		&ecs.DescribeContainerInstancesInput{
			Cluster:            aws.String(os.Getenv("CLUSTER")),
			ContainerInstances: res.ContainerInstanceArns,
		},
	)

	if err != nil {
		return nil, err
	}

	var instanceIds []*string
	for _, i := range ecsRes.ContainerInstances {
		instanceIds = append(instanceIds, i.Ec2InstanceId)
	}

	ec2Res, err := EC2().DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{Name: aws.String("instance-id"), Values: instanceIds},
		},
	})

	if err != nil {
		return nil, err
	}

	ec2Instances := make(map[string]*ec2.Instance)
	for _, r := range ec2Res.Reservations {
		for _, i := range r.Instances {
			ec2Instances[*i.InstanceId] = i
		}
	}

	var instances []*Instance
	for _, i := range ecsRes.ContainerInstances {
		// figure out the CPU and memory metrics
		var cpu, memory InstanceResource

		for _, r := range i.RegisteredResources {
			switch *r.Name {
			case "CPU":
				cpu.Total = int(*r.IntegerValue)
			case "MEMORY":
				memory.Total = int(*r.IntegerValue)
			}
		}

		for _, r := range i.RemainingResources {
			switch *r.Name {
			case "CPU":
				cpu.Free = int(*r.IntegerValue)
				cpu.Used = cpu.Total - cpu.Free
			case "MEMORY":
				memory.Free = int(*r.IntegerValue)
				memory.Used = memory.Total - memory.Free
			}
		}

		// find the matching Instance from the EC2 response
		ec2Instance := ec2Instances[*i.Ec2InstanceId]

		// build up the struct
		instance := &Instance{
			Agent:   *i.AgentConnected,
			Cpu:     truncate(cpu.PercentUsed(), 4),
			Memory:  truncate(memory.PercentUsed(), 4),
			Id:      *i.Ec2InstanceId,
			Ip:      *ec2Instance.PublicIpAddress,
			Pending: int(*i.PendingTasksCount),
			Running: int(*i.RunningTasksCount),
			Status:  strings.ToLower(*i.Status),
		}

		instances = append(instances, instance)
	}

	return instances, nil
}
