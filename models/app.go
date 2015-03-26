package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/cloudformation"
)

type App struct {
	Name string

	Status     string
	Release    string
	Repository string

	Outputs    map[string]string
	Parameters map[string]string
	Tags       map[string]string
}

type Apps []App

func ListApps() (Apps, error) {
	res, err := CloudFormation.DescribeStacks(&cloudformation.DescribeStacksInput{})

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
	res, err := CloudFormation.DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String(name)})

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
		"Repository": a.Repository,
	}

	tags := map[string]string{
		"System": "convox",
		"Type":   "app",
	}

	req := &cloudformation.CreateStackInput{
		StackName:    aws.String(a.Name),
		TemplateBody: aws.String(formation),
	}

	for key, value := range params {
		req.Parameters = append(req.Parameters, cloudformation.Parameter{ParameterKey: aws.String(key), ParameterValue: aws.String(value)})
	}

	for key, value := range tags {
		req.Tags = append(req.Tags, cloudformation.Tag{Key: aws.String(key), Value: aws.String(value)})
	}

	_, err = CloudFormation.CreateStack(req)

	return err
}

func (a *App) Delete() error {
	return CloudFormation.DeleteStack(&cloudformation.DeleteStackInput{StackName: aws.String(a.Name)})
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

func (a *App) Formation() (string, error) {
	formation, err := buildFormationTemplate("app", "formation", a)

	if err != nil {
		return "", err
	}

	// printLines(formation)

	return prettyJson(formation)
}

func (a *App) ProcessFormation() string {
	formation := ""

	for _, p := range a.Processes() {
		env := a.ServiceEnv()

		f, err := p.Formation(env)

		if err != nil {
			panic(err)
		}

		formation += f
	}

	return formation
}

func (a *App) ServiceEnv() string {
	env := ""

	for _, r := range a.Services() {
		e, err := r.Env()

		if err != nil {
			panic(err)
		}

		env += e
	}

	return env
}

func (a *App) ServiceFormation() string {
	formation := ""

	for _, r := range a.Services() {
		f, err := r.Formation()

		if err != nil {
			panic(err)
		}

		formation += f
	}

	return formation
}

func (a *App) WatchForCompletion(change *Change, original Events) {
	for {
		req := &cloudformation.DescribeStacksInput{StackName: aws.String(a.Name)}
		res, err := CloudFormation.DescribeStacks(req)

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

func (a *App) Ami() string {
	release, err := GetRelease(a.Name, a.Release)

	if err != nil {
		return ""
	}

	return release.Ami
}

func (a *App) Builds() Builds {
	builds, err := ListBuilds(a.Name)

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
		if err.(aws.APIError).Message == "Requested resource not found" {
			return Processes{}
		} else {
			panic(err)
		}
	}

	return processes
}

func (a *App) ProcessNames() []string {
	processes := a.Processes()

	pp := make([]string, len(processes))

	for i, p := range processes {
		pp[i] = p.Name
	}

	return pp
}

func (a *App) Releases() Releases {
	releases, err := ListReleases(a.Name)

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
	services, err := ListServices(a.Name)

	if err != nil {
		if err.(aws.APIError).Message == "Requested resource not found" {
			return Services{}
		} else {
			panic(err)
		}
	}

	return services
}

func (a *App) Subnets() Subnets {
	subnets, _ := ListSubnets()
	return subnets
}

func appFromStack(stack cloudformation.Stack) *App {
	params := stackParameters(stack)

	return &App{
		Name:       coalesce(stack.StackName, "<unknown>"),
		Status:     humanStatus(*stack.StackStatus),
		Repository: params["Repository"],
		Release:    params["Release"],
	}
}
