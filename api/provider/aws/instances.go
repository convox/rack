package aws

import (
	"os"
	"strings"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/api/structs"
)

func (p *AWSProvider) InstanceList() (structs.Instances, error) {
	res, err := p.listContainerInstances(&ecs.ListContainerInstancesInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
	})

	if err != nil {
		return nil, err
	}

	ecsRes, err := p.ecs().DescribeContainerInstances(
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

	ec2Res, err := p.ec2().DescribeInstances(&ec2.DescribeInstancesInput{
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

	var instances structs.Instances

	for _, i := range ecsRes.ContainerInstances {
		// figure out the CPU and memory metrics
		var cpu, memory structs.InstanceResource

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
		instance := structs.Instance{
			Cpu:    cpu.PercentUsed(),
			Memory: memory.PercentUsed(),
			Id:     *i.Ec2InstanceId,
		}

		if i.AgentConnected != nil {
			instance.Agent = *i.AgentConnected
		}

		if ec2Instance != nil {
			if ec2Instance.PrivateIpAddress != nil {
				instance.PrivateIp = *ec2Instance.PrivateIpAddress
			}

			if ec2Instance.PublicIpAddress != nil {
				instance.PublicIp = *ec2Instance.PublicIpAddress
			}

			if ec2Instance.LaunchTime != nil {
				instance.Started = *ec2Instance.LaunchTime
			}
		}

		if i.RunningTasksCount != nil {
			instance.Processes = int(*i.RunningTasksCount)
		}

		if i.Status != nil {
			instance.Status = strings.ToLower(*i.Status)
		}

		instances = append(instances, instance)
	}

	return instances, nil
}
