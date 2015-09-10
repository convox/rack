package models

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
)

type Process struct {
	App         string
	Binds       []string
	Command     string
	ContainerId string
	Count       int
	CPU         int64
	DockerHost  string
	Id          string
	Image       string
	Memory      int64
	Name        string
	Release     string
	ServiceType string
	TaskARN     string
}

type Processes []Process

type ProcessTop struct {
	Titles    []string
	Processes [][]string
}

type ProcessRunOptions struct {
	Command string
	Process string
}

func ListProcesses(app string) (Processes, error) {
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
		tres, err := ECS().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: task.TaskDefinitionARN,
		})

		if err != nil {
			return nil, err
		}

		if !strings.HasPrefix(*tres.TaskDefinition.Family, app+"-") && *tres.TaskDefinition.Family != app {
			continue
		}

		definitions := map[string]*ecs.ContainerDefinition{}

		for _, cd := range tres.TaskDefinition.ContainerDefinitions {
			definitions[*cd.Name] = cd
		}

		hostVolumes := make(map[string]string)

		for _, v := range tres.TaskDefinition.Volumes {
			hostVolumes[*v.Name] = *v.Host.SourcePath
		}

		for _, container := range task.Containers {
			parts := strings.Split(*container.ContainerARN, "-")
			id := parts[len(parts)-1]

			ps := Process{
				Id:      id,
				TaskARN: *task.TaskARN,
				Name:    *container.Name,
				App:     app,
			}

			if td, ok := definitions[ps.Name]; ok {
				for _, env := range td.Environment {
					if *env.Name == "RELEASE" {
						ps.Release = *env.Value
					}
				}

				for _, m := range td.MountPoints {
					ps.Binds = append(ps.Binds, fmt.Sprintf("%v:%v", hostVolumes[*m.SourceVolume], *m.ContainerPath))
				}
			}

			ps.Image = *definitions[ps.Name].Image
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

					ps.DockerHost = ip

					containers, err := ps.Docker().ListContainers(docker.ListContainersOptions{
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
						ps.ContainerId = containers[0].ID
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

func (p *Process) Docker() *docker.Client {
	client, _ := docker.NewClient(fmt.Sprintf("http://%s:2376", p.DockerHost))

	if os.Getenv("TEST_DOCKER_HOST") != "" {
		client, _ = docker.NewClient(os.Getenv("TEST_DOCKER_HOST"))
	}

	return client
}

func (p *Process) Top() (*ProcessTop, error) {
	res, err := p.Docker().TopContainer(p.ContainerId, "")

	if err != nil {
		return nil, err
	}

	info := &ProcessTop{
		Titles:    res.Titles,
		Processes: res.Processes,
	}

	return info, nil
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
		TaskDefinition: aws.String(resources[UpperName(options.Process)+"ECSTaskDefinition"].Id),
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

func (p *Process) RunAttached(command string, rw io.ReadWriter) error {
	env, err := GetEnvironment(p.App)

	if err != nil {
		return err
	}

	ea := make([]string, 0)

	for k, v := range env {
		ea = append(ea, fmt.Sprintf("%s=%s", k, v))
	}

	res, err := p.Docker().CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Env:          ea,
			OpenStdin:    true,
			Tty:          true,
			Cmd:          []string{"sh", "-c", command},
			Image:        p.Image,
		},
		HostConfig: &docker.HostConfig{
			Binds: p.Binds,
		},
	})

	if err != nil {
		return err
	}

	go p.Docker().AttachToContainer(docker.AttachToContainerOptions{
		Container:    res.ID,
		InputStream:  rw,
		OutputStream: rw,
		ErrorStream:  rw,
		Stream:       true,
		Stdin:        true,
		Stdout:       true,
		Stderr:       true,
		RawTerminal:  true,
	})

	// hacky
	time.Sleep(100 * time.Millisecond)

	err = p.Docker().StartContainer(res.ID, nil)

	if err != nil {
		return err
	}

	code, err := p.Docker().WaitContainer(res.ID)

	rw.Write([]byte(fmt.Sprintf("F1E49A85-0AD7-4AEF-A618-C249C6E6568D:%d", code)))

	if err != nil {
		return err
	}

	return nil
}

func copyWait(w io.Writer, r io.Reader, wg *sync.WaitGroup) {
	io.Copy(w, r)
	wg.Done()
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
