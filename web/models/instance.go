package models

import (
	"fmt"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/ec2"
)

type Instance struct {
	Id    string
	State string

	App     string
	Process string
}

type Instances []Instance

func ListInstances(app, process string) (Instances, error) {
	resources, err := ListResources(app)

	if err != nil {
		panic(err)
	}

	asg := resources[fmt.Sprintf("%sInstances", upperName(process))].PhysicalId
	fmt.Printf("asg %+v\n", asg)

	filter := ec2.NewFilter()
	filter.Add("instance-state-name", "running")
	filter.Add("instance-state-name", "pending")
	filter.Add("tag:aws:autoscaling:groupName", asg)

	res, err := EC2.DescribeInstances(nil, filter)

	if err != nil {
		return nil, err
	}

	instances := make(Instances, 0)

	for _, r := range res.Reservations {
		for _, i := range r.Instances {
			instances = append(instances, Instance{
				Id:      i.InstanceId,
				State:   i.State.Name,
				App:     app,
				Process: process,
			})
		}
	}

	return instances, nil
}

func (i *Instance) Metrics() *Metrics {
	metrics, err := InstanceMetrics(i.App, i.Process, i.Id)

	if err != nil {
		panic(err)
	}

	return metrics
}
