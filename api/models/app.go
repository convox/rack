package models

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/client"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecr"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/s3"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
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
	return GetAppFast(name, true, true)
}

func GetAppFast(name string, bound bool, unbound bool) (*App, error) {
	stackName := shortNameToStackName(name)

	switch {
	case !bound && !unbound:
		// User requested no lookups, return empty App.
		return &App{Name: name}, nil
	case bound && !unbound:
		// User requested bound lookup only, reset short name.
		name = stackName
	case !bound && unbound:
		// User requested unbound lookup only, reset long name.
		stackName = name
	}

	res, err := DescribeStack(stackName)

	// Setting to the same value results in at most a single lookup.
	if name != stackName && awsError(err) == "ValidationError" {
		// Only lookup an unbound app if the name/stackName differ and the
		// bound lookup fails.
		res, err = DescribeStack(name)
	}

	if err != nil {
		return nil, err
	}

	if len(res.Stacks) != 1 {
		return nil, fmt.Errorf("could not load stack for app: %s", name)
	}

	tags := stackTags(res.Stacks[0])

	if tags["Rack"] != "" && tags["Rack"] != os.Getenv("RACK") {
		return nil, fmt.Errorf("no such app: %s", name)
	}

	app := appFromStack(res.Stacks[0])

	return app, nil
}

var regexValidAppName = regexp.MustCompile(`\A[a-zA-Z][-a-zA-Z0-9]{3,29}\z`)

func (a *App) StackName() string {
	if a.Tags == nil {
		return shortNameToStackName(a.Name)
	}

	if _, ok := a.Tags["Name"]; ok {
		// Use the name provided by the user (they *should* match).
		return shortNameToStackName(a.Name)
	}

	return a.Name
}

func (a *App) Create() error {
	helpers.TrackEvent("kernel-app-create-start", nil)

	if !regexValidAppName.MatchString(a.Name) {
		return fmt.Errorf("app name can contain only alphanumeric characters and dashes and must be between 4 and 30 characters")
	}

	formation, err := a.Formation()

	if err != nil {
		helpers.TrackEvent("kernel-app-create-error", nil)
		return err
	}

	params := map[string]string{
		"Cluster": os.Getenv("CLUSTER"),
		"Subnets": os.Getenv("SUBNETS"),
		"Version": os.Getenv("RELEASE"),
		"VPC":     os.Getenv("VPC"),
	}

	if os.Getenv("ENCRYPTION_KEY") != "" {
		params["Key"] = os.Getenv("ENCRYPTION_KEY")
	}

	tags := map[string]string{
		"Rack":   os.Getenv("RACK"),
		"System": "convox",
		"Type":   "app",
		"Name":	  a.Name,
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

func (a *App) Cleanup() error {
	err := cleanupBucket(a.Outputs["Settings"])

	if err != nil {
		return err
	}

	builds, err := ListBuilds(a.Name)

	if err != nil {
		return err
	}

	for _, build := range builds {
		go cleanupBuild(build)
	}

	releases, err := ListReleases(a.Name)

	if err != nil {
		return err
	}

	for _, release := range releases {
		go cleanupRelease(release)
	}

	return nil
}

func (a *App) Delete() error {
	helpers.TrackEvent("kernel-app-delete-start", nil)

	_, err := CloudFormation().DeleteStack(&cloudformation.DeleteStackInput{StackName: aws.String(a.StackName())})

	if err != nil {
		helpers.TrackEvent("kernel-app-delete-error", nil)
		return err
	}

	go a.Cleanup()

	helpers.TrackEvent("kernel-app-delete-success", nil)

	NotifySuccess("app:delete", map[string]string{"name": a.Name})

	return nil
}

func (a *App) UpdateParamsAndTemplate(changes map[string]string, template string) error {
	req := &cloudformation.UpdateStackInput{
		StackName:    aws.String(a.StackName()),
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
	}

	if template != "" {
		req.TemplateURL = aws.String(template)
	} else {
		req.UsePreviousTemplate = aws.Bool(true)
	}

	params := a.Parameters

	for key, val := range changes {
		params[key] = val
	}

	for key, val := range params {
		req.Parameters = append(req.Parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String(key),
			ParameterValue: aws.String(val),
		})
	}

	_, err := UpdateStack(req)

	return err
}

func (a *App) UpdateParams(changes map[string]string) error {
	return a.UpdateParamsAndTemplate(changes, "")
}

func (a *App) Formation() (string, error) {
	data, err := buildTemplate("app", "app", Manifest{})

	if err != nil {
		return "", err
	}

	return string(data), nil
}

// During the transition from Kinesis to CloudWatch Logs, apps might not have been
// re-deployed and provisioned a LogGroup.
// Conditionally fall back to reading from Kinesis in this case.
func (a *App) SubscribeLogs(output chan []byte, quit chan bool) error {
	go subscribeKinesis(a.Outputs["Kinesis"], output, quit)
	return nil
}

func (a *App) ForkRelease() (*Release, error) {
	release, err := a.LatestRelease()

	if err != nil {
		return nil, err
	}

	if release == nil {
		r := NewRelease(a.Name)
		release = &r
	}

	release.Id = generateId("R", 10)
	release.Created = time.Time{}

	return release, nil
}

func (a *App) LatestRelease() (*Release, error) {
	releases, err := ListReleases(a.Name)

	if err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return nil, nil
	}

	return &releases[0], nil
}

