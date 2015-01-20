package provider

import (
	"fmt"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/dynamodb"
)

type App struct {
	Name   string
	Status string

	Cluster   *Cluster
	Processes []Process
}

func AppList(cluster string) ([]App, error) {
	dres, err := appsTable(cluster).Scan(nil)

	if err != nil {
		return nil, err
	}

	status := map[string]string{}

	cres, err := CloudFormation.DescribeStacks("", "")

	if err != nil {
		return nil, err
	}

	apps := make([]App, 0)

	for _, app := range dres {
		name := app["name"].Value
		apps = append(apps, App{
			Name:   name,
			Status: humanStatus(status[fmt.Sprintf("%s-%s", cluster, name)]),
		})
	}

	for _, stack := range cres.Stacks {
		tags := stackTags(stack)
		if tags["type"] != "app" {
			continue
		}
		found := false
		for i, app := range apps {
			if tags["app"] == app.Name {
				apps[i].Status = stack.StackStatus
				found = true
				break
			}
		}
		if !found {
			apps = append(apps, App{
				Name:   tags["app"],
				Status: stack.StackStatus,
			})
		}
	}

	return apps, nil
}

func AppSync(cluster, name string) error {
	app := &App{Name: name}

	ps, err := ProcessList(cluster, name)

	if err != nil {
		return err
	}

	app.Processes = ps

	return nil
}

func AppCreate(cluster, app string, options map[string]string) error {
	attributes := []dynamodb.Attribute{
		*dynamodb.NewStringAttribute("name", app),
		*dynamodb.NewStringAttribute("created-at", "now"),
	}

	for k, v := range options {
		attributes = append(attributes, *dynamodb.NewStringAttribute(k, v))
	}

	_, err := appsTable(cluster).PutItem(app, "", attributes)

	return err
}

func AppDelete(cluster string, name string) error {
	_, err := CloudFormation.DeleteStack(fmt.Sprintf("%s-%s", cluster, name))

	if err != nil {
		return err
	}

	_, err = appsTable(cluster).DeleteItem(&dynamodb.Key{name, ""})

	return err
}
