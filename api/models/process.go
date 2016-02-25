package models

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
	"github.com/convox/rack/api/config"
	"github.com/convox/rack/api/provider"
)

type Process struct {
	Id      string    `json:"id"`
	App     string    `json:"app"`
	Command string    `json:"command"`
	Host    string    `json:"host"`
	Image   string    `json:"image"`
	Name    string    `json:"name"`
	Ports   []string  `json:"ports"`
	Release string    `json:"release"`
	Size    int64     `json:"size"`
	Cpu     float64   `json:"cpu"`
	Memory  float64   `json:"memory"`
	Started time.Time `json:"started"`

	binds       []string `json:"-"`
	containerId string   `json:"-"`
	taskArn     string   `json:"-"`
}

type Processes []Process

func GetAppServices(app string) ([]*ecs.Service, error) {
	services := []*ecs.Service{}

	resources, err := ListResources(app)

	if err != nil {
		return nil, err
	}

	arns := []*string{}

	i := 0
	for _, r := range resources {
		i = i + 1

		if r.Type == "Custom::ECSService" {
			arns = append(arns, aws.String(r.Id))
		}

		//have to make requests in batches of ten
		if len(arns) == 10 || (i == len(resources) && len(arns) > 0) {
			dres, err := ECS().DescribeServices(&ecs.DescribeServicesInput{
				Cluster:  aws.String(os.Getenv("CLUSTER")),
				Services: arns,
			})

			if err != nil {
				return nil, err
			}

			services = append(services, dres.Services...)
			arns = []*string{}
		}
	}

	return services, nil
}

func ListProcesses(app string) (Processes, error) {
	a, err := GetApp(app)

	if err != nil {
		return nil, err
	}

	resources, err := a.Resources()

	if err != nil {
		return nil, err
	}

	services := []string{}

	for _, resource := range resources {
		if resource.Type == "Custom::ECSService" {
			parts := strings.Split(resource.Id, "/")
			services = append(services, parts[len(parts)-1])
		}
	}

	// get ECS and EC2 instance info up front
	lres, err := ECS().ListContainerInstances(&ecs.ListContainerInstancesInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
	})

	if err != nil {
		return nil, err
	}

	dres, err := ECS().DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(os.Getenv("CLUSTER")),
		ContainerInstances: lres.ContainerInstanceArns,
	})

	if err != nil {
		return nil, err
	}

	instanceIds := []*string{}

	for _, i := range dres.ContainerInstances {
		instanceIds = append(instanceIds, i.Ec2InstanceId)
	}

	ires, err := EC2().DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: instanceIds,
	})

	if err != nil {
		return nil, err
	}

	psch := make(chan Process)
	errch := make(chan error)
	num := 0

	for _, service := range services {
		taskArns, err := ECS().ListTasks(&ecs.ListTasksInput{
			Cluster:     aws.String(os.Getenv("CLUSTER")),
			ServiceName: aws.String(service),
		})

		if err != nil {
			return nil, err
		}

		if len(taskArns.TaskArns) == 0 {
			continue
		}

		tasks, err := ECS().DescribeTasks(&ecs.DescribeTasksInput{
			Cluster: aws.String(os.Getenv("CLUSTER")),
			Tasks:   taskArns.TaskArns,
		})

		if err != nil {
			return nil, err
		}

		for _, task := range tasks.Tasks {
			td, err := ECS().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
				TaskDefinition: task.TaskDefinitionArn,
			})

			if err != nil {
				return nil, err
			}

			var ci *ecs.ContainerInstance
			var ec2i *ec2.Instance

			for _, i := range dres.ContainerInstances {
				if *i.ContainerInstanceArn == *task.ContainerInstanceArn {
					ci = i
					break
				}
			}

			if ci == nil {
				return nil, fmt.Errorf("could not find ECS instance")
			}

			for _, r := range ires.Reservations {
				for _, i := range r.Instances {
					if *ci.Ec2InstanceId == *i.InstanceId {
						ec2i = i
						break
					}
				}
			}

			if ec2i == nil {
				return nil, fmt.Errorf("could not find EC2 instance")
			}

			for _, cd := range td.TaskDefinition.ContainerDefinitions {
				var cc *ecs.Container

				for _, c := range task.Containers {
					if *c.Name == *cd.Name {
						cc = c
						break
					}
				}

				go fetchProcess(app, *task, *td.TaskDefinition, *cd, *cc, *ci, *ec2i, psch, errch)

				num += 1
			}
		}
	}

	pss := Processes{}

	for i := 0; i < num; i++ {
		select {
		case ps := <-psch:
			pss = append(pss, ps)
		case err := <-errch:
			return nil, err
		}
	}

	pending, err := ListPendingProcesses(app)

	if err != nil {
		return nil, err
	}

	pss = append(pss, pending...)

	// This codepath gets the wrong IP for the Docker host
	// It should get the internal IP running on AWS
	// oneoff, err := ListOneoffProcesses(app)

	// if err != nil {
	// 	return nil, err
	// }

	// pss = append(pss, oneoff...)

	sort.Sort(pss)

	return pss, nil
}

