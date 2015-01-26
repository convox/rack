package models

import (
	"fmt"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/goamz/goamz/cloudformation"
)

type App struct {
	Name string

	Status     string
	Outputs    map[string]string
	Parameters map[string]string
	Repository string
	Release    string

	Builds    Builds
	Processes Processes
	Releases  Releases
	Resources Resources
}

type Apps []App

func ListApps() (Apps, error) {
	res, err := CloudFormation.DescribeStacks("", "")

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
	res, err := CloudFormation.DescribeStacks(fmt.Sprintf("convox-%s", name), "")

	if err != nil {
		return nil, err
	}

	if len(res.Stacks) != 1 {
		return nil, fmt.Errorf("could not load stack for app: %s", name)
	}

	app := appFromStack(res.Stacks[0])

	app.Outputs = stackOutputs(res.Stacks[0])
	app.Parameters = stackParameters(res.Stacks[0])

	builds, err := ListBuilds(app.Name)

	if err != nil {
		return nil, err
	}

	app.Builds = builds

	processes, err := ListProcesses(app.Name)

	if err != nil {
		return nil, err
	}

	app.Processes = processes

	releases, err := ListReleases(app.Name)

	if err != nil {
		return nil, err
	}

	app.Releases = releases

	return app, nil
}

func (a *App) Create() error {
	formation, err := buildTemplate("formation", "formation", a)

	if err != nil {
		return err
	}

	printLines(formation)

	params := map[string]string{
		"Repository": a.Repository,
	}

	tags := map[string]string{
		"System": "convox",
		"Type":   "app",
	}

	return createStack(formation, fmt.Sprintf("convox-%s", a.Name), params, tags)
}

func (a *App) Delete() error {
	_, err := CloudFormation.DeleteStack(fmt.Sprintf("convox-%s", a.Name))
	return err
}

func (a *App) Formation() (string, error) {
	fmt.Printf("a %+v\n", a)

	fmt.Printf("a.Ami() %+v\n", a.Ami())

	formation, err := buildTemplate("formation", "formation", a)

	if err != nil {
		return "", err
	}

	// printLines(formation)

	return prettyJson(formation)
}

func (a *App) Ami() string {
	release, err := GetRelease(a.Name, a.Release)

	fmt.Printf("release %+v\n", release)
	fmt.Printf("err %+v\n", err)

	if err != nil {
		return ""
	}

	return release.Ami
}

func (a *App) ProcessFormation() string {
	formation := ""

	for _, p := range a.Processes {
		env := a.ResourceEnv()

		f, err := p.Formation(env)

		if err != nil {
			panic(err)
		}

		formation += f
	}

	return formation
}

func (a *App) ResourceEnv() string {
	env := ""

	for _, r := range a.Resources {
		e, err := r.Env()

		if err != nil {
			panic(err)
		}

		env += e
	}

	return env
}

func (a *App) ResourceFormation() string {
	formation := ""

	for _, r := range a.Resources {
		f, err := r.Formation()

		if err != nil {
			panic(err)
		}

		formation += f
	}

	return formation
}

func (a *App) Subnets() Subnets {
	return ListSubnets()
}

func appFromStack(stack cloudformation.Stack) *App {
	params := stackParameters(stack)

	return &App{
		Name:       stack.StackName[7:],
		Status:     humanStatus(stack.StackStatus),
		Repository: params["Repository"],
		Release:    params["Release"],
	}
}
