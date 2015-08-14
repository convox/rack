package models

import (
	"fmt"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ec2"
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

	asg := resources[fmt.Sprintf("%sInstances", UpperName(process))].Id

	req := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{Name: aws.String("instance-state-name"), Values: []*string{aws.String("running"), aws.String("pending")}},
			&ec2.Filter{Name: aws.String("tag:aws:autoscaling:groupName"), Values: []*string{aws.String(asg)}},
		},
	}

	res, err := EC2().DescribeInstances(req)

	if err != nil {
		return nil, err
	}

	instances := make(Instances, 0)

	for _, r := range res.Reservations {
		for _, i := range r.Instances {
			instances = append(instances, Instance{
				Id:      *i.InstanceID,
				State:   *i.State.Name,
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
