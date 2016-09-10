package models

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/client"
	"github.com/convox/rack/manifest"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/fsouza/go-dockerclient"
)

var CustomTopic = os.Getenv("CUSTOM_TOPIC")

var StatusCodePrefix = client.StatusCodePrefix

type App struct {
	Name    string `json:"name"`
	Release string `json:"release"`
	Status  string `json:"status"`

	Outputs    map[string]string `json:"-"`
	Parameters map[string]string `json:"-"`
	Tags       map[string]string `json:"-"`
}

type Apps []App

func ListApps() (Apps, error) {
	res, err := DescribeStacks()

	if err != nil {
		return nil, err
	}

	apps := make(Apps, 0)

	for _, stack := range res.Stacks {
		tags := stackTags(stack)

		if tags["System"] == "convox" && tags["Type"] == "app" {
			if tags["Rack"] == "" || tags["Rack"] == os.Getenv("RACK") {
				apps = append(apps, *appFromStack(stack))
			}
		}
	}

	return apps, nil
}

func GetApp(name string) (*App, error) {
	stackName := shortNameToStackName(name)
	app, err := getAppByStackName(stackName)

	if name != stackName && awsError(err) == "ValidationError" {
		// Only lookup an unbound app if the name/stackName differ and the
		// bound lookup fails.
		app, err = getAppByStackName(name)
	}

	if app != nil {
		if app.Tags["Rack"] != "" && app.Tags["Rack"] != os.Getenv("RACK") {
			return nil, fmt.Errorf("no such app: %s", name)

		} else if len(app.Tags) == 0 && name != os.Getenv("RACK") {
			// This checks for a rack. An app with zero tags is a rack (this assumption should be addressed).
			// Makes sure the name equals current rack name; otherwise error out.
			return nil, fmt.Errorf("invalid rack: %s", name)
		}
	}

	return app, err
}

func GetAppBound(name string) (*App, error) {
	return getAppByStackName(shortNameToStackName(name))
}

func GetAppUnbound(name string) (*App, error) {
	return getAppByStackName(name)
}

func getAppByStackName(stackName string) (*App, error) {
	res, err := DescribeStack(stackName)

	if err != nil {
		return nil, err
	}

	app := appFromStack(res.Stacks[0])

	return app, nil
}

var regexValidAppName = regexp.MustCompile(`\A[a-zA-Z][-a-zA-Z0-9]{3,29}\z`)

func (a *App) IsBound() bool {
	if a.Tags == nil {
		// Default to bound.
		return true
	}

	if _, ok := a.Tags["Name"]; ok {
		// Bound apps MUST have a "Name" tag.
		return true
	}

	// Tags are present but "Name" tag is not, so we have an unbound app.
	return false
}

// StackName returns the app's stack if the app is bound. Otherwise returns the short name.
func (a *App) StackName() string {
	if a.IsBound() {
		return shortNameToStackName(a.Name)
	}

	return a.Name
}

