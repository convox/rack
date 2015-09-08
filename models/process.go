package models

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ecs"
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
