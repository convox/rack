package aws

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/api/cache"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/manifest"
	"github.com/convox/rack/manifest1"
	"github.com/fsouza/go-dockerclient"
	shellquote "github.com/kballard/go-shellquote"
)

// StatusCodePrefix is sent to the client to let it know the exit code is coming next
const StatusCodePrefix = "F1E49A85-0AD7-4AEF-A618-C249C6E6568D:"

// ProcessExec runs a command in an existing Process
func (p *AWSProvider) ProcessExec(app, pid, command string, stream io.ReadWriter, opts structs.ProcessExecOptions) error {
	log := Logger.At("ProcessExec").Namespace("app=%q pid=%q command=%q", app, pid, command).Start()

	pss, err := p.ProcessList(app)
	if err != nil {
		log.Error(err)
		return err
	}

	pidFound := false
	for _, p := range pss {
		if p.ID == pid {
			pidFound = true
			break
		}
	}

	if !pidFound {
		return errorNotFound(fmt.Sprintf("process ID not found for %s", app))
	}

	arn, err := p.taskArnFromPid(pid)
	if err != nil {
		log.Error(err)
		return err
	}

	task, err := p.describeTask(arn)
	if err != nil {
		log.Error(err)
		return err
	}
	if len(task.Containers) < 1 {
		return log.Errorf("no running container for process: %s", pid)
	}

	cires, err := p.describeContainerInstances(&ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(p.Cluster),
		ContainerInstances: []*string{task.ContainerInstanceArn},
	})
	if err != nil {
		log.Error(err)
		return err
	}
	if len(cires.ContainerInstances) < 1 {
		return log.Errorf("could not find instance for process: %s", pid)
	}

	dc, err := p.dockerInstance(*cires.ContainerInstances[0].Ec2InstanceId)
	if err != nil {
		log.Error(err)
		return err
	}

	cs, err := dc.ListContainers(docker.ListContainersOptions{
		All: true,
		Filters: map[string][]string{
			"label": {fmt.Sprintf("com.amazonaws.ecs.task-arn=%s", arn)},
		},
	})
	if err != nil {
		log.Error(err)
		return err
	}
	if len(cs) != 1 {
		return log.Errorf("could not find container for task: %s", arn)
	}

	cmd := []string{"sh", "-c", command}

	a, err := p.AppGet(app)
	if err != nil {
		return err
	}

	if a.Tags["Generation"] == "2" {
		cmd = append([]string{"/convox-env"}, cmd...)
	}

	eres, err := dc.CreateExec(docker.CreateExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          cmd,
		Container:    cs[0].ID,
	})
	if err != nil {
		log.Error(err)
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
		InputStream:  ioutil.NopCloser(stream),
		OutputStream: stream,
		ErrorStream:  stream,
		RawTerminal:  true,
		Success:      success,
	})

	if err != nil {
		log.Error(err)
		return err
	}

	ires, err := dc.InspectExec(eres.ID)
	if err != nil {
		log.Error(err)
		return err
	}

	if _, err := stream.Write([]byte(fmt.Sprintf("%s%d\n", StatusCodePrefix, ires.ExitCode))); err != nil {
		log.Error(err)
		return err
	}

	log.Success()
	return nil
}

// ProcessGet returns the specified process for an app
func (p *AWSProvider) ProcessGet(app, pid string) (*structs.Process, error) {
	log := Logger.At("ProcessGet").Namespace("app=%q pid=%s", app, pid).Start()

	tasks, err := p.appTaskARNs(app)
	if err != nil {
		return nil, log.Error(err)
	}

	for _, t := range tasks {
		if strings.HasSuffix(t, pid) {
			pss, err := p.taskProcesses([]string{t})
			if err != nil {
				return nil, log.Error(err)
			}

			if len(pss) > 0 {
				return &pss[0], nil
			}
		}
	}

	return nil, log.Error(errorNotFound(fmt.Sprintf("no such process: %s", pid)))
}

// ProcessList returns a list of processes for an app
func (p *AWSProvider) ProcessList(app string) (structs.Processes, error) {
	log := Logger.At("ProcessList").Namespace("app=%q", app).Start()

	tasks, err := p.appTaskARNs(app)
	if err != nil {
		return nil, log.Error(err)
	}

	ps, err := p.taskProcesses(tasks)
	if err != nil {
		return nil, log.Error(err)
	}

	for i := range ps {
		ps[i].App = app
	}

	return ps, nil
}

