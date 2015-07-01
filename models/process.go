package models

import (
	"fmt"
	"os"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ecs"
)

type Process struct {
	Name    string
	Command string
	Count   int

	ServiceType string

	App string
}

type Processes []Process

type ProcessRunOptions struct {
	Command string
}

func ListProcesses(app string) (Processes, error) {
	req := &ecs.DescribeServicesInput{
		Cluster:  aws.String(os.Getenv("CLUSTER")),
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

	ps := Processes{}
	links := make(map[string]string)

	for _, cd := range tres.TaskDefinition.ContainerDefinitions {
		if !strings.HasPrefix(*cd.Name, "convox-") {
			ps = append(ps, Process{
				App:   app,
				Name:  *cd.Name,
				Count: int(*service.DesiredCount),
			})

			for _, l := range cd.Links {
				ls := strings.Split(*l, ":")

				if len(ls) == 2 {
					links[ls[0]] = ls[1]
				}
			}
		}
	}

	for i, p := range ps {
		if _, ok := links[p.Name]; ok {
			ps[i].ServiceType = links[p.Name]
		}
	}

	return ps, nil
}

func GetProcess(app, name string) (*Process, error) {
	processes, err := ListProcesses(app)

	if err != nil {
		return nil, err
	}

	for _, p := range processes {
		if p.Name == name {
			return &p, nil
		}
	}

	return nil, nil
}

func (p *Process) Run(options ProcessRunOptions) error {
	app, err := GetApp(p.App)

	if err != nil {
		return err
	}

	resources := app.Resources()

	req := &ecs.RunTaskInput{
		Cluster:        aws.String(os.Getenv("CLUSTER")),
		Count:          aws.Long(1),
		TaskDefinition: aws.String(resources["TaskDefinition"].Id),
	}

	if options.Command != "" {
		req.Overrides = &ecs.TaskOverride{
			ContainerOverrides: []*ecs.ContainerOverride{
				&ecs.ContainerOverride{
					Name: aws.String(p.Name),
					Command: []*string{
						aws.String("sh"),
						aws.String("-c"),
						aws.String(options.Command),
					},
				},
			},
		}
	}

	_, err = ECS().RunTask(req)

	if err != nil {
		return err
	}

	return nil
}

func (p *Process) SubscribeLogs(output chan []byte, quit chan bool) error {
	// resources, err := ListResources(p.App)
	// fmt.Printf("err %+v\n", err)

	// if err != nil {
	//   return err
	// }

	// done := make(chan bool)
	// go subscribeKinesis(p.Name, resources[fmt.Sprintf("%sKinesis", upperName(p.Name))].Id, output, done)

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
