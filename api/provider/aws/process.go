package aws

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
	"github.com/convox/rack/api/structs"
)

var (
	StatusCodePrefix = "F1E49A85-0AD7-4AEF-A618-C249C6E6568D:" // needs to be random
)

func (p *AWSProvider) ProcessList(app string) (structs.Processes, error) {
	psch := make(chan structs.Process)
	errch := make(chan error)
	num := 0

	tasks, err := p.appTasks(app)

	for _, task := range tasks {
		td, err := p.ecs().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: task.TaskDefinitionArn,
		})

		if err != nil {
			return nil, err
		}

		for _, cd := range td.TaskDefinition.ContainerDefinitions {
			var cc *ecs.Container

			for _, c := range task.Containers {
				if *c.Name == *cd.Name {
					cc = c
				}
			}

			go p.fetchProcess(app, *task, *td.TaskDefinition, *cd, *cc, psch, errch)

			num += 1
		}
	}

	pss := structs.Processes{}

	for i := 0; i < num; i++ {
		select {
		case ps := <-psch:
			pss = append(pss, ps)
		case err := <-errch:
			return nil, err
		}
	}

	pending, err := p.pendingProcesses(app)

	if err != nil {
		return nil, err
	}

	pss = append(pss, pending...)

	// FIXME This codepath gets the wrong IP for the Docker host
	// It should get the internal IP running on AWS
	// oneoff, err := ListOneoffProcesses(app)

	// if err != nil {
	// 	return nil, err
	// }

	// pss = append(pss, oneoff...)

	sort.Sort(pss)

	return pss, nil
}

func (p *AWSProvider) ProcessGet(app, pid string) (*structs.Process, error) {
	processes, err := p.ProcessList(app)

	if err != nil {
		return nil, err
	}

	for _, p := range processes {
		if p.Id == pid {
			return &p, nil
		}
	}

	return nil, nil
}

func (p *AWSProvider) ProcessStop(app, pid string) error {
	tasks, err := p.appTasks(app)

	if err != nil {
		return err
	}

	for _, task := range tasks {
		for _, container := range task.Containers {
			if pidFromContainer(*container) == pid {
				req := &ecs.StopTaskInput{
					Cluster: aws.String(os.Getenv("CLUSTER")),
					Task:    aws.String(*task.TaskArn),
				}

				_, err := p.ecs().StopTask(req)

				if err != nil {
					return err
				}

				return nil
			}
		}
	}

	return fmt.Errorf("cannot stop pid: %s", pid)
}

func (p *AWSProvider) ProcessExec(app string, pid, command string, rw io.ReadWriter) error {
	ps, err := p.ProcessGet(app, pid)

	if err != nil {
		return err
	}

	d, err := dockerClient(fmt.Sprintf("http://%s:2376", ps.Host))

	if err != nil {
		return err
	}

	res, err := d.CreateExec(docker.CreateExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{"sh", "-c", command},
		Container:    ps.Container,
	})

	if err != nil {
		return err
	}

	// Create pipes so StartExec closes pipes, not the websocket.
	ir, iw := io.Pipe()
	or, ow := io.Pipe()

	go io.Copy(iw, rw)
	go io.Copy(rw, or)

	err = d.StartExec(res.ID, docker.StartExecOptions{
		Detach:       false,
		Tty:          true,
		InputStream:  ir,
		OutputStream: ow,
		ErrorStream:  ow,
		RawTerminal:  true,
	})

	if err != nil && err != io.ErrClosedPipe {
		return err
	}

	ires, err := d.InspectExec(res.ID)

	if err != nil {
		return err
	}

	_, err = rw.Write([]byte(fmt.Sprintf("%s%d\n", StatusCodePrefix, ires.ExitCode)))

	if err != nil {
		return err
	}

	return nil
}