func (p *AWSProvider) stackTasks(stack string) ([]string, error) {
	log := Logger.At("stackTasks").Namespace("stack=%q", stack).Start()

	rres, err := p.describeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(stack),
	})
	if err != nil {
		log.Error(err)
		return nil, err
	}

	services := []string{}

	for _, r := range rres.StackResources {
		switch *r.ResourceType {
		case "AWS::ECS::Service", "Custom::ECSService":
			services = append(services, *r.PhysicalResourceId)
		}
	}

	tasks := []string{}

	for _, s := range services {
		err := p.ecs().ListTasksPages(&ecs.ListTasksInput{
			Cluster:     aws.String(p.Cluster),
			ServiceName: aws.String(s),
		},
			func(page *ecs.ListTasksOutput, lastPage bool) bool {
				for _, arn := range page.TaskArns {
					tasks = append(tasks, *arn)
				}
				return true
			},
		)
		if err != nil {
			log.Error(err)
			return nil, err
		}
	}

	log.Success()
	return tasks, nil
}

// appTaskARNs retuns a list of ECS Task (aka process) ARNs that correspond to an app
// This includes one-off processes, build tasks, etc.
func (p *AWSProvider) appTaskARNs(app string) ([]string, error) {
	tasks, err := p.stackTasks(fmt.Sprintf("%s-%s", p.Rack, app))
	if ae, ok := err.(awserr.Error); ok && ae.Code() == "ValidationError" {
		return nil, errorNotFound(fmt.Sprintf("no such app: %s", app))
	}
	if err != nil {
		return nil, err
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
		tasks = append(tasks, *task)
	}

	// list build processes
	if p.Cluster != p.BuildCluster {
		ores, err := p.ecs().ListTasks(&ecs.ListTasksInput{
			Cluster:   aws.String(p.BuildCluster),
			StartedBy: aws.String(fmt.Sprintf("convox.%s", app)),
		})
		if err != nil {
			return nil, err
		}

		for _, task := range ores.TaskArns {
			tasks = append(tasks, *task)
		}
	}

	return tasks, nil
}

const describeTasksPageSize = 100

func (p *AWSProvider) taskProcesses(tasks []string) (structs.Processes, error) {
	log := Logger.At("taskProcesses").Namespace("tasks=%q", tasks).Start()

	pss := structs.Processes{}

	for i := 0; i < len(tasks); i += describeTasksPageSize {
		ptasks := []string{}

		if len(tasks) < i+describeTasksPageSize {
			ptasks = append(ptasks, tasks[i:]...)
		} else {
			ptasks = append(ptasks, tasks[i:i+describeTasksPageSize]...)
		}

		psst := structs.Processes{}
		psch := make(chan structs.Process, len(ptasks))
		errch := make(chan error)

		if len(ptasks) == 0 {
			return structs.Processes{}, nil
		}

		iptasks := make([]*string, len(ptasks))

		for i := range ptasks {
			iptasks[i] = &ptasks[i]
		}

		tres, err := p.describeTasks(&ecs.DescribeTasksInput{
			Cluster: aws.String(p.Cluster),
			Tasks:   iptasks,
		})
		if err != nil {
			log.Error(err)
			return nil, err
		}

		ecsTasks := make([]*ecs.Task, len(tres.Tasks))
		copy(ecsTasks, tres.Tasks)

		// list tasks on build cluster too
		if p.Cluster != p.BuildCluster {
			tres, err := p.describeTasks(&ecs.DescribeTasksInput{
				Cluster: aws.String(p.BuildCluster),
				Tasks:   iptasks,
			})
			if err != nil {
				log.Error(err)
				return nil, err
			}

			for _, task := range tres.Tasks {
				ecsTasks = append(ecsTasks, task)
			}
		}

		for _, task := range ecsTasks {
			if p.IsTest() {
				p.fetchProcess(task, psch, errch)
			} else {
				go p.fetchProcess(task, psch, errch)
			}
		}

		for i := 0; i < len(ecsTasks); i++ {
			select {
			case <-time.After(30 * time.Second):
				return nil, fmt.Errorf("timeout")
			case err := <-errch:
				return nil, err
			case ps := <-psch:
				psst = append(psst, ps)
			}
		}

		pss = append(pss, psst...)
	}

	instances, err := p.rackInstances()
	if err != nil {
		return nil, err
	}

	for i, ps := range pss {
		if inst, ok := instances[ps.Instance]; ok {
			ps.Host = *inst.PrivateIpAddress
			pss[i] = ps
		}
	}

	sort.Sort(pss)

	log.Success()
	return pss, nil
}

