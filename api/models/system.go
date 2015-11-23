package models

import (
	"fmt"
	"os"
	"strconv"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/client"
)

type System client.System

func GetSystem() (*System, error) {
	rack := os.Getenv("RACK")

	res, err := CloudFormation().DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String(rack)})

	if err != nil {
		return nil, err
	}

	if len(res.Stacks) != 1 {
		return nil, fmt.Errorf("could not load stack for app: %s", rack)
	}

	stack := res.Stacks[0]
	params := stackParameters(stack)

	count, err := strconv.Atoi(params["InstanceCount"])

	if err != nil {
		return nil, err
	}

	r := &System{
		Count:   count,
		Name:    rack,
		Status:  humanStatus(*stack.StackStatus),
		Type:    params["InstanceType"],
		Version: os.Getenv("RELEASE"),
	}

	return r, nil
}

func (s *System) GetInstances() ([]*client.Instance, error) {
	res, err := ECS().ListContainerInstances(
		&ecs.ListContainerInstancesInput{
			Cluster: aws.String(os.Getenv("CLUSTER")),
		},
	)

	if err != nil {
		return nil, err
	}

	dres, err := ECS().DescribeContainerInstances(
		&ecs.DescribeContainerInstancesInput{
			Cluster:            aws.String(os.Getenv("CLUSTER")),
			ContainerInstances: res.ContainerInstanceArns,
		},
	)

	if err != nil {
		return nil, err
	}

	var instances []*client.Instance

	for _, i := range dres.ContainerInstances {
		instance := &client.Instance{
			Agent:   *i.AgentConnected,
			Running: int(*i.RunningTasksCount),
			Pending: int(*i.PendingTasksCount),
			Status:  *i.Status,
			Id:      *i.Ec2InstanceId,
		}

		for _, r := range i.RegisteredResources {
			switch *r.Name {
			case "CPU":
				instance.Cpu.Total = int(*r.IntegerValue)
			case "MEMORY":
				instance.Memory.Total = int(*r.IntegerValue)
			}
		}

		for _, r := range i.RegisteredResources {
			switch *r.Name {
			case "CPU":
				instance.Cpu.Total = int(*r.IntegerValue)
			case "MEMORY":
				instance.Memory.Total = int(*r.IntegerValue)
			}
		}

		for _, r := range i.RemainingResources {
			switch *r.Name {
			case "CPU":
				instance.Cpu.Free = int(*r.IntegerValue)
				instance.Cpu.Used = instance.Cpu.Total - instance.Cpu.Free
			case "MEMORY":
				instance.Memory.Free = int(*r.IntegerValue)
				instance.Memory.Used = instance.Memory.Total - instance.Memory.Free
			}
		}

		instances = append(instances, instance)
	}

	return instances, nil
}

func (r *System) Save() error {
	rack := os.Getenv("RACK")

	app, err := GetApp(rack)

	if err != nil {
		return err
	}

	mac, err := maxAppConcurrency()

	// dont scale the rack below the max concurrency plus one
	// see formation.go for more details
	if err == nil && r.Count < (mac+1) {
		return fmt.Errorf("max process concurrency is %d, can't scale rack below %d instances", mac, mac+1)
	}

	params := map[string]string{
		"InstanceCount": strconv.Itoa(r.Count),
		"InstanceType":  r.Type,
		"Version":       r.Version,
	}

	// Report cluster size change
	helpers.TrackEvent("kernel-cluster-monitor", fmt.Sprintf("count=%d type=%s", r.Count, r.Type))

	template := fmt.Sprintf("https://convox.s3.amazonaws.com/release/%s/formation.json", r.Version)

	return app.UpdateParamsAndTemplate(params, template)
}

func maxAppConcurrency() (int, error) {
	apps, err := ListApps()

	if err != nil {
		return 0, err
	}

	max := 0

	for _, app := range apps {
		rel, err := app.LatestRelease()

		if err != nil {
			return 0, err
		}

		if rel == nil {
			continue
		}

		m, err := LoadManifest(rel.Manifest)

		if err != nil {
			return 0, err
		}

		f, err := ListFormation(app.Name)

		if err != nil {
			return 0, err
		}

		for _, me := range m {
			if len(me.ExternalPorts()) > 0 {
				entry := f.Entry(me.Name)

				if entry != nil && entry.Count > max {
					max = entry.Count
				}
			}
		}
	}

	return max, nil
}
