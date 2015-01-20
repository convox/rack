package provider

import (
	"fmt"
	"strings"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/dynamodb"
)

type App struct {
	Name   string
	Status string

	Cluster   *Cluster
	Processes []Process
}

func AppList(cluster string) ([]App, error) {
	res, err := CloudFormation.DescribeStacks("", "")

	if err != nil {
		return nil, err
	}

	apps := make([]App, 0)

	for _, stack := range res.Stacks {
		tags := flattenTags(stack.Tags)
		if tags["type"] == "app" && tags["cluster"] == cluster {
			apps = append(apps, App{
				Name:   tags["app"],
				Status: humanStatus(stack.StackStatus),
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

func AppCreate(cluster, app string) error {
	outputs, err := stackOutputs(cluster)

	if err != nil {
		return err
	}

	vpc := outputs["Vpc"]
	rt := outputs["RouteTable"]

	base, err := nextAvailableSubnet(vpc)

	if err != nil {
		return err
	}

	params := &AppParams{
		Name:    upperName(app),
		Cluster: upperName(cluster),
		Cidr:    base,
		Vpc:     vpc,
	}

	azs, err := availabilityZones()

	if err != nil {
		return err
	}

	subnets, err := divideSubnet(base, len(azs))

	if err != nil {
		return err
	}

	params.Subnets = make([]AppParamsSubnet, len(azs))

	for i, az := range azs {
		params.Subnets[i] = AppParamsSubnet{
			Name:             fmt.Sprintf("Subnet%d", i),
			AvailabilityZone: az,
			Cidr:             subnets[i],
			RouteTable:       rt,
			Vpc:              vpc,
		}
	}

	uparams := UserdataParams{
		Process:   "web",
		Env:       map[string]string{"FOO": "bar"},
		Resources: []UserdataParamsResource{},
		Ports:     []int{5000},
	}

	userdata, err := parseTemplate("userdata", uparams)

	if err != nil {
		return err
	}

	params.Processes = []AppParamsProcess{
		{
			Name:              "Web",
			Process:           "web",
			Count:             2,
			Vpc:               vpc,
			App:               app,
			Ami:               "ami-acb1cfc4",
			Cluster:           cluster,
			AvailabilityZones: azs,
			UserData:          userdata,
		},
	}

	formation, err := parseTemplate("app", params)

	lines := strings.Split(formation, "\n")

	for i, line := range lines {
		fmt.Printf("%d: %s\n", i, line)
	}

	if err != nil {
		return err
	}

	tags := map[string]string{
		"type":    "app",
		"cluster": cluster,
		"app":     app,
		"subnet":  base,
	}

	err = createStackFromTemplate(formation, fmt.Sprintf("%s-%s", cluster, app), tags)

	if err != nil {
		return err
	}

	attributes := []dynamodb.Attribute{
		*dynamodb.NewStringAttribute("name", app),
		*dynamodb.NewStringAttribute("created-at", "now"),
	}

	_, err = appsTable(cluster).PutItem(app, "", attributes)

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