// ProcessRun runs a new Process
func (p *AWSProvider) ProcessRun(app, process string, opts structs.ProcessRunOptions) (string, error) {
	log := Logger.At("ProcessRun").Namespace("app=%q process=%q", app, process).Start()

	if opts.Stream != nil {
		return p.processRunAttached(app, process, opts)
	}

	td, err := p.taskDefinitionForRun(app, process, opts.Release)
	if err != nil {
		log.Error(err)
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
				{
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
		log.Error(err)
		return "", err
	}

	log.Success()
	return arnToPid(*task.TaskArn), nil
}

// ProcessStop stops a Process
func (p *AWSProvider) ProcessStop(app, pid string) error {
	log := Logger.At("ProcessStop").Namespace("app=%q pid=%q", app, pid).Start()

	arn, err := p.taskArnFromPid(pid)
	if err != nil {
		log.Error(err)
		return err
	}

	_, err = p.ecs().StopTask(&ecs.StopTaskInput{
		Cluster: aws.String(p.Cluster),
		Task:    aws.String(arn),
	})
	if err != nil {
		log.Error(err)
		return err
	}

	log.Success()
	return nil
}

func arnToPid(arn string) string {
	parts := strings.Split(arn, "-")
	return parts[len(parts)-1]
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

func commandString(cs []string) string {
	cmd := make([]string, len(cs))

	for i, c := range cs {
		if strings.Contains(c, " ") {
			cmd[i] = fmt.Sprintf("%q", c)
		} else {
			cmd[i] = c
		}
	}

	return strings.Join(cmd, " ")
}

func (p *AWSProvider) containerDefinitionForTask(arn string) (*ecs.ContainerDefinition, error) {
	cd, ok := cache.Get("containerDefinitionForTask", arn).(*ecs.ContainerDefinition)
	if ok {
		return cd, nil
	}

	res, err := p.describeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(arn),
	})
	if err != nil {
		return nil, err
	}
	if len(res.TaskDefinition.ContainerDefinitions) < 1 {
		return nil, fmt.Errorf("invalid container definitions for task: %s", arn)
	}

	if !p.SkipCache {
		if err := cache.Set("containerDefinitionForTask", arn, res.TaskDefinition.ContainerDefinitions[0], 10*time.Second); err != nil {
			return nil, err
		}
	}

	return res.TaskDefinition.ContainerDefinitions[0], nil
}

func (p *AWSProvider) containerInstance(id string) (*ecs.ContainerInstance, error) {
	ci, ok := cache.Get("containerInstance", id).(*ecs.ContainerInstance)
	if ok {
		return ci, nil
	}

	res, err := p.describeContainerInstances(&ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(p.Cluster),
		ContainerInstances: []*string{aws.String(id)},
	})
	if err != nil {
		return nil, err
	}
	// check the build cluster too
	for _, f := range res.Failures {
		if f.Reason != nil && *f.Reason == "MISSING" && p.BuildCluster != p.Cluster {
			res, err = p.describeContainerInstances(&ecs.DescribeContainerInstancesInput{
				Cluster:            aws.String(p.BuildCluster),
				ContainerInstances: []*string{aws.String(id)},
			})
			if err != nil {
				return nil, err
			}
			break
		}
	}

	if len(res.ContainerInstances) != 1 {
		return nil, fmt.Errorf("could not find container instance: %s", id)
	}

	if !p.SkipCache {
		if err := cache.Set("containerInstance", id, res.ContainerInstances[0], 10*time.Second); err != nil {
			return nil, err
		}
	}

	return res.ContainerInstances[0], nil
}

func (p *AWSProvider) describeInstance(id string) (*ec2.Instance, error) {
	instance, ok := cache.Get("describeInstance", id).(*ec2.Instance)
	if ok {
		return instance, nil
	}

	res, err := p.ec2().DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(id)},
	})
	if err != nil {
		return nil, err
	}
	if len(res.Reservations) != 1 || len(res.Reservations[0].Instances) != 1 {
		return nil, fmt.Errorf("could not find instance: %s", id)
	}

	if !p.SkipCache {
		if err := cache.Set("describeInstance", id, res.Reservations[0].Instances[0], 10*time.Second); err != nil {
			return nil, err
		}
	}

	return res.Reservations[0].Instances[0], nil
}

