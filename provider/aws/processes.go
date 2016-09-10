package aws

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/manifest"
	"github.com/fsouza/go-dockerclient"
)

const (
	statusCodePrefix = "F1E49A85-0AD7-4AEF-A618-C249C6E6568D:"
)

// ProcessExec runs a command in an existing Process
func (p *AWSProvider) ProcessExec(app, pid, command string, stream io.ReadWriter, opts structs.ProcessExecOptions) error {
	arn, err := p.taskArnFromPid(pid)
	if err != nil {
		return err
	}

	task, err := p.describeTask(arn)
	if err != nil {
		return err
	}
	if len(task.Containers) < 1 {
		return fmt.Errorf("no running container for process: %s", pid)
	}

	cires, err := p.ecs().DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(p.Cluster),
		ContainerInstances: []*string{task.ContainerInstanceArn},
	})
	if err != nil {
		return err
	}
	if len(cires.ContainerInstances) < 1 {
		return fmt.Errorf("could not find instance for process: %s", pid)
	}

	dc, err := p.dockerInstance(*cires.ContainerInstances[0].Ec2InstanceId)
	if err != nil {
		return err
	}

	cs, err := dc.ListContainers(docker.ListContainersOptions{
		All: true,
		Filters: map[string][]string{
			"label": []string{fmt.Sprintf("com.amazonaws.ecs.task-arn=%s", arn)},
		},
	})
	if err != nil {
		return err
	}
	if len(cs) != 1 {
		return fmt.Errorf("could not find container for task: %s", arn)
	}

	eres, err := dc.CreateExec(docker.CreateExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{"sh", "-c", command},
		Container:    cs[0].ID,
	})
	if err != nil {
		return err
	}

	success := make(chan struct{})

	go func() {
		<-success
		dc.ResizeExecTTY(eres.ID, opts.Height, opts.Width)
		success <- struct{}{}
	}()

	err = dc.StartExec(eres.ID, docker.StartExecOptions{
		Detach:       false,
		Tty:          true,
		InputStream:  stream,
		OutputStream: stream,
		ErrorStream:  stream,
		RawTerminal:  true,
		Success:      success,
	})
	if err != nil {
		return err
	}

	ires, err := dc.InspectExec(eres.ID)
	if err != nil {
		return err
	}

	if _, err := stream.Write([]byte(fmt.Sprintf("%s%d\n", statusCodePrefix, ires.ExitCode))); err != nil {
		return err
	}

	return nil
}

// ProcessList returns a list of processes for an app
func (p *AWSProvider) ProcessList(app string) (structs.Processes, error) {
	rres, err := p.describeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(fmt.Sprintf("%s-%s", p.Rack, app)),
	})
	if ae, ok := err.(awserr.Error); ok && ae.Code() == "ValidationError" {
		return nil, ErrorNotFound(fmt.Sprintf("no such app: %s", app))
	}
	if err != nil {
		return nil, err
	}

	services := []string{}

	for _, r := range rres.StackResources {
		switch *r.ResourceType {
		case "AWS::ECS::Service", "Custom::ECSService":
			services = append(services, *r.PhysicalResourceId)
		}
	}

	tasks := []*string{}

	for _, s := range services {
		tres, err := p.ecs().ListTasks(&ecs.ListTasksInput{
			Cluster:     aws.String(p.Cluster),
			ServiceName: aws.String(s),
		})
		if err != nil {
			return nil, err
		}
		for _, arn := range tres.TaskArns {
			tasks = append(tasks, arn)
		}
	}

	// list one-off processes
	ores, err := p.ecs().ListTasks(&ecs.ListTasksInput{
		Cluster:   aws.String(p.Cluster),
		StartedBy: aws.String(fmt.Sprintf("convox.%s", app)),
	})
	if err != nil {
		return nil, err
	}

	for _, task := range ores.TaskArns {
		tasks = append(tasks, task)
	}

	tres, err := p.ecs().DescribeTasks(&ecs.DescribeTasksInput{
		Cluster: aws.String(p.Cluster),
		Tasks:   tasks,
	})
	if err != nil {
		return nil, err
	}

	pss := structs.Processes{}
	psch := make(chan structs.Process)
	errch := make(chan error)
	timeout := time.After(30 * time.Second)

	for _, task := range tres.Tasks {
		go p.fetchProcess(task, psch, errch)
	}

	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("timeout")
		case err := <-errch:
			return nil, err
		case ps := <-psch:
			ps.App = app
			pss = append(pss, ps)
		}

		if len(pss) == len(tres.Tasks) {
			break
		}
	}

	sort.Sort(pss)

	return pss, nil
}