func (a *App) ExecAttached(pid, command string, rw io.ReadWriter) error {
	var ps Process

	pss, err := ListProcesses(a.Name)

	if err != nil {
		return err
	}

	for _, p := range pss {
		if p.Id == pid {
			ps = p
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

func (a *App) RunAttached(process, command string, rw io.ReadWriter) error {
	env, err := GetEnvironment(a.Name)

	if err != nil {
		return err
	}

	ea := make([]string, 0)

	for k, v := range env {
		ea = append(ea, fmt.Sprintf("%s=%s", k, v))
	}

	release, err := a.LatestRelease()

	if err != nil {
		return err
	}

	if release == nil {
		return fmt.Errorf("no releases for app: %s", a.Name)
	}

	manifest, err := LoadManifest(release.Manifest)

	if err != nil {
		return err
	}

	me := manifest.Entry(process)

	if me == nil {
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
		image = fmt.Sprintf("%s/%s-%s:%s", os.Getenv("REGISTRY_HOST"), a.StackName(), me.Name, release.Build)
		repository = fmt.Sprintf("%s/%s-%s", os.Getenv("REGISTRY_HOST"), a.StackName(), me.Name)
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
			Env:          ea,
			OpenStdin:    true,
			Tty:          true,
			Cmd:          []string{"sh", "-c", command},
			Image:        image,
			Labels: map[string]string{
				"com.convox.rack.type":    "oneoff",
				"com.convox.rack.app":     a.Name,
				"com.convox.rack.process": process,
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

	code, err := d.WaitContainer(res.ID)

	if err != nil {
		return err
	}

	_, err = rw.Write([]byte(fmt.Sprintf("%s%d\n", StatusCodePrefix, code)))

	if err != nil {
		return err
	}

	return nil
}

func (a *App) RunDetached(process, command string) error {
	resources, err := a.Resources()

	if err != nil {
		return err
	}

	req := &ecs.RunTaskInput{
		Cluster:        aws.String(os.Getenv("CLUSTER")),
		Count:          aws.Int64(1),
		TaskDefinition: aws.String(resources[UpperName(process)+"ECSTaskDefinition"].Id),
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

	if err != nil {
		return err
	}

	return nil
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

func cleanupBucket(bucket string) error {
	req := &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
	}

	res, err := S3().ListObjectVersions(req)

	if err != nil {
		return err
	}

	for _, d := range res.DeleteMarkers {
		go cleanupBucketObject(bucket, *d.Key, *d.VersionId)
	}

	for _, v := range res.Versions {
		go cleanupBucketObject(bucket, *v.Key, *v.VersionId)
	}

	return nil
}

func cleanupBucketObject(bucket, key, version string) {
	req := &s3.DeleteObjectInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String(key),
		VersionId: aws.String(version),
	}

	_, err := S3().DeleteObject(req)

	if err != nil {
		fmt.Printf("error: %s\n", err)
	}
}

func cleanupBuild(build Build) {
	err := build.Cleanup()

	if err != nil {
		fmt.Printf("error: %s\n", err)
	}
}

func cleanupRelease(release Release) {
	err := release.Cleanup()

	if err != nil {
		fmt.Printf("error: %s\n", err)
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
