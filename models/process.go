package models

import (
	"fmt"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ecs"
)

type Process struct {
	Name    string
	Command string
	Count   int

	App string
}

type Processes []Process

func ListProcesses(app string) (Processes, error) {
	a, err := GetApp(app)

	if err != nil {
		return nil, err
	}

	req := &ecs.DescribeServicesInput{
		Cluster:  aws.String(a.Cluster),
		Services: []*string{aws.String(app)},
	}

	sres, err := ECS().DescribeServices(req)

	if err != nil {
		return nil, err
	}

	if len(sres.Services) != 1 {
		return nil, fmt.Errorf("could not find service: %s", app)
	}

	service := sres.Services[0]

	tres, err := ECS().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: service.TaskDefinition,
	})

	if err != nil {
		return nil, err
	}

	fmt.Printf("tres.TaskDefinition %+v\n", tres.TaskDefinition)

	ps := Processes{}

	for _, cd := range tres.TaskDefinition.ContainerDefinitions {
		ps = append(ps, Process{
			App:   app,
			Name:  *cd.Name,
			Count: 1,
		})
	}

	return ps, nil
}

func GetProcess(app, name string) (*Process, error) {
	req := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{Name: aws.String("tag:System"), Values: []*string{aws.String("convox")}},
			&ec2.Filter{Name: aws.String("tag:Type"), Values: []*string{aws.String("app")}},
			&ec2.Filter{Name: aws.String("tag:App"), Values: []*string{aws.String(app)}},
		},
	}

	res, err := EC2().DescribeInstances(req)

	if err != nil {
		return nil, err
	}

	count := 0

	for _, r := range res.Reservations {
		count += len(r.Instances)
	}

	process := &Process{
		Name:  name,
		Count: count,
		App:   app,
	}

	return process, nil
}

func (p *Process) SubscribeLogs(output chan []byte, quit chan bool) error {
	resources, err := ListResources(p.App)
	fmt.Printf("err %+v\n", err)

	if err != nil {
		return err
	}

	done := make(chan bool)
	go subscribeKinesis(p.Name, resources[fmt.Sprintf("%sKinesis", upperName(p.Name))].Id, output, done)

	return nil
}

func (p *Process) Instances() Instances {
	instances, err := ListInstances(p.App, p.Name)

	if err != nil {
		panic(err)
	}

	return instances
}

func (p *Process) Metrics() *Metrics {
	metrics, err := ProcessMetrics(p.App, p.Name)

	if err != nil {
		panic(err)
	}

	return metrics
}

func (p *Process) Resources() Resources {
	resources, err := ListProcessResources(p.App, p.Name)

	if err != nil {
		panic(err)
	}

	return resources
}

func (p *Process) Userdata() string {
	return `""`
}
