package models

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws/awserr"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/cloudformation"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/s3"
)

var CustomTopic = os.Getenv("CUSTOM_TOPIC")

type App struct {
	Name string

	Status     string
	Repository string

	Outputs    map[string]string
	Parameters map[string]string
	Tags       map[string]string
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
			apps = append(apps, *appFromStack(stack))
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

	app := appFromStack(res.Stacks[0])

	app.Outputs = stackOutputs(res.Stacks[0])
	app.Parameters = stackParameters(res.Stacks[0])
	app.Tags = stackTags(res.Stacks[0])

	return app, nil
}

func (a *App) Create() error {
	formation, err := a.Formation()

	if err != nil {
		return err
	}

	params := map[string]string{
		"Cluster":    os.Getenv("CLUSTER"),
		"Kernel":     CustomTopic,
		"Repository": a.Repository,
		"Subnets":    os.Getenv("SUBNETS"),
		"VPC":        os.Getenv("VPC"),
	}

	tags := map[string]string{
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

	return err
}

func (a *App) Cleanup() error {
	err := cleanupBucket(a.Outputs["Settings"])

	if err != nil {
		return err
	}

	l := make(map[string]string)
	builds, err := ListBuilds(a.Name, l)

	if err != nil {
		return err
	}

	for _, build := range builds {
		go cleanupBuild(build)
	}

	releases, err := ListReleases(a.Name, l)

	if err != nil {
		return err
	}

	for _, release := range releases {
		go cleanupRelease(release)
	}

	return nil
}

func (a *App) Delete() error {
	name := a.Name

	_, err := CloudFormation().DeleteStack(&cloudformation.DeleteStackInput{StackName: aws.String(name)})

	if err != nil {
		return err
	}

	go a.Cleanup()

	return nil
}

func (a *App) UpdateParams(changes map[string]string) error {
	req := &cloudformation.UpdateStackInput{
		StackName:           aws.String(a.Name),
		UsePreviousTemplate: aws.Boolean(true),
		Capabilities:        []*string{aws.String("CAPABILITY_IAM")},
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

	res, err := CloudFormation().UpdateStack(req)

	fmt.Printf("res = %+v\n", res)
	fmt.Printf("err = %+v\n", err)

	return err
}

func (a *App) Formation() (string, error) {
	data, err := exec.Command("docker", "run", fmt.Sprintf("convox/app:%s", os.Getenv("RELEASE")), "-mode", "staging").CombinedOutput()

	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (a *App) SubscribeLogs(output chan []byte, quit chan bool) error {
	done := make(chan bool)

	go subscribeKinesis(a.Outputs["Kinesis"], output, done)

	return nil
}

func (a *App) ActiveRelease() string {
	return a.Parameters["Release"]
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
	l := make(map[string]string)
	releases, err := ListReleases(a.Name, l)

	if err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return nil, nil
	}

	return &releases[0], nil
}

func (a *App) WatchForCompletion(change *Change, original Events) {
	for {
		req := &cloudformation.DescribeStacksInput{StackName: aws.String(a.Name)}

		res, err := CloudFormation().DescribeStacks(req)

		if err != nil {
			panic(err)
		}

		if len(res.Stacks) < 1 {
			panic(fmt.Errorf("no such stack: %s", a.Name))
		}

		status := *res.Stacks[0].StackStatus

		latest, err := ListEvents(a.Name)

		events := Events{}
		for _, event := range latest {
			if event.Id == original[0].Id {
				break
			}
			events = append(events, event)
		}

		transactions, err := GroupEvents(events)
		if err != nil {
			panic(err)
		}

		data, err := json.Marshal(ChangeMetadata{
			Events:       events,
			Transactions: transactions,
		})

		change.Metadata = string(data)
		change.Save()

		if status == "UPDATE_COMPLETE" || status == "UPDATE_ROLLBACK_COMPLETE" {
			break
		}

		time.Sleep(2 * time.Second)
	}

	change.Status = "complete"
	change.Save()
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
		r := regexp.MustCompile(fmt.Sprintf("%sPort([0-9]+)Balancer", upperName(ps)))

		if matches := r.FindStringSubmatch(key); len(matches) == 2 {
			ports[matches[1]] = value
		}
	}

	return ports
}

func (a *App) Builds() Builds {
	l := make(map[string]string)
	builds, err := ListBuilds(a.Name, l)

	if err != nil {
		if err.(awserr.Error).Message() == "Requested resource not found" {
			return Builds{}
		} else {
			panic(err)
		}
	}

	return builds
}

func (a *App) Changes() Changes {
	changes, err := ListChanges(a.Name)

	if err != nil {
		panic(err)
	}

	return changes
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
			if check[1] == a.Parameters[fmt.Sprintf("%sPort%sHost", upperName(ps.Name), port)] {
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
	l := make(map[string]string)
	releases, err := ListReleases(a.Name, l)

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

func (a *App) Services() Services {
	services, err := ListServices(a.Name)

	if err != nil {
		panic(err)
	}

	return services
}

func appFromStack(stack *cloudformation.Stack) *App {
	params := stackParameters(stack)

	return &App{
		Name:       *stack.StackName,
		Status:     humanStatus(*stack.StackStatus),
		Repository: params["Repository"],
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
		go cleanupBucketObject(bucket, *d.Key, *d.VersionID)
	}

	for _, v := range res.Versions {
		go cleanupBucketObject(bucket, *v.Key, *v.VersionID)
	}

	return nil
}

func cleanupBucketObject(bucket, key, version string) {
	req := &s3.DeleteObjectInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String(key),
		VersionID: aws.String(version),
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