func (a *App) Create() error {
	helpers.TrackEvent("kernel-app-create-start", nil)

	if !regexValidAppName.MatchString(a.Name) {
		return fmt.Errorf("app name can contain only alphanumeric characters and dashes and must be between 4 and 30 characters")
	}

	m := manifest.Manifest{
		Services: make(map[string]manifest.Service),
	}

	formation, err := a.Formation(m)
	if err != nil {
		helpers.TrackEvent("kernel-app-create-error", nil)
		return err
	}

	// SubnetsPrivate is a List<AWS::EC2::Subnet::Id> and can not be empty
	// So reuse SUBNETS if SUBNETS_PRIVATE is not set
	subnetsPrivate := os.Getenv("SUBNETS_PRIVATE")
	if subnetsPrivate == "" {
		subnetsPrivate = os.Getenv("SUBNETS")
	}

	params := map[string]string{
		"Cluster":        os.Getenv("CLUSTER"),
		"Internal":       os.Getenv("INTERNAL"),
		"Private":        os.Getenv("PRIVATE"),
		"Subnets":        os.Getenv("SUBNETS"),
		"SubnetsPrivate": subnetsPrivate,
		"Version":        os.Getenv("RELEASE"),
		"VPC":            os.Getenv("VPC"),
		"VPCCIDR":        os.Getenv("VPCCIDR"),
	}

	if os.Getenv("ENCRYPTION_KEY") != "" {
		params["Key"] = os.Getenv("ENCRYPTION_KEY")
	}

	tags := map[string]string{
		"Rack":   os.Getenv("RACK"),
		"System": "convox",
		"Type":   "app",
		"Name":   a.Name,
	}

	req := &cloudformation.CreateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(a.StackName()),
		TemplateBody: aws.String(formation),
	}

	for key, value := range params {
		req.Parameters = append(req.Parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String(key),
			ParameterValue: aws.String(value),
		})
	}

	for key, value := range tags {
		req.Tags = append(req.Tags, &cloudformation.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	_, err = CloudFormation().CreateStack(req)

	if err != nil {
		helpers.TrackEvent("kernel-app-create-error", nil)
		return err
	}

	helpers.TrackEvent("kernel-app-create-success", nil)

	NotifySuccess("app:create", map[string]string{"name": a.Name})

	return nil
}

func (a *App) Delete() error {
	helpers.TrackEvent("kernel-app-delete-start", nil)

	err := Provider().AppDelete(a.Name)
	if err != nil {
		return err
	}

	NotifySuccess("app:delete", map[string]string{"name": a.Name})

	return nil
}

// Shortcut for updating current parameters
// If template changed, more care about new or removed parameters must be taken (see Release.Promote or System.Save)
func (a *App) UpdateParams(changes map[string]string) error {
	req := &cloudformation.UpdateStackInput{
		StackName:           aws.String(a.StackName()),
		Capabilities:        []*string{aws.String("CAPABILITY_IAM")},
		UsePreviousTemplate: aws.Bool(true),
	}

	// sort parameters by key name to make test requests stable
	var keys []string
	for key := range a.Parameters {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		if updatedValue, present := changes[key]; present {
			req.Parameters = append(req.Parameters, &cloudformation.Parameter{
				ParameterKey:   aws.String(key),
				ParameterValue: aws.String(updatedValue),
			})
		} else {
			req.Parameters = append(req.Parameters, &cloudformation.Parameter{
				ParameterKey:     aws.String(key),
				UsePreviousValue: aws.Bool(true),
			})
		}
	}

	_, err := UpdateStack(req)

	return err
}

func (a *App) Formation(m manifest.Manifest) (string, error) {
	tmplData := map[string]interface{}{
		"App":      a,
		"Manifest": m,
	}
	data, err := buildTemplate("app", "app", tmplData)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (a *App) ForkRelease() (*Release, error) {
	release := structs.NewRelease(a.Name)

	if a.Release != "" {
		r, err := Provider().ReleaseGet(a.Name, a.Release)
		if err != nil {
			return nil, err
		}
		release = r
	}

	release.Id = generateId("R", 10)
	release.Created = time.Now()

	env, err := Provider().EnvironmentGet(a.Name)
	if err != nil {
		fmt.Printf("fn=ForkRelease level=error msg=\"error getting environment: %s\"", err)
	}

	release.Env = env.Raw()

	return &Release{
		Id:       release.Id,
		App:      release.App,
		Build:    release.Build,
		Env:      release.Env,
		Manifest: release.Manifest,
		Created:  release.Created,
	}, nil
}

func (a *App) ExecAttached(pid, command string, height, width int, rw io.ReadWriter) error {
	var ps Process

	pss, err := ListProcesses(a.Name)
	if err != nil {
		return err
	}

	for _, p := range pss {
		if p.Id == pid {
			ps = *p
			break
		}
	}

	if ps.Id == "" {
		return fmt.Errorf("no such process id: %s", pid)
	}

	d, err := ps.Docker()
	if err != nil {
		return err
	}

	res, err := d.CreateExec(docker.CreateExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{"sh", "-c", command},
		Container:    ps.containerId,
	})
	if err != nil {
		return err
	}

	id := res.ID

	// Create pipes so StartExec closes pipes, not the websocket.
	ir, iw := io.Pipe()
	or, ow := io.Pipe()

	go io.Copy(iw, rw)
	go io.Copy(rw, or)

	success := make(chan struct{})

	go func() {
		<-success
		d.ResizeExecTTY(id, height, width)
		success <- struct{}{}
	}()

	err = d.StartExec(res.ID, docker.StartExecOptions{
		Detach:       false,
		Tty:          true,
		InputStream:  ir,
		OutputStream: ow,
		ErrorStream:  ow,
		RawTerminal:  true,
		Success:      success,
	})
	// comparing with io.ErrClosedPipe isn't working
	if err != nil && !strings.HasSuffix(err.Error(), "closed pipe") {
		return err
	}

	ires, err := d.InspectExec(res.ID)
	if err != nil {
		return err
	}

	_, err = rw.Write([]byte(fmt.Sprintf("%s%d\n", StatusCodePrefix, ires.ExitCode)))
	return err
}

// RunAttached runs a command in the foreground (e.g blocking) and writing the output from said command to rw.
func (a *App) RunAttached(process, command, releaseID string, height, width int, rw io.ReadWriter) error {
	//TODO: A lot of logic in here should be moved to the provider interface.

	resources, err := a.Resources()
	if err != nil {
		return err
	}

	if releaseID == "" {
		releaseID = a.Release
	}

	release, err := GetRelease(a.Name, releaseID)
	if err != nil {
		return err
	}

	var container *ecs.ContainerDefinition
	unpromotedRelease := false

	task, err := ECS().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(resources[UpperName(process)+"ECSTaskDefinition"].Id),
	})
	if err != nil {
		return err
	}

	for _, container = range task.TaskDefinition.ContainerDefinitions {
		if *container.Name == process {
			break
		}
	}

	// This would force an app to be promoted once before being able to run a process.
	if container == nil {
		return fmt.Errorf("unable to find container for %s", process)
	}

	// If the release ID provided does not equal the active one, some logic is needed to determine the next steps.
	// - For a previous release, we iterate over the previous TaskDefinition revisions (starting with the latest) looking for the releaseID specified.
	// - If the release has yet to be promoted, we use the most recent TaskDefinition with the provided release's environment.
	if releaseID != a.Release {

		_, releaseContainer, err := findAppDefinitions(process, releaseID, *task.TaskDefinition.Family, 20)
		if err != nil {
			return err
		}

		// If container is nil, the release most likely hasn't been promoted and thus no TaskDefinition for it.
		if releaseContainer != nil {
			container = releaseContainer

		} else {
			fmt.Printf("Unable to find container for %s. Basing container off of most recent release: %s.\n", process, a.Release)
			unpromotedRelease = true
		}
	}

	var rawEnvs []string
	for _, env := range container.Environment {
		rawEnvs = append(rawEnvs, fmt.Sprintf("%s=%s", *env.Name, *env.Value))
	}
	containerEnvs := structs.Environment{}
	containerEnvs.LoadRaw(strings.Join(rawEnvs, "\n"))

	// Update any environment variables that might be part of the unpromoted release.
	if unpromotedRelease {

		releaseEnv := structs.Environment{}
		releaseEnv.LoadRaw(release.Env)

		for key, value := range releaseEnv {
			containerEnvs[key] = value
		}

		containerEnvs["RELEASE"] = release.Id
	}

	m, err := manifest.Load([]byte(release.Manifest))
	if err != nil {
		return err
	}

	me, ok := m.Services[process]
	if !ok {
		return fmt.Errorf("no such process: %s", process)
	}

	binds := []string{}
	host := ""

	pss, err := ListProcesses(a.Name)
	if err != nil {
		return err
	}

	for _, ps := range pss {
		if ps.Name == process {
			binds = ps.binds
			host = fmt.Sprintf("http://%s:2376", ps.Host)
			break
		}
	}

	var image, repository, tag, username, password, serverAddress string

	if registryId := a.Outputs["RegistryId"]; registryId != "" {
		image = fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s.%s", registryId, os.Getenv("AWS_REGION"), a.Outputs["RegistryRepository"], me.Name, release.Build)
		repository = fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s", registryId, os.Getenv("AWS_REGION"), a.Outputs["RegistryRepository"])
		tag = fmt.Sprintf("%s.%s", me.Name, release.Build)

		resp, err := ECR().GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{
			RegistryIds: []*string{aws.String(a.Outputs["RegistryId"])},
		})

		if err != nil {
			return err
		}

		if len(resp.AuthorizationData) < 1 {
			return fmt.Errorf("no authorization data")
		}

		endpoint := *resp.AuthorizationData[0].ProxyEndpoint
		serverAddress = endpoint[8:]

		data, err := base64.StdEncoding.DecodeString(*resp.AuthorizationData[0].AuthorizationToken)

		if err != nil {
			return err
		}

		parts := strings.SplitN(string(data), ":", 2)

		username = parts[0]
		password = parts[1]
	} else {
		image = fmt.Sprintf("%s/%s-%s:%s", os.Getenv("REGISTRY_HOST"), a.Name, me.Name, release.Build)
		repository = fmt.Sprintf("%s/%s-%s", os.Getenv("REGISTRY_HOST"), a.Name, me.Name)
		tag = release.Build
		serverAddress = os.Getenv("REGISTRY_HOST")
		username = "convox"
		password = os.Getenv("PASSWORD")
	}

	d, err := Docker(host)
	if err != nil {
		return err
	}

	err = d.PullImage(docker.PullImageOptions{
		Repository: repository,
		Tag:        tag,
	}, docker.AuthConfiguration{
		ServerAddress: serverAddress,
		Username:      username,
		Password:      password,
	})
	if err != nil {
		return err
	}

	res, err := d.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Env:          containerEnvs.List(),
			OpenStdin:    true,
			Tty:          true,
			Cmd:          []string{"sh", "-c", command},
			Image:        image,
			Labels: map[string]string{
				"com.convox.rack.type":    "oneoff",
				"com.convox.rack.app":     a.Name,
				"com.convox.rack.process": process,
				"com.convox.rack.release": release.Id,
			},
		},
		HostConfig: &docker.HostConfig{
			Binds: binds,
		},
	})
	if err != nil {
		return err
	}

	ir, iw := io.Pipe()
	or, ow := io.Pipe()

	go d.AttachToContainer(docker.AttachToContainerOptions{
		Container:    res.ID,
		InputStream:  ir,
		OutputStream: ow,
		ErrorStream:  ow,
		Stream:       true,
		Stdin:        true,
		Stdout:       true,
		Stderr:       true,
		RawTerminal:  true,
	})

	go io.Copy(iw, rw)
	go io.Copy(rw, or)

	// hacky
	time.Sleep(100 * time.Millisecond)

	err = d.StartContainer(res.ID, nil)
	if err != nil {
		return err
	}

	err = d.ResizeContainerTTY(res.ID, height, width)
	if err != nil {
		// In some cases, a container might finish and exit by the time ResizeContainerTTY is called.
		// Resizing the TTY shouldn't cause the call to error out for cases like that.
		fmt.Printf("fn=RunAttached level=warning msg=\"unable to resize container: %s\"", err)
	}

	code, err := d.WaitContainer(res.ID)
	if err != nil {
		return err
	}

	_, err = rw.Write([]byte(fmt.Sprintf("%s%d\n", StatusCodePrefix, code)))
	return err
}

