package models

import (
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"time"

	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/client"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
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
	res, err := CloudFormation().DescribeStacks(&cloudformation.DescribeStacksInput{})

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
	res, err := CloudFormation().DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String(name)})

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
	}

	req := &cloudformation.CreateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(a.Name),
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

	name := a.Name

	_, err := CloudFormation().DeleteStack(&cloudformation.DeleteStackInput{StackName: aws.String(name)})

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
		StackName:    aws.String(a.Name),
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

	_, err := CloudFormation().UpdateStack(req)

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

	var image string

	if registryId := a.Outputs["RegistryId"]; registryId != "" {
		image = fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s.%s", registryId, os.Getenv("AWS_REGION"), a.Outputs["RegistryRepository"], me.Name, release.Build)
	} else {
		image = fmt.Sprintf("%s/%s-%s:%s", os.Getenv("REGISTRY_HOST"), a.Name, me.Name, release.Build)
	}
	fmt.Println(image)

	d, err := Docker(host)

	if err != nil {
		return err
	}

	var repository string

	if registryId := a.Outputs["RegistryId"]; registryId != "" {
		repository = fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s", registryId, os.Getenv("AWS_REGION"), a.Outputs["RegistryRepository"])
	} else {
		repository = fmt.Sprintf("%s/%s-%s", os.Getenv("REGISTRY_HOST"), a.Name, me.Name)
	}

	err = d.PullImage(docker.PullImageOptions{
		Repository: repository,
		Tag:        release.Build,
	}, docker.AuthConfiguration{
		ServerAddress: os.Getenv("REGISTRY_HOST"),
		Username:      "convox",
		Password:      os.Getenv("PASSWORD"),
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
	resources := a.Resources()

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

	_, err := ECS().RunTask(req)

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

func (a *App) Builds() Builds {
	builds, err := ListBuilds(a.Name)

	if err != nil {
		if err.(awserr.Error).Message() == "Requested resource not found" {
			return Builds{}
		} else {
			panic(err)
		}
	}

	return builds
}

func (a *App) Created() bool {
	return len(a.Outputs) != 0
}

func (a *App) Deployments() Deployments {
	deployments, err := ListDeployments(a.Name)

	if err != nil {
		panic(err)
	}

	return deployments
}

func (a *App) HealthCheck() string {
	return a.Outputs["HealthCheck"]
}

func (a *App) HealthCheckEndpoints() []string {
	pp := []string{}

	for _, ps := range a.Processes() {
		for _, port := range a.ProcessPorts(ps.Name) {
			pp = append(pp, fmt.Sprintf("%s:%s", ps.Name, port))
		}
	}

	return pp
}

var regexpHealthCheckEndpoint = regexp.MustCompile(`HTTP:(\d+)(.*)`)

func (a *App) HealthCheckEndpoint() string {
	check := regexpHealthCheckEndpoint.FindStringSubmatch(a.Parameters["Check"])

	if len(check) != 3 {
		return ""
	}

	for _, ps := range a.Processes() {
		for _, port := range a.ProcessPorts(ps.Name) {
			if check[1] == a.Parameters[fmt.Sprintf("%sPort%sHost", UpperName(ps.Name), port)] {
				return fmt.Sprintf("%s:%s", ps.Name, port)
			}
		}
	}

	return ""
}

func (a *App) HealthCheckPath() string {
	check := regexpHealthCheckEndpoint.FindStringSubmatch(a.Parameters["Check"])

	if len(check) != 3 {
		return ""
	}

	return check[2]
}

func (a *App) ELBReady() bool {
	_, err := net.LookupCNAME(a.Outputs["BalancerHost"])

	return err == nil
}

func (a *App) Metrics() *Metrics {
	metrics, err := AppMetrics(a.Name)

	if err != nil {
		panic(err)
	}

	return metrics
}

func (a *App) Processes() Processes {
	processes, err := ListProcesses(a.Name)

	if err != nil {
		if aerr, ok := err.(awserr.RequestFailure); ok && aerr.StatusCode() == 400 {
			return Processes{}
		} else {
			// panic(err)
		}
	}

	return processes
}

func (a *App) Releases() Releases {
	releases, err := ListReleases(a.Name)

	if err != nil {
		if err.(awserr.Error).Message() == "Requested resource not found" {
			return Releases{}
		} else {
			panic(err)
		}
	}

	return releases
}

func (a *App) Resources() Resources {
	resources, err := ListResources(a.Name)

	if err != nil {
		panic(err)
	}

	return resources
}

func appFromStack(stack *cloudformation.Stack) *App {
	return &App{
		Name:       *stack.StackName,
		Release:    stackParameters(stack)["Release"],
		Status:     humanStatus(*stack.StackStatus),
		Outputs:    stackOutputs(stack),
		Parameters: stackParameters(stack),
		Tags:       stackTags(stack),
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
