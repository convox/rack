package models

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/cloudformation"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/s3"
)

type App struct {
	Cluster string
	Name    string

	Status     string
	Release    string
	Repository string

	Outputs    map[string]string
	Parameters map[string]string
	Tags       map[string]string
}

type Apps []App

func ListApps(cluster string) (Apps, error) {
	res, err := CloudFormation().DescribeStacks(&cloudformation.DescribeStacksInput{})

	if err != nil {
		return nil, err
	}

	apps := make(Apps, 0)

	for _, stack := range res.Stacks {
		tags := stackTags(stack)

		if tags["System"] == "convox" && tags["Type"] == "app" && tags["Cluster"] == cluster {
			apps = append(apps, *appFromStack(stack))
		}
	}

	return apps, nil
}

func GetApp(cluster, name string) (*App, error) {
	res, err := CloudFormation().DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String(fmt.Sprintf("%s-%s", cluster, name))})

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
	cluster, err := GetCluster(a.Cluster)

	if err != nil {
		return err
	}

	formation, err := a.Formation()

	if err != nil {
		return err
	}

	params := map[string]string{
		"Repository": a.Repository,
		"Subnets":    cluster.Subnets,
		"VPC":        cluster.Vpc,
	}

	tags := map[string]string{
		"Cluster": a.Cluster,
		"System":  "convox",
		"Type":    "app",
	}

	req := &cloudformation.CreateStackInput{
		StackName:    aws.String(fmt.Sprintf("%s-%s", a.Cluster, a.Name)),
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

	builds, err := ListBuilds(a.Cluster, a.Name)

	if err != nil {
		return err
	}

	for _, build := range builds {
		go cleanupBuild(build)
	}

	releases, err := ListReleases(a.Cluster, a.Name)

	if err != nil {
		return err
	}

	for _, release := range releases {
		go cleanupRelease(release)
	}

	return nil
}

func (a *App) Delete() error {
	name := fmt.Sprintf("%s-%s", a.Cluster, a.Name)

	_, err := CloudFormation().DeleteStack(&cloudformation.DeleteStackInput{StackName: aws.String(name)})

	if err != nil {
		return err
	}

	go a.Cleanup()

	return nil
}

func (a *App) Formation() (string, error) {
	data, err := exec.Command("docker", "run", "convox/app").CombinedOutput()

	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (a *App) SubscribeLogs(output chan []byte, quit chan bool) error {
	resources := a.Resources()
	processes := a.Processes()

	done := make([](chan bool), len(processes))

	for i, ps := range processes {
		done[i] = make(chan bool)
		go subscribeKinesis(ps.Name, resources[fmt.Sprintf("%sKinesis", upperName(ps.Name))].Id, output, done[i])
	}

	return nil
}

func (a *App) ForkRelease() (*Release, error) {
	release, err := a.LatestRelease()

	if err != nil {
		return nil, err
	}

	if release == nil {
		*release = NewRelease(a.Cluster, a.Name)
	}

	release.Id = generateId("R", 10)
	release.Created = time.Time{}

	return release, nil
}

func (a *App) LatestRelease() (*Release, error) {
	releases, err := ListReleases(a.Cluster, a.Name)

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

func (a *App) Builds() Builds {
	builds, err := ListBuilds(a.Cluster, a.Name)

	if err != nil {
		if err.(aws.APIError).Message == "Requested resource not found" {
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
	return a.Status != "creating"
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
		if err.(aws.APIError).StatusCode == 400 {
			return Processes{}
		} else {
			panic(err)
		}
	}

	return processes
}

func (a *App) Releases() Releases {
	releases, err := ListReleases(a.Cluster, a.Name)

	if err != nil {
		if err.(aws.APIError).Message == "Requested resource not found" {
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
	services, err := ListServices(a.Cluster, a.Name)

	if err != nil {
		if err.(aws.APIError).Message == "Requested resource not found" {
			return Services{}
		} else {
			panic(err)
		}
	}

	return services
}

func appFromStack(stack *cloudformation.Stack) *App {
	params := stackParameters(stack)
	tags := stackTags(stack)

	return &App{
		Name:       strings.SplitN(*stack.StackName, "-", 2)[1],
		Cluster:    tags["Cluster"],
		Status:     humanStatus(*stack.StackStatus),
		Repository: params["Repository"],
		Release:    params["Release"],
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