func (p *AWSProvider) ProcessStats(app, pid string) (*structs.ProcessStats, error) {
	ps, err := p.ProcessGet(app, pid)

	if err != nil {
		return nil, err
	}

	d, err := dockerClient(fmt.Sprintf("http://%s:2376", ps.Host))

	if err != nil {
		return nil, err
	}

	stch := make(chan *docker.Stats)
	dnch := make(chan bool)

	options := docker.StatsOptions{
		ID:     ps.Container,
		Stats:  stch,
		Done:   dnch,
		Stream: false,
	}

	go d.Stats(options)

	stat := <-stch
	dnch <- true

	pcpu := stat.PreCPUStats.CPUUsage.TotalUsage
	psys := stat.PreCPUStats.SystemCPUUsage

	stats := &structs.ProcessStats{}

	stats.Cpu = calculateCPUPercent(pcpu, psys, stat)

	if stat.MemoryStats.Limit > 0 {
		stats.Memory = float64(stat.MemoryStats.Usage) / float64(stat.MemoryStats.Limit)
	}

	return stats, nil
}

/** helpers ****************************************************************************************/

// // from https://github.com/docker/docker/blob/master/api/client/stats.go
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

func (p *AWSProvider) pendingProcesses(app string) (structs.Processes, error) {
	pss := structs.Processes{}

	services, err := p.appServices(app)

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

			tres, err := p.ecs().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
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
					ps := structs.Process{
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

// func (p *AWSProvider) oneOffProcesses(app string) (structs.Processes, error) {
//   instances, err := p.InstanceList()

//   if err != nil {
//     return nil, err
//   }

//   procs := structs.Processes{}

//   for _, instance := range instances {
//     d, err := Docker(instance.Ip)

//     if err != nil {
//       return nil, err
//     }

//     pss, err := d.ListContainers(docker.ListContainersOptions{
//       Filters: map[string][]string{
//         "label": []string{
//           "com.convox.rack.type=oneoff",
//           fmt.Sprintf("com.convox.rack.app=%s", app),
//         },
//       },
//     })

//     if err != nil {
//       return nil, err
//     }

//     for _, ps := range pss {
//       procs = append(procs, Process{
//         Id:      ps.ID[0:12],
//         Command: ps.Command,
//         Name:    ps.Labels["com.convox.rack.process"],
//         Release: ps.Labels["com.convox.rack.release"],
//         Started: time.Unix(ps.Created, 0),
//       })
//     }
//   }

//   return procs, nil
// }

func (p *AWSProvider) fetchProcess(app string, task ecs.Task, td ecs.TaskDefinition, cd ecs.ContainerDefinition, c ecs.Container, psch chan structs.Process, errch chan error) {
	ps := structs.Process{
		Id:    pidFromContainer(c),
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
		ps.Binds = append(ps.Binds, fmt.Sprintf("%v:%v", hostVolumes[*m.SourceVolume], *m.ContainerPath))
	}

	cres, err := p.ecs().DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(os.Getenv("CLUSTER")),
		ContainerInstances: []*string{task.ContainerInstanceArn},
	})

	if err != nil {
		errch <- err
		return
	}

	if len(cres.ContainerInstances) != 1 {
		errch <- fmt.Errorf("could not find ECS instance")
		return
	}

	ci := cres.ContainerInstances[0]

	ires, err := p.ec2().DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{Name: aws.String("instance-id"), Values: []*string{ci.Ec2InstanceId}},
		},
	})

	if err != nil {
		errch <- err
		return
	}

	if len(ires.Reservations) != 1 || len(ires.Reservations[0].Instances) != 1 {
		errch <- fmt.Errorf("could not find EC2 instance")
		return
	}

	instance := ires.Reservations[0].Instances[0]

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

	d, err := dockerClient(fmt.Sprintf("http://%s:2376", ps.Host))

	if err != nil {
		errch <- fmt.Errorf("could not communicate with docker")
		return
	}

	containers, err := d.ListContainers(docker.ListContainersOptions{
		Filters: map[string][]string{
			"label": []string{
				fmt.Sprintf("com.amazonaws.ecs.task-arn=%s", *task.TaskArn),
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

	ps.Command = containers[0].Command
	ps.Container = containers[0].ID
	ps.Started = time.Unix(containers[0].Created, 0)

	for _, port := range containers[0].Ports {
		ps.Ports = append(ps.Ports, fmt.Sprintf("%d:%d", port.PublicPort, port.PrivatePort))
	}

	psch <- ps
}

func pidFromContainer(c ecs.Container) string {
	idp := strings.Split(*c.ContainerArn, "-")
	return idp[len(idp)-1]
}