func (p *AWSProvider) describeTask(arn string) (*ecs.Task, error) {
	res, err := p.describeTasks(&ecs.DescribeTasksInput{
		Cluster: aws.String(p.Cluster),
		Tasks:   []*string{aws.String(arn)},
	})
	// check the build cluster too
	for _, f := range res.Failures {
		if f.Reason != nil && *f.Reason == "MISSING" && p.BuildCluster != p.Cluster {
			res, err = p.describeTasks(&ecs.DescribeTasksInput{
				Cluster: aws.String(p.BuildCluster),
				Tasks:   []*string{aws.String(arn)},
			})
			break
		}
	}
	if err != nil {
		return nil, err
	}
	if len(res.Tasks) != 1 {
		return nil, fmt.Errorf("could not fetch process status")
	}
	return res.Tasks[0], nil
}

func (p *AWSProvider) fetchProcess(task *ecs.Task, psch chan structs.Process, errch chan error) {
	if len(task.Containers) < 1 {
		errch <- fmt.Errorf("invalid task: %s", *task.TaskDefinitionArn)
		return
	}

	cd, err := p.containerDefinitionForTask(*task.TaskDefinitionArn)
	if err != nil {
		errch <- err
		return
	}

	instance := ""

	if task.ContainerInstanceArn != nil {
		ci, err := p.containerInstance(*task.ContainerInstanceArn)
		if err != nil {
			errch <- err
			return
		}

		if ci.Ec2InstanceId != nil {
			instance = *ci.Ec2InstanceId
		}
	}

	env := map[string]string{}
	for _, e := range cd.Environment {
		env[*e.Name] = *e.Value
	}

	for _, o := range task.Overrides.ContainerOverrides {
		for _, p := range o.Environment {
			env[*p.Name] = *p.Value
		}
	}

	labels := map[string]string{}
	for k, v := range cd.DockerLabels {
		labels[k] = *v
	}

	container := task.Containers[0]

	ports := []string{}
	for _, p := range container.NetworkBindings {
		ports = append(ports, fmt.Sprintf("%d:%d", *p.HostPort, *p.ContainerPort))
	}

	ps := structs.Process{
		ID:       arnToPid(*task.TaskArn),
		Name:     *container.Name,
		App:      coalesces(labels["convox.app"], env["APP"]),
		Release:  coalesces(labels["convox.release"], env["RELEASE"]),
		Image:    *cd.Image,
		Instance: instance,
		Ports:    ports,
	}

	// guard for nil
	if task.StartedAt != nil {
		ps.Started = *task.StartedAt
	}

	if len(cd.Command) > 0 {
		p := make([]string, len(cd.Command))

		for i, c := range cd.Command {
			p[i] = *c
		}

		ps.Command = shellquote.Join(p...)
	}

	// TODO: figure out a way to do this less expensively

	// dc, err := p.dockerInstance(ps.Instance)
	// if err != nil {
	//   errch <- err
	//   return
	// }

	// sch := make(chan *docker.Stats, 1)
	// err = dc.Stats(docker.StatsOptions{
	//   ID:     cs[0].ID,
	//   Stats:  sch,
	//   Stream: false,
	// })
	// if err != nil {
	//   fmt.Printf("docker stats error: %s", err)
	//   psch <- ps
	// }

	// stat := <-sch
	// if stat != nil {
	//   pcpu := stat.PreCPUStats.CPUUsage.TotalUsage
	//   psys := stat.PreCPUStats.SystemCPUUsage

	//   ps.CPU = truncate(calculateCPUPercent(pcpu, psys, stat), 4)

	//   if stat.MemoryStats.Limit > 0 {
	//     ps.Memory = truncate(float64(stat.MemoryStats.Usage)/float64(stat.MemoryStats.Limit), 4)
	//   }
	// }

	psch <- ps
}