// ProcessRun runs a new Process
func (p *AWSProvider) ProcessRun(app, process string, opts structs.ProcessRunOptions) (string, error) {
	if opts.Stream != nil {
		return p.processRunAttached(app, process, opts)
	}

	td, err := p.taskDefinitionForRun(app, process, opts.Release)
	if err != nil {
		return "", err
	}

	req := &ecs.RunTaskInput{
		Cluster:        aws.String(p.Cluster),
		Count:          aws.Int64(1),
		StartedBy:      aws.String(fmt.Sprintf("convox.%s", app)),
		TaskDefinition: aws.String(td),
	}

	if opts.Command != "" {
		req.Overrides = &ecs.TaskOverride{
			ContainerOverrides: []*ecs.ContainerOverride{
				&ecs.ContainerOverride{
					Name: aws.String(process),
					Command: []*string{
						aws.String("sh"),
						aws.String("-c"),
						aws.String(opts.Command),
					},
				},
			},
		}
	}

	task, err := p.runTask(req)
	if err != nil {
		return "", nil
	}

	return arnToPid(*task.TaskArn), nil
}

func (p *AWSProvider) runTask(req *ecs.RunTaskInput) (*ecs.Task, error) {
	res, err := p.ecs().RunTask(req)
	switch {
	case err != nil:
		return nil, err
	case len(res.Failures) > 0:
		switch *res.Failures[0].Reason {
		case "RESOURCE:MEMORY":
			return nil, fmt.Errorf("not enough memory available to start process")
		case "RESOURCE:PORTS":
			return nil, fmt.Errorf("no instance with available ports to start process")
		}
	case len(res.Tasks) != 1 || len(res.Tasks[0].Containers) != 1:
		return nil, fmt.Errorf("could not start process")
	}
	return res.Tasks[0], nil
}

func (p *AWSProvider) processRunAttached(app, process string, opts structs.ProcessRunOptions) (string, error) {
	td, err := p.taskDefinitionForRun(app, process, opts.Release)
	if err != nil {
		return "", err
	}

	req := &ecs.RunTaskInput{
		Cluster:        aws.String(p.Cluster),
		Count:          aws.Int64(1),
		StartedBy:      aws.String(fmt.Sprintf("convox.%s", app)),
		TaskDefinition: aws.String(td),
	}

	if opts.Command != "" {
		req.Overrides = &ecs.TaskOverride{
			ContainerOverrides: []*ecs.ContainerOverride{
				&ecs.ContainerOverride{
					Name: aws.String(process),
					Command: []*string{
						aws.String("sh"),
						aws.String("-c"),
						aws.String("sleep 60"),
					},
				},
			},
		}
	}

	task, err := p.runTask(req)
	if err != nil {
		return "", err
	}

	status, err := p.waitForTask(*task.TaskArn)
	if err != nil {
		return "", err
	}
	if status != "RUNNING" {
		return "", fmt.Errorf("error starting container")
	}

	pid := arnToPid(*task.TaskArn)

	err = p.ProcessExec(app, pid, opts.Command, opts.Stream, structs.ProcessExecOptions{
		Height: opts.Height,
		Width:  opts.Width,
	})
	if err != nil {
		return "", err
	}

	return pid, nil
}