// RunDetached runs a command in the background (e.g. non-blocking).
func (a *App) RunDetached(process, command, releaseID string) error {
	resources, err := a.Resources()
	if err != nil {
		return err
	}

	taskDefinitionArn := resources[UpperName(process)+"ECSTaskDefinition"].Id

	if releaseID == "" {
		releaseID = a.Release
	}

	release, err := GetRelease(a.Name, releaseID)
	if err != nil {
		return err
	}

	// If the releaseID specified isn't the app's current release:
	// - We have to find the right task definition OR
	// - create a new/temp task definition to run a release that hasn't been promoted.
	if releaseID != a.Release {
		task, err := ECS().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: aws.String(taskDefinitionArn),
		})
		if err != nil {
			return err
		}

		td, _, err := findAppDefinitions(process, releaseID, *task.TaskDefinition.Family, 20)
		if err != nil {
			return err

		} else if td != nil {
			taskDefinitionArn = *td.TaskDefinitionArn

		} else {
			// If reached, the release exist but doesn't have a task definition (isn't promoted).
			// Create a task definition to run that release.

			var cd *ecs.ContainerDefinition
			for _, cd = range task.TaskDefinition.ContainerDefinitions {
				if *cd.Name == process {
					break
				}
				cd = nil
			}
			if cd == nil {
				return fmt.Errorf("unable to find container for process %s and release %s", process, releaseID)
			}

			env := structs.Environment{}
			env.LoadRaw(release.Env)

			for _, containerKV := range cd.Environment {
				for key, value := range env {

					if *containerKV.Name == "RELEASE" {
						*containerKV.Value = releaseID
						break

					}

					if *containerKV.Name == key {
						*containerKV.Value = value
						break
					}
				}
			}

			taskInput := &ecs.RegisterTaskDefinitionInput{
				ContainerDefinitions: []*ecs.ContainerDefinition{
					cd,
				},
				Family:  task.TaskDefinition.Family,
				Volumes: []*ecs.Volume{},
			}

			resp, err := ECS().RegisterTaskDefinition(taskInput)
			if err != nil {
				return err
			}

			taskDefinitionArn = *resp.TaskDefinition.TaskDefinitionArn
		}
	}

	req := &ecs.RunTaskInput{
		Cluster:        aws.String(os.Getenv("CLUSTER")),
		Count:          aws.Int64(1),
		StartedBy:      aws.String("convox"),
		TaskDefinition: aws.String(taskDefinitionArn),
	}

	if command != "" {
		req.Overrides = &ecs.TaskOverride{
			ContainerOverrides: []*ecs.ContainerOverride{
				&ecs.ContainerOverride{
					Name: aws.String(process),
					Command: []*string{
						aws.String("sh"),
						aws.String("-c"),
						aws.String(command),
					},
				},
			},
		}
	}

	_, err = ECS().RunTask(req)

	return err
}