func (p *AWSProvider) generateTaskDefinition1(app, process, release string) (*ecs.RegisterTaskDefinitionInput, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	r, err := p.ReleaseGet(app, release)
	if err != nil {
		return nil, err
	}

	m, err := manifest1.Load([]byte(r.Manifest))
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

	secureEnvRoleName := ""

	for _, r := range rs.StackResources {
		if *r.LogicalResourceId == sn {
			sarn = *r.PhysicalResourceId
		}
		if *r.LogicalResourceId == "SecureEnvironmentRole" {
			secureEnvRoleName = *r.PhysicalResourceId
		}
	}
	if sarn == "" {
		return nil, fmt.Errorf("could not find service for process: %s", process)
	}
	if secureEnvRoleName == "" && s.UseSecureEnvironment() {
		return nil, fmt.Errorf("cound not find secure environment role for process: %s", process)
	}

	sres, err := p.describeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(p.Cluster),
		Services: []*string{aws.String(sarn)},
	})
	if err != nil {
		return nil, err
	}
	if len(sres.Services) != 1 {
		return nil, fmt.Errorf("could not look up service for process: %s", process)
	}

	tres, err := p.describeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
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
		DockerLabels: map[string]*string{
			"convox.process.type": aws.String("oneoff"),
		},
		Essential:         aws.Bool(true),
		Image:             aws.String(fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s.%s", a.Outputs["RegistryId"], p.Region, a.Outputs["RegistryRepository"], process, r.Build)),
		MemoryReservation: aws.Int64(512),
		Name:              aws.String(process),
		Privileged:        aws.Bool(s.Privileged),
	}

	if len(s.Command.Array) > 0 {
		cd.Command = make([]*string, len(s.Command.Array))
		for i, c := range s.Command.Array {
			cd.Command[i] = aws.String(c)
		}
	} else if s.Command.String != "" {
		cd.Command = []*string{aws.String("sh"), aws.String("-c"), aws.String(s.Command.String)}
	}

	env := map[string]string{}

	base := map[string]string{
		"APP":        app,
		"AWS_REGION": p.Region,
		"LOG_GROUP":  a.Outputs["LogGroup"],
		"PROCESS":    process,
		"RACK":       p.Rack,
		"RELEASE":    release,
	}

	for k, v := range base {
		env[k] = v
	}

	for _, e := range s.Environment {
		env[e.Name] = e.Value
	}

	if s.UseSecureEnvironment() {
		env["SECURE_ENVIRONMENT_URL"] = a.Parameters["Environment"]
		env["SECURE_ENVIRONMENT_TYPE"] = "envfile"
		env["SECURE_ENVIRONMENT_KEY"] = a.Parameters["Key"]
	} else {
		for _, e := range strings.Split(r.Env, "\n") {
			p := strings.SplitN(e, "=", 2)
			if len(p) == 2 {
				env[p[0]] = p[1]
			}
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

			env[k] = lv
		}
	}

	keys := []string{}

	for k := range env {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		cd.Environment = append(cd.Environment, &ecs.KeyValuePair{
			Name:  aws.String(k),
			Value: aws.String(env[k]),
		})
	}

	tr := a.Parameters["TaskRole"]

	if secureEnvRoleName != "" && s.UseSecureEnvironment() {
		tr = fmt.Sprintf("convox/%s", secureEnvRoleName)
	}

	req := &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{cd},
		Family:               aws.String(fmt.Sprintf("%s-%s-%s", p.Rack, app, process)),
		TaskRoleArn:          &tr,
	}

	for i, mv := range s.MountableVolumes() {
		name := fmt.Sprintf("volume-%d", i)
		host := fmt.Sprintf("/volumes/%s-%s/%s%s", p.Rack, app, process, mv.Host)

		req.Volumes = append(req.Volumes, &ecs.Volume{
			Name: aws.String(name),
			Host: &ecs.HostVolumeProperties{
				SourcePath: aws.String(host),
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

func (p *AWSProvider) generateTaskDefinition2(app, process, release string) (*ecs.RegisterTaskDefinitionInput, error) {
	r, err := p.ReleaseGet(app, release)
	if err != nil {
		return nil, err
	}

	env := structs.Environment{}
	env.LoadEnvironment([]byte(r.Env))

	m, err := manifest.Load([]byte(r.Manifest), manifest.Environment(env))
	if err != nil {
		return nil, err
	}

	s, err := m.Service(process)
	if err != nil {
		return nil, err
	}

	sarn, err := p.appResource(app, fmt.Sprintf("Service%sService", upperName(process)))
	if err != nil {
		return nil, err
	}

	sres, err := p.describeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(p.Cluster),
		Services: []*string{aws.String(sarn)},
	})
	if err != nil {
		return nil, err
	}
	if len(sres.Services) != 1 {
		return nil, fmt.Errorf("could not look up service for process: %s", process)
	}

	tres, err := p.describeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: sres.Services[0].TaskDefinition,
	})
	if err != nil {
		return nil, err
	}
	if len(tres.TaskDefinition.ContainerDefinitions) < 1 {
		return nil, fmt.Errorf("could not find container definition for process: %s", process)
	}

	ocd := tres.TaskDefinition.ContainerDefinitions[0]

	labels := ocd.DockerLabels
	labels["convox.process.type"] = aws.String("oneoff")

	aid, err := p.accountId()
	if err != nil {
		return nil, err
	}

	reg, err := p.appResource(app, "Registry")
	if err != nil {
		return nil, err
	}

	settings, err := p.appResource(app, "Settings")
	if err != nil {
		return nil, err
	}

	senv := s.EnvironmentDefaults()

	for _, e := range ocd.Environment {
		senv[*e.Name] = *e.Value
	}

	senv["CONVOX_ENV_URL"] = fmt.Sprintf("s3://%s/releases/%s/env", settings, release)

	cenv := []*ecs.KeyValuePair{}

	for k, v := range senv {
		cenv = append(cenv, &ecs.KeyValuePair{Name: aws.String(k), Value: aws.String(v)})
	}

	cd := &ecs.ContainerDefinition{
		Command:           []*string{aws.String("sh"), aws.String("-c"), aws.String(s.Command)},
		DockerLabels:      labels,
		Environment:       cenv,
		Essential:         aws.Bool(true),
		Image:             aws.String(fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s.%s", aid, p.Region, reg, process, r.Build)),
		MemoryReservation: aws.Int64(512),
		MountPoints:       tres.TaskDefinition.ContainerDefinitions[0].MountPoints,
		Name:              aws.String(process),
		Privileged:        aws.Bool(s.Privileged),
	}

	req := &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{cd},
		Family:               aws.String(fmt.Sprintf("%s-%s-%s", p.Rack, app, process)),
		TaskRoleArn:          tres.TaskDefinition.TaskRoleArn,
		Volumes:              tres.TaskDefinition.Volumes,
	}

	return req, nil
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
				{
					Name: aws.String(process),
					Command: []*string{
						aws.String("sleep"),
						aws.String("3600"),
					},
				},
			},
		}
	}

	task, err := p.runTask(req)
	if err != nil {
		return "", err
	}

	defer p.stopTask(*task.TaskArn)

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
	if err != nil && !strings.Contains(err.Error(), "use of closed network") {
		return "", err
	}

	return pid, nil
}