func ListPendingProcesses(app string) (Processes, error) {
	pss := Processes{}

	services, err := GetAppServices(app)

	if err != nil {
		return nil, err
	}

	for _, service := range services {
		// Test every service deployment for running != pending, to put in a placeholder
		for _, d := range service.Deployments {
			if *d.Status != "PRIMARY" {
				continue
			}

			if *d.DesiredCount == *d.RunningCount {
				continue
			}

			tres, err := ECS().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
				TaskDefinition: d.TaskDefinition,
			})

			if err != nil {
				return nil, err
			}

			if len(tres.TaskDefinition.ContainerDefinitions) == 0 {
				continue
			}

			for i := *d.RunningCount; i < (*d.DesiredCount - *d.PendingCount); i++ {
				for _, cd := range tres.TaskDefinition.ContainerDefinitions {
					ps := Process{
						Id:   "pending",
						Name: *cd.Name,
						Size: *cd.Memory,
					}

					for _, env := range cd.Environment {
						if *env.Name == "RELEASE" {
							ps.Release = *env.Value
						}
					}

					pss = append(pss, ps)
				}
			}
		}
	}

	return pss, nil
}

func ListOneoffProcesses(app string) (Processes, error) {
	instances, err := provider.InstanceList()

	if err != nil {
		return nil, err
	}

	procs := Processes{}

	for _, instance := range instances {
		d, err := Docker(instance.Ip)

		if err != nil {
			return nil, err
		}

		pss, err := d.ListContainers(docker.ListContainersOptions{
			Filters: map[string][]string{
				"label": []string{
					"com.convox.rack.type=oneoff",
					fmt.Sprintf("com.convox.rack.app=%s", app),
				},
			},
		})

		if err != nil {
			return nil, err
		}

		for _, ps := range pss {
			procs = append(procs, Process{
				Id:      ps.ID[0:12],
				Command: ps.Command,
				Name:    ps.Labels["com.convox.rack.process"],
				Release: ps.Labels["com.convox.rack.release"],
				Started: time.Unix(ps.Created, 0),
			})
		}
	}

	return procs, nil
}

