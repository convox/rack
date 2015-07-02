package models

import (
	"fmt"
	"os"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ecs"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
)

type Process struct {
	App         string
	Command     string
	Count       int
	CPU         int64
	Id          string
	Memory      int64
	Name        string
	ServiceType string
	TaskARN     string
}

type Processes []Process

type ProcessRunOptions struct {
	Command string
}

func ListProcesses(app string) (Processes, error) {
	req := &ecs.ListTasksInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
		Family:  aws.String(app),
	}

	res, err := ECS().ListTasks(req)

	if err != nil {
		return nil, err
	}

	treq := &ecs.DescribeTasksInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
		Tasks:   res.TaskARNs,
	}

	tres, err := ECS().DescribeTasks(treq)

	pss := Processes{}

	for _, task := range tres.Tasks {
		parts := strings.Split(*task.TaskARN, "-")
		id := parts[len(parts)-1]

		tres, err := ECS().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: task.TaskDefinitionARN,
		})

		if err != nil {
			return nil, err
		}

		definitions := map[string]*ecs.ContainerDefinition{}

		for _, cd := range tres.TaskDefinition.ContainerDefinitions {
			definitions[*cd.Name] = cd
		}

		for _, container := range task.Containers {
			ps := Process{
				Id:      id,
				TaskARN: *task.TaskARN,
				Name:    *container.Name,
				App:     app,
			}

			ps.CPU = *definitions[ps.Name].CPU
			ps.Memory = *definitions[ps.Name].Memory

			cres, err := ECS().DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
				Cluster:            aws.String(os.Getenv("CLUSTER")),
				ContainerInstances: []*string{task.ContainerInstanceARN},
			})

			if err != nil {
				return nil, err
			}

			if len(cres.ContainerInstances) == 1 {
				ci := cres.ContainerInstances[0]

				ires, err := EC2().DescribeInstances(&ec2.DescribeInstancesInput{
					Filters: []*ec2.Filter{
						&ec2.Filter{Name: aws.String("instance-id"), Values: []*string{ci.EC2InstanceID}},
					},
				})

				if err != nil {
					return nil, err
				}

				if len(ires.Reservations) == 1 && len(ires.Reservations[0].Instances) == 1 {
					instance := ires.Reservations[0].Instances[0]

					ip := *instance.PrivateIPAddress

					if os.Getenv("DEVELOPMENT") == "true" {
						ip = *instance.PublicIPAddress
					}

					containers, err := Docker(ip).ListContainers(docker.ListContainersOptions{
						Filters: map[string][]string{
							"label": []string{
								fmt.Sprintf("com.amazonaws.ecs.task-arn=%s", ps.TaskARN),
								fmt.Sprintf("com.amazonaws.ecs.container-name=%s", ps.Name),
							},
						},
					})

					if err != nil {
						return nil, err
					}

					if len(containers) == 1 {
						ps.Command = containers[0].Command
					}
				}
			}

			if !strings.HasPrefix(*container.Name, "convox-") {
				pss = append(pss, ps)
			}
		}
	}

	return pss, nil
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

func GetProcessById(app, id string) (*Process, error) {
	processes, err := ListProcesses(app)

	if err != nil {
		return nil, err
	}

	for _, p := range processes {
		if p.Id == id {
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

func (p *Process) Stop() error {
	req := &ecs.StopTaskInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
		Task:    aws.String(p.TaskARN),
	}

	_, err := ECS().StopTask(req)

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