func (a *App) TaskDefinitionFamily() string {
	return a.Name
}

func (a *App) BalancerHost() string {
	return a.Outputs["BalancerHost"]
}

func (a *App) BalancerPorts(ps string) map[string]string {
	host := a.BalancerHost()

	bp := map[string]string{}

	for original, current := range a.ProcessPorts(ps) {
		bp[original] = fmt.Sprintf("%s:%s", host, current)
	}

	return bp
}

func (a *App) ProcessPorts(ps string) map[string]string {
	ports := map[string]string{}

	for key, value := range a.Outputs {
		r := regexp.MustCompile(fmt.Sprintf("%sPort([0-9]+)Balancer", UpperName(ps)))

		if matches := r.FindStringSubmatch(key); len(matches) == 2 {
			ports[matches[1]] = value
		}
	}

	return ports
}

func (a *App) Resources() (Resources, error) {
	resources, err := ListResources(a.Name)
	if err != nil {
		return nil, err
	}

	return resources, nil
}

func appFromStack(stack *cloudformation.Stack) *App {
	name := *stack.StackName
	tags := stackTags(stack)
	if value, ok := tags["Name"]; ok {
		// StackName probably includes the Rack prefix, prefer Name tag.
		name = value
	}
	return &App{
		Name:       name,
		Release:    stackParameters(stack)["Release"],
		Status:     humanStatus(*stack.StackStatus),
		Outputs:    stackOutputs(stack),
		Parameters: stackParameters(stack),
		Tags:       tags,
	}
}