func fetchProcess(app string, task ecs.Task, td ecs.TaskDefinition, cd ecs.ContainerDefinition, c ecs.Container, ci ecs.ContainerInstance, instance ec2.Instance, psch chan Process, errch chan error) {
	idp := strings.Split(*c.ContainerArn, "-")
	id := idp[len(idp)-1]

	ps := Process{
		Id:    id,
		App:   app,
		Image: *cd.Image,
		Name:  *cd.Name,
		Ports: []string{},
		Size:  *cd.Memory,
	}

	for _, env := range cd.Environment {
		if *env.Name == "RELEASE" {
			ps.Release = *env.Value
		}
	}

	hostVolumes := make(map[string]string)

	for _, v := range td.Volumes {
		hostVolumes[*v.Name] = *v.Host.SourcePath
	}

	for _, m := range cd.MountPoints {
		ps.binds = append(ps.binds, fmt.Sprintf("%v:%v", hostVolumes[*m.SourceVolume], *m.ContainerPath))
	}

	ps.taskArn = *task.TaskArn

	// if there's no private ip address we have no more information to grab
	if instance.PrivateIpAddress == nil {
		psch <- ps
		return
	}

	// Connect to a Docker client
	// In testing use the stub Docker server.
	// In development, modify the security group for port 2376 and use the public IP
	// In production, use the private IP

	ip := *instance.PrivateIpAddress

	if os.Getenv("DEVELOPMENT") == "true" {
		ip = *instance.PublicIpAddress
	}

	ps.Host = ip

	d, err := ps.Docker()

	if err != nil {
		errch <- fmt.Errorf("could not communicate with docker")
		return
	}

	containers, err := d.ListContainers(docker.ListContainersOptions{
		Filters: map[string][]string{
			"label": []string{
				fmt.Sprintf("com.amazonaws.ecs.task-arn=%s", ps.taskArn),
				fmt.Sprintf("com.amazonaws.ecs.container-name=%s", ps.Name),
			},
		},
	})

	if err != nil {
		errch <- err
		return
	}

	if len(containers) != 1 {
		fmt.Println(`ns=kernel at=processes.list state=error message="could not find container"`)
		psch <- ps
		return
	}

	ps.containerId = containers[0].ID
	ps.Command = containers[0].Command
	ps.Started = time.Unix(containers[0].Created, 0)

	for _, port := range containers[0].Ports {
		ps.Ports = append(ps.Ports, fmt.Sprintf("%d:%d", port.PublicPort, port.PrivatePort))
	}

	psch <- ps
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

// Sort processes by name and id
// Processes with a 'pending' id will naturally come last by design
func (ps Processes) Less(i, j int) bool {
	psi := fmt.Sprintf("%s-%s", ps[i].Name, ps[i].Id)
	psj := fmt.Sprintf("%s-%s", ps[j].Name, ps[j].Id)

	return psi < psj
}

func (ps Processes) Swap(i, j int) {
	ps[i], ps[j] = ps[j], ps[i]
}

var ErrPending = errors.New("can not get docker client for non-running container")

func (p *Process) Docker() (*docker.Client, error) {
	if p.Id == "pending" {
		return nil, ErrPending
	}

	if os.Getenv("TEST") == "true" {
		return Docker(config.TestConfig.DockerHost)
	}

	return Docker(fmt.Sprintf("http://%s:2376", p.Host))
}

func (p *Process) FetchStats() error {
	d, err := p.Docker()

	if err != nil {
		if err == ErrPending {
			return nil
		}

		return fmt.Errorf("could not communicate with docker")
	}

	stch := make(chan *docker.Stats)
	dnch := make(chan bool)

	options := docker.StatsOptions{
		ID:     p.containerId,
		Stats:  stch,
		Done:   dnch,
		Stream: false,
	}

	go d.Stats(options)

	stat := <-stch
	dnch <- true

	pcpu := stat.PreCPUStats.CPUUsage.TotalUsage
	psys := stat.PreCPUStats.SystemCPUUsage

	p.Cpu = truncate(calculateCPUPercent(pcpu, psys, stat), 4)

	if stat.MemoryStats.Limit > 0 {
		p.Memory = truncate(float64(stat.MemoryStats.Usage)/float64(stat.MemoryStats.Limit), 4)
	}

	return nil
}

func (p *Process) FetchStatsAsync(psch chan Process, errch chan error) {
	errch <- p.FetchStats()
	psch <- *p
}

func (p *Process) Stop() error {
	if p.taskArn == "" {
		return fmt.Errorf("can not stop one-off processes")
	}

	req := &ecs.StopTaskInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
		Task:    aws.String(p.taskArn),
	}

	_, err := ECS().StopTask(req)

	if err != nil {
		return err
	}

	return nil
}

// from https://github.com/docker/docker/blob/master/api/client/stats.go
func calculateCPUPercent(previousCPU, previousSystem uint64, v *docker.Stats) float64 {
	var (
		cpuPercent = 0.0
		// calculate the change for the cpu usage of the container in between readings
		cpuDelta = float64(v.CPUStats.CPUUsage.TotalUsage - previousCPU)
		// calculate the change for the entire system between readings
		systemDelta = float64(v.CPUStats.SystemCPUUsage - previousSystem)
	)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(len(v.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return cpuPercent
}
