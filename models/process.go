package models

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ecs"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
)

type Process struct {
	Id      string `json:"id"`
	App     string `json:"app"`
	Command string `json:"command"`
	Image   string `json:"image"`
	Name    string `json:"name"`

	Binds   []string `json:"-"`
	TaskARN string   `json:"-"`
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

			cp := make([]string, len(cd.Command))

			for i, part := range cd.Command {
				cp[i] = *part
			}

			ps := Process{
				Id:      id,
				App:     app,
				Command: strings.Join(cp, " "),
				Image:   *cd.Image,
				Name:    *cd.Name,
			}

			hostVolumes := make(map[string]string)

			for _, v := range tres.TaskDefinition.Volumes {
				hostVolumes[*v.Name] = *v.Host.SourcePath
			}

			for _, m := range cd.MountPoints {
				ps.Binds = append(ps.Binds, fmt.Sprintf("%v:%v", hostVolumes[*m.SourceVolume], *m.ContainerPath))
			}

			ps.TaskARN = *task.TaskARN

			pss = append(pss, ps)
		}
	}

	sort.Sort(pss)

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

// func (p *Process) Top() (*ProcessTop, error) {
//   res, err := p.Docker().TopContainer(p.ContainerId, "")

//   if err != nil {
//     return nil, err
//   }

//   info := &ProcessTop{
//     Titles:    res.Titles,
//     Processes: res.Processes,
//   }

//   return info, nil
// }

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

	d := Docker()

	res, err := d.CreateContainer(docker.CreateContainerOptions{
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

	go d.AttachToContainer(docker.AttachToContainerOptions{
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

	err = d.StartContainer(res.ID, nil)

	if err != nil {
		return err
	}

	code, err := d.WaitContainer(res.ID)

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

func (ps Processes) Len() int {
	return len(ps)
}

func (ps Processes) Less(i, j int) bool {
	return ps[i].Name < ps[j].Name
}

func (ps Processes) Swap(i, j int) {
	ps[i], ps[j] = ps[j], ps[i]
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