// ProcessStop stops a Process
func (p *AWSProvider) ProcessStop(app, pid string) error {
	arn, err := p.taskArnFromPid(pid)
	if err != nil {
		return err
	}

	_, err = p.ecs().StopTask(&ecs.StopTaskInput{
		Cluster: aws.String(p.Cluster),
		Task:    aws.String(arn),
	})
	if err != nil {
		return err
	}

	return nil
}

func arnToPid(arn string) string {
	parts := strings.Split(arn, "-")
	return parts[len(parts)-1]
}

func commandString(cs []*string) string {
	cmd := make([]string, len(cs))

	for i, c := range cs {
		if strings.Contains(*c, " ") {
			cmd[i] = fmt.Sprintf("%q", *c)
		} else {
			cmd[i] = *c
		}
	}

	return strings.Join(cmd, " ")
}

func (p *AWSProvider) containerDefinitionForTask(arn string) (*ecs.ContainerDefinition, error) {
	res, err := p.ecs().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(arn),
	})
	if err != nil {
		return nil, err
	}
	if len(res.TaskDefinition.ContainerDefinitions) != 1 {
		return nil, fmt.Errorf("invalid container definitions for task: %s", arn)
	}

	return res.TaskDefinition.ContainerDefinitions[0], nil
}

func (p *AWSProvider) containerInstance(id string) (*ecs.ContainerInstance, error) {
	res, err := p.ecs().DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(p.Cluster),
		ContainerInstances: []*string{aws.String(id)},
	})
	if err != nil {
		return nil, err
	}
	if len(res.ContainerInstances) != 1 {
		return nil, fmt.Errorf("could not find container instance: %s", id)
	}

	return res.ContainerInstances[0], nil
}

func (p *AWSProvider) describeInstance(id string) (*ec2.Instance, error) {
	res, err := p.ec2().DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(id)},
	})
	if err != nil {
		return nil, err
	}
	if len(res.Reservations) != 1 || len(res.Reservations[0].Instances) != 1 {
		return nil, fmt.Errorf("could not find instance: %s", id)
	}

	return res.Reservations[0].Instances[0], nil
}

func (p *AWSProvider) describeTask(arn string) (*ecs.Task, error) {
	res, err := p.ecs().DescribeTasks(&ecs.DescribeTasksInput{
		Cluster: aws.String(p.Cluster),
		Tasks:   []*string{aws.String(arn)},
	})
	if err != nil {
		return nil, err
	}
	if len(res.Tasks) != 1 {
		return nil, fmt.Errorf("could not fetch process status")
	}
	return res.Tasks[0], nil
}

func (p *AWSProvider) fetchProcess(task *ecs.Task, psch chan structs.Process, errch chan error) {
	if len(task.Containers) != 1 {
		errch <- fmt.Errorf("invalid task: %s", *task.TaskDefinitionArn)
		return
	}

	cd, err := p.containerDefinitionForTask(*task.TaskDefinitionArn)
	if err != nil {
		errch <- err
		return
	}

	ci, err := p.containerInstance(*task.ContainerInstanceArn)
	if err != nil {
		errch <- err
		return
	}

	host, err := p.describeInstance(*ci.Ec2InstanceId)
	if err != nil {
		errch <- err
		return
	}

	env := map[string]string{}
	for _, e := range cd.Environment {
		env[*e.Name] = *e.Value
	}

	container := task.Containers[0]

	ports := []string{}
	for _, p := range container.NetworkBindings {
		ports = append(ports, fmt.Sprintf("%d:%d", *p.HostPort, *p.ContainerPort))
	}

	cmd := commandString(cd.Command)

	for _, o := range task.Overrides.ContainerOverrides {
		if o.Command != nil {
			cmd = commandString(o.Command)
		}
	}

	ps := structs.Process{
		ID: arnToPid(*task.TaskArn),
		// App:      app,
		Name:     *container.Name,
		Release:  env["RELEASE"],
		Command:  cmd,
		Host:     *host.PrivateIpAddress,
		Image:    *cd.Image,
		Instance: *ci.Ec2InstanceId,
		Ports:    ports,
	}

	// guard for nil
	if task.StartedAt != nil {
		ps.Started = *task.StartedAt
	}

	psch <- ps
}