func (s Apps) Len() int {
	return len(s)
}

func (s Apps) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

func (s Apps) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (a App) CronJobs(m manifest.Manifest) []CronJob {
	cronjobs := []CronJob{}

	for _, entry := range m.Services {
		labels := entry.LabelsByPrefix("convox.cron")
		for key, value := range labels {
			cronjob := NewCronJobFromLabel(key, value)
			e := entry
			cronjob.Service = &e
			cronjob.App = &a
			cronjobs = append(cronjobs, cronjob)
		}
	}
	return cronjobs
}

// findAppDefinitions looks for a specific ECS task revision and container definition that matches an app's process name and release ID.
// Given the taskDefinitionFamily prefix, this function will iterate the task's revisions starting with the most recent up to count revisions.
func findAppDefinitions(process, releaseID, taskDefinitionFamily string, count int) (*ecs.TaskDefinition, *ecs.ContainerDefinition, error) {
	//TODO: Move this function over to the aws apps provider implemntation once the Run methods have been ported over.

	var containerDefinition *ecs.ContainerDefinition

	ts, err := ECS().ListTaskDefinitions(&ecs.ListTaskDefinitionsInput{
		FamilyPrefix: aws.String(taskDefinitionFamily),
	})
	if err != nil {
		return nil, nil, err
	}

	startRevision := len(ts.TaskDefinitionArns) - 1
	maxRevision := 0
	// Only check the last 20 task definition revisions to run a release from.
	// Avoid any API rate limits and iterating over hudreds of task definitions.
	if startRevision > count {
		maxRevision = startRevision - count
	}

	for i := startRevision; i >= maxRevision; i-- {
		taskDefinition, err := ECS().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: ts.TaskDefinitionArns[i],
		})
		if err != nil {
			continue
		}

		if taskDefinition == nil {
			return nil, nil, fmt.Errorf("unable to retrieve task definition for %s", taskDefinitionFamily)
		}

		/// Loop logic used to find a previous release.
	ContainerSearch:
		for _, containerDefinition = range taskDefinition.TaskDefinition.ContainerDefinitions {

			if *containerDefinition.Name != process {
				continue ContainerSearch
			}

			for _, kv := range containerDefinition.Environment {
				if *kv.Name == "RELEASE" {
					if *kv.Value == releaseID {
						return taskDefinition.TaskDefinition, containerDefinition, nil
					}
				}
			}
		}
		////////////////////////////////////////////////////
	}

	return nil, nil, nil
}