func (p *AWSProvider) rackInstances() (map[string]ec2.Instance, error) {
	req := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("tag:Rack"),
				Values: []*string{aws.String(os.Getenv("RACK"))},
			},
		},
	}

	instances := map[string]ec2.Instance{}

	err := p.ec2().DescribeInstancesPages(req, func(res *ec2.DescribeInstancesOutput, last bool) bool {
		for _, r := range res.Reservations {
			for _, i := range r.Instances {
				instances[*i.InstanceId] = *i
			}
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	return instances, nil
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

func (p *AWSProvider) stopTask(arn string) error {
	_, err := p.ecs().StopTask(&ecs.StopTaskInput{
		Cluster: aws.String(p.Cluster),
		Task:    aws.String(arn),
	})

	return err
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

	a, err := p.AppGet(app)
	if err != nil {
		return "", err
	}

	var td *ecs.RegisterTaskDefinitionInput

	switch a.Tags["Generation"] {
	case "2":
		td, err = p.generateTaskDefinition2(app, process, release)
		if err != nil {
			return "", err
		}
	default:
		td, err = p.generateTaskDefinition1(app, process, release)
		if err != nil {
			return "", err
		}
	}

	// if a.Tags["Generation"] == "2

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
			"id": {S: aws.String(release)},
		},
		UpdateExpression: aws.String("set definitions = :definitions"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":definitions": {S: aws.String(string(jtasks))},
		},
	})
	if err != nil {
		return "", err
	}

	return *res.TaskDefinition.TaskDefinitionArn, nil
}

// truncat a float to a given precision
// ex:  truncate(3.1459, 2) -> 3.14
func truncate(f float64, precision int) float64 {
	p := math.Pow10(precision)
	return float64(int(f*p)) / p
}

func (p *AWSProvider) waitForTask(arn string) (string, error) {
	timeout := time.After(300 * time.Second)
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