func (p *AWSProvider) generateTaskDefinition(app, process, release string) (*ecs.RegisterTaskDefinitionInput, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	r, err := p.ReleaseGet(app, release)
	if err != nil {
		return nil, err
	}

	m, err := manifest.Load([]byte(r.Manifest))
	if err != nil {
		return nil, err
	}

	s, ok := m.Services[process]
	if !ok {
		return nil, fmt.Errorf("no such process: %s", process)
	}

	rs, err := p.describeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(fmt.Sprintf("%s-%s", p.Rack, app)),
	})
	if err != nil {
		return nil, err
	}

	sarn := ""
	sn := fmt.Sprintf("Service%s", upperName(process))

	for _, r := range rs.StackResources {
		if *r.LogicalResourceId == sn {
			sarn = *r.PhysicalResourceId
		}
	}
	if sarn == "" {
		return nil, fmt.Errorf("could not find service for process: %s", process)
	}

	sres, err := p.ecs().DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(p.Cluster),
		Services: []*string{aws.String(sarn)},
	})
	if err != nil {
		return nil, err
	}
	if len(sres.Services) != 1 {
		return nil, fmt.Errorf("could not look up service for process: %s", process)
	}

	tres, err := p.ecs().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: sres.Services[0].TaskDefinition,
	})
	if err != nil {
		return nil, err
	}
	if len(tres.TaskDefinition.ContainerDefinitions) < 1 {
		return nil, fmt.Errorf("could not find container definition for process: %s", process)
	}

	senv := map[string]string{}

	for _, e := range tres.TaskDefinition.ContainerDefinitions[0].Environment {
		senv[*e.Name] = *e.Value
	}

	cd := &ecs.ContainerDefinition{
		Essential:         aws.Bool(true),
		Image:             aws.String(fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s.%s", a.Outputs["RegistryId"], p.Region, a.Outputs["RegistryRepository"], process, r.Build)),
		MemoryReservation: aws.Int64(512),
		Name:              aws.String(process),
	}

	if len(s.Command.Array) > 0 {
		cd.Command = make([]*string, len(s.Command.Array))
		for i, c := range s.Command.Array {
			cd.Command[i] = aws.String(c)
		}
	} else if s.Command.String != "" {
		cd.Command = []*string{aws.String("sh"), aws.String("-c"), aws.String(s.Command.String)}
	}

	base := map[string]string{
		"APP":        app,
		"AWS_REGION": p.Region,
		"LOG_GROUP":  a.Outputs["LogGroup"],
		"PROCESS":    process,
		"RACK":       p.Rack,
		"RELEASE":    release,
	}

	for k, v := range base {
		cd.Environment = append(cd.Environment, &ecs.KeyValuePair{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}

	for k, v := range s.Environment {
		cd.Environment = append(cd.Environment, &ecs.KeyValuePair{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}

	for _, e := range strings.Split(r.Env, "\n") {
		p := strings.SplitN(e, "=", 2)
		if len(p) == 2 {
			cd.Environment = append(cd.Environment, &ecs.KeyValuePair{
				Name:  aws.String(p[0]),
				Value: aws.String(p[1]),
			})
		}
	}

	vars := []string{"SCHEME", "USERNAME", "PASSWORD", "HOST", "PORT", "PATH", "URL"}

	for _, l := range s.Links {
		prefix := strings.Replace(strings.ToUpper(l), "-", "_", -1)

		for _, v := range vars {
			k := fmt.Sprintf("%s_%s", prefix, v)

			lv, ok := senv[k]
			if !ok {
				return nil, fmt.Errorf("could not find link var: %s", k)
			}

			cd.Environment = append(cd.Environment, &ecs.KeyValuePair{
				Name:  aws.String(k),
				Value: aws.String(lv),
			})
		}
	}

	req := &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{cd},
		Family:               aws.String(fmt.Sprintf("%s-%s-%s", p.Rack, app, process)),
	}

	for i, mv := range s.MountableVolumes() {
		name := fmt.Sprintf("volume-%d", i)

		req.Volumes = append(req.Volumes, &ecs.Volume{
			Name: aws.String(name),
			Host: &ecs.HostVolumeProperties{
				SourcePath: aws.String(mv.Host),
			},
		})

		req.ContainerDefinitions[0].MountPoints = append(req.ContainerDefinitions[0].MountPoints, &ecs.MountPoint{
			SourceVolume:  aws.String(name),
			ContainerPath: aws.String(mv.Container),
			ReadOnly:      aws.Bool(false),
		})
	}

	return req, nil
}

func (p *AWSProvider) resolveRelease(app, release string) (string, error) {
	if release != "" {
		return release, nil
	}

	a, err := p.AppGet(app)
	if err != nil {
		return "", err
	}
	if a.Release == "" {
		return "", fmt.Errorf("no releases for app: %s", app)
	}

	return a.Release, nil
}

func (p *AWSProvider) taskArnFromPid(pid string) (string, error) {
	token := ""

	for {
		req := &ecs.ListTasksInput{
			Cluster: aws.String(p.Cluster),
		}
		if token != "" {
			req.NextToken = aws.String(token)
		}

		res, err := p.ecs().ListTasks(req)
		if err != nil {
			return "", err
		}

		for _, arn := range res.TaskArns {
			if arnToPid(*arn) == pid {
				return *arn, nil
			}
		}

		if res.NextToken == nil {
			break
		}

		token = *res.NextToken
	}

	return "", fmt.Errorf("could not find process")
}

func (p *AWSProvider) taskDefinitionsForPrefix(prefix string) ([]string, error) {
	tds := []string{}

	for {
		res, err := p.ecs().ListTaskDefinitions(&ecs.ListTaskDefinitionsInput{
			FamilyPrefix: aws.String(prefix),
		})
		if err != nil {
			return nil, err
		}

		for _, td := range res.TaskDefinitionArns {
			tds = append(tds, *td)
		}

		break
	}

	return tds, nil
}

func (p *AWSProvider) taskDefinitionForRun(app, process, release string) (string, error) {
	release, err := p.resolveRelease(app, release)
	if err != nil {
		return "", nil
	}

	item, err := p.fetchRelease(app, release)
	if err != nil {
		return "", err
	}

	var tasks map[string]string
	err = json.Unmarshal([]byte(coalesce(item["definitions"], "{}")), &tasks)
	if err != nil {
		return "", err
	}

	if task, ok := tasks[fmt.Sprintf("%s.run", process)]; ok {
		return task, nil
	}

	td, err := p.generateTaskDefinition(app, process, release)
	if err != nil {
		return "", err
	}

	res, err := p.ecs().RegisterTaskDefinition(td)
	if err != nil {
		return "", err
	}

	tasks[fmt.Sprintf("%s.run", process)] = *res.TaskDefinition.TaskDefinitionArn

	jtasks, err := json.Marshal(tasks)
	if err != nil {
		return "", err
	}

	_, err = p.dynamodb().UpdateItem(&dynamodb.UpdateItemInput{
		TableName: aws.String(p.DynamoReleases),
		Key: map[string]*dynamodb.AttributeValue{
			"id": &dynamodb.AttributeValue{S: aws.String(release)},
		},
		UpdateExpression: aws.String("set definitions = :definitions"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":definitions": &dynamodb.AttributeValue{S: aws.String(string(jtasks))},
		},
	})
	if err != nil {
		return "", err
	}

	return *res.TaskDefinition.TaskDefinitionArn, nil
}

func (p *AWSProvider) waitForTask(arn string) (string, error) {
	timeout := time.After(60 * time.Second)
	tick := time.Tick(1 * time.Second)

	for {
		select {
		case <-tick:
			task, err := p.describeTask(arn)
			if err != nil {
				return "", err
			}
			if status := *task.LastStatus; status != "PENDING" {
				return status, nil
			}
		case <-timeout:
			return "", fmt.Errorf("timeout starting process")
		}
	}
}
