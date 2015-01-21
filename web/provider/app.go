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
	Releases  []Release
}

type AppParams struct {
	AvailabilityZones []string
	Name              string
	Cluster           string
	Cidr              string
	Processes         []AppParamsProcess
	Subnets           []AppParamsSubnet
	Vpc               string
}

type AppParamsSubnet struct {
	AvailabilityZone string
	Cidr             string
	Name             string
	RouteTable       string
	Vpc              string
}

type AppParamsProcess struct {
	Ami               string
	App               string
	AvailabilityZones []string
	Cluster           string
	Count             int
	Name              string
	UserData          string
	Vpc               string
}

func AppList(cluster string) ([]App, error) {
	res, err := CloudFormation.DescribeStacks("", "")

	if err != nil {
		return nil, err
	}

	apps := make([]App, 0)

	for _, stack := range res.Stacks {
		tags := stackTags(stack)
		if tags["type"] == "app" && tags["cluster"] == cluster {
			apps = append(apps, App{
				Name:   tags["app"],
				Status: humanStatus(stack.StackStatus),
			})
		}
	}

	return apps, nil
}

func AppShow(cluster, app string) (*App, error) {
	apps, err := AppList(cluster)

	for _, a := range apps {
		if a.Name == app {
			a.Cluster = &Cluster{Name: cluster}

			if r, err := ReleaseList(cluster, app); err == nil {
				a.Releases = r
			}

			if err != nil {
				return nil, err
			}

			return &a, nil
		}
	}

	return nil, fmt.Errorf("no such app %s in cluster %s", app, cluster)
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

	if err != nil {
		return err
	}

	params, err := appParams(cluster, app)

	if err != nil {
		return err
	}

	template, err := buildTemplate("app", params)

	if err != nil {
		return err
	}

	printLines(template)

	tags := map[string]string{
		"type":    "app",
		"cluster": cluster,
		"app":     app,
		"subnet":  params.Cidr,
	}

	return createStackFromTemplate(template, fmt.Sprintf("%s-%s", cluster, app), tags)
}

func AppDelete(cluster string, name string) error {
	_, err := CloudFormation.DeleteStack(fmt.Sprintf("%s-%s", cluster, name))

	if err != nil {
		return err
	}

	_, err = appsTable(cluster).DeleteItem(&dynamodb.Key{name, ""})

	return err
}

func appsTable(cluster string) *dynamodb.Table {
	pk := dynamodb.PrimaryKey{dynamodb.NewStringAttribute("name", ""), nil}
	table := DynamoDB.NewTable(fmt.Sprintf("%s-apps", cluster), pk)
	return table
}

func appParams(cluster, app string) (*AppParams, error) {
	outputs, err := stackOutputs(cluster)

	if err != nil {
		return nil, err
	}

	vpc := outputs["Vpc"]
	rt := outputs["RouteTable"]

	base, err := nextAvailableSubnet(vpc)

	if err != nil {
		return nil, err
	}

	azs, err := availabilityZones()

	if err != nil {
		return nil, err
	}

	params := &AppParams{
		AvailabilityZones: azs,
		Cluster:           cluster,
		Cidr:              base,
		Name:              app,
		Vpc:               vpc,
	}

	subnets, err := divideSubnet(base, len(azs))

	if err != nil {
		return nil, err
	}

	params.Subnets = make([]AppParamsSubnet, len(azs))

	for i, az := range azs {
		params.Subnets[i] = AppParamsSubnet{
			AvailabilityZone: az,
			Cidr:             subnets[i],
			Name:             fmt.Sprintf("Subnet%d", i),
			RouteTable:       rt,
			Vpc:              vpc,
		}
	}

	return params, nil
}
