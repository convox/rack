package models

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ecs"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
)

type Process struct {
	Id      string    `json:"id"`
	App     string    `json:"app"`
	Command string    `json:"command"`
	Host    string    `json:"host"`
	Image   string    `json:"image"`
	Name    string    `json:"name"`
	Ports   []string  `json:"ports"`
	Started time.Time `json:"started"`

	binds       []string `json:"-"`
	containerId string   `json:"-"`
	taskARN     string   `json:"-"`
}

type Processes []Process

func ListProcesses(app string) (Processes, error) {
	_, err := GetApp(app)

	if err != nil {
		return nil, err
	}

	req := &ecs.ListTasksInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
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
		for _, c := range task.Containers {
			tres, err := ECS().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
				TaskDefinition: task.TaskDefinitionARN,
			})

			if err != nil {
				return nil, err
			}

			if !strings.HasPrefix(*tres.TaskDefinition.Family, app+"-") && *tres.TaskDefinition.Family != app {
				continue
			}

			if len(tres.TaskDefinition.ContainerDefinitions) < 1 {
				return nil, fmt.Errorf("no container definition")
			}

			cd := *(tres.TaskDefinition.ContainerDefinitions[0])

			idp := strings.Split(*c.ContainerARN, "-")
			id := idp[len(idp)-1]

			ps := Process{
				Id:    id,
				App:   app,
				Image: *cd.Image,
				Name:  *cd.Name,
				Ports: []string{},
			}

			hostVolumes := make(map[string]string)

			for _, v := range tres.TaskDefinition.Volumes {
				hostVolumes[*v.Name] = *v.Host.SourcePath
			}

			fmt.Printf("cd %+v\n", cd)

			for _, m := range cd.MountPoints {
				ps.binds = append(ps.binds, fmt.Sprintf("%v:%v", hostVolumes[*m.SourceVolume], *m.ContainerPath))
			}

			ps.taskARN = *task.TaskARN

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

					ps.Host = ip

					d, err := ps.Docker()

					if err != nil {
						return nil, fmt.Errorf("could not communicate with docker")
					}

					containers, err := d.ListContainers(docker.ListContainersOptions{
						Filters: map[string][]string{
							"label": []string{
								fmt.Sprintf("com.amazonaws.ecs.task-arn=%s", ps.taskARN),
								fmt.Sprintf("com.amazonaws.ecs.container-name=%s", ps.Name),
							},
						},
					})

					if err != nil {
						return nil, err
					}

					if len(containers) == 1 {
						fmt.Printf("containers[0] %+v\n", containers[0])

						ps.containerId = containers[0].ID
						ps.Command = containers[0].Command
						ps.Started = time.Unix(containers[0].Created, 0)

						for _, port := range containers[0].Ports {
							ps.Ports = append(ps.Ports, fmt.Sprintf("%d:%d", port.PublicPort, port.PrivatePort))
						}
					}
				}
			}

			pss = append(pss, ps)
		}
	}

	sort.Sort(pss)

	return pss, nil
}

func GetProcess(app, id string) (*Process, error) {
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

func (ps Processes) Len() int {
	return len(ps)
}

func (ps Processes) Less(i, j int) bool {
	return ps[i].Name < ps[j].Name
}

func (ps Processes) Swap(i, j int) {
	ps[i], ps[j] = ps[j], ps[i]
}

func (p *Process) Docker() (*docker.Client, error) {
	return Docker(fmt.Sprintf("http://%s:2376", p.Host))
}

func (p *Process) Stop() error {
	req := &ecs.StopTaskInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
		Task:    aws.String(p.taskARN),
	}

	_, err := ECS().StopTask(req)

	if err != nil {
		return err
	}

	return nil
}
