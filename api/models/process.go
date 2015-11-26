package models

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
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
	Cpu     float64   `json:"cpu"`
	Memory  float64   `json:"memory"`
	Started time.Time `json:"started"`

	binds       []string `json:"-"`
	containerId string   `json:"-"`
	taskArn     string   `json:"-"`
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
		Tasks:   res.TaskArns,
	}

	tres, err := ECS().DescribeTasks(treq)

	pss := Processes{}

	psch := make(chan Process)
	errch := make(chan error)
	num := 0

	for _, task := range tres.Tasks {
		tres, err := ECS().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: task.TaskDefinitionArn,
		})

		if err != nil {
			return nil, err
		}

		for _, cd := range tres.TaskDefinition.ContainerDefinitions {
			family := *tres.TaskDefinition.Family

			if len(tres.TaskDefinition.ContainerDefinitions) == 0 {
				continue
			}

			// if this is the rack stack the family name is the app name
			// otherwise family should be app name + process name
			if app == os.Getenv("RACK") {
				if family != app {
					continue
				}
			} else {
				if family != fmt.Sprintf("%s-%s", app, *tres.TaskDefinition.ContainerDefinitions[0].Name) {
					continue
				}
			}

			var cc *ecs.Container

			for _, c := range task.Containers {
				if *c.Name == *cd.Name {
					cc = c
				}
			}

			go fetchProcess(app, *task, *tres.TaskDefinition, *cd, *cc, psch, errch)

			num += 1
		}
	}

	for i := 0; i < num; i++ {
		select {
		case ps := <-psch:
			pss = append(pss, ps)
		case err := <-errch:
			return nil, err
		}
	}

	sort.Sort(pss)

	return pss, nil
}

func fetchProcess(app string, task ecs.Task, td ecs.TaskDefinition, cd ecs.ContainerDefinition, c ecs.Container, psch chan Process, errch chan error) {
	idp := strings.Split(*c.ContainerArn, "-")
	id := idp[len(idp)-1]

	ps := Process{
		Id:    id,
		App:   app,
		Image: *cd.Image,
		Name:  *cd.Name,
		Ports: []string{},
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

	cres, err := ECS().DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(os.Getenv("CLUSTER")),
		ContainerInstances: []*string{task.ContainerInstanceArn},
	})

	if err != nil {
		errch <- err
		return
	}

	if len(cres.ContainerInstances) != 1 {
		errch <- fmt.Errorf("could not find instance")
		return
	}

	ci := cres.ContainerInstances[0]

	ires, err := EC2().DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{Name: aws.String("instance-id"), Values: []*string{ci.Ec2InstanceId}},
		},
	})

	if err != nil {
		errch <- err
		return
	}

	if len(ires.Reservations) != 1 || len(ires.Reservations[0].Instances) != 1 {
		errch <- fmt.Errorf("could not find instance")
		return
	}

	instance := ires.Reservations[0].Instances[0]

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

func (ps Processes) Less(i, j int) bool {
	return ps[i].Name < ps[j].Name
}

func (ps Processes) Swap(i, j int) {
	ps[i], ps[j] = ps[j], ps[i]
}

func (p *Process) Docker() (*docker.Client, error) {
	return Docker(fmt.Sprintf("http://%s:2376", p.Host))
}

func (p *Process) FetchStats() error {
	d, err := p.Docker()

	if err != nil {
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
