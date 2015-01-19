package provider

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strings"

	caws "github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/aws"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/dynamodb"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/ec2"

	gaws "github.com/convox/kernel/web/Godeps/_workspace/src/github.com/goamz/goamz/aws"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/goamz/goamz/cloudformation"
)

var (
	cauth = caws.Auth{AccessKey: os.Getenv("AWS_ACCESS"), SecretKey: os.Getenv("AWS_SECRET")}
	gauth = gaws.Auth{AccessKey: os.Getenv("AWS_ACCESS"), SecretKey: os.Getenv("AWS_SECRET")}
)

var (
	CloudFormation = cloudformation.New(gauth, gaws.Regions[os.Getenv("AWS_REGION")])
	DynamoDB       = dynamodb.New(cauth, caws.Regions[os.Getenv("AWS_REGION")])
	EC2            = ec2.New(cauth, caws.Regions[os.Getenv("AWS_REGION")])
)

type App struct {
	Name   string
	Status string

	Cluster   *Cluster
	Processes []Process
}

type Cluster struct {
	Name   string
	Id     string
	Status string

	Apps              []App
	AvailabilityZones []string
	Subnets           []Subnet
}

type Process struct {
	Name      string
	UpperName string
}

type Subnet struct {
	AvailabilityZone string
	Cidr             string
	Id               string
}

func (a *App) UpperName() string {
	return upperName(a.Name)
}

func (c *Cluster) UpperName() string {
	return upperName(c.Name)
}

func ClusterList() ([]Cluster, error) {
	res, err := CloudFormation.DescribeStacks("", "")

	if err != nil {
		return nil, err
	}

	clusters := make([]Cluster, 0)

	for _, stack := range res.Stacks {
		if flattenTags(stack.Tags)["type"] == "cluster" {
			clusters = append(clusters, Cluster{Name: stack.StackName, Status: humanStatus(stack.StackStatus)})
		}
	}

	return clusters, nil
}

func ClusterCreate(name string) error {
	cluster := &Cluster{Name: name}

	zres, err := EC2.DescribeAvailabilityZones(nil, nil)

	if err != nil {
		return fmt.Errorf("could not describe availability zones while creating stack %s: %s", name, err)
	}

	for i, zone := range zres.AvailabilityZones {
		cidr := fmt.Sprintf("10.0.%d.0/19", i*32)
		cluster.Subnets = append(cluster.Subnets, Subnet{AvailabilityZone: zone.Name, Cidr: cidr})
	}

	tags := map[string]string{
		"type":    "cluster",
		"cluster": name,
	}

	err = createStackFromTemplate("cluster", name, cluster, tags)

	if err != nil {
		return fmt.Errorf("could not create stack %s: %s", name, err)
	}

	return nil
}

func ClusterDelete(name string) error {
	_, err := CloudFormation.DeleteStack(name)

	if err != nil {
		return fmt.Errorf("could not delete stack %s: %s", name, err)
	}

	return nil
}

func AppsTable(cluster string) *dynamodb.Table {
	pk := dynamodb.PrimaryKey{dynamodb.NewStringAttribute("name", ""), nil}
	table := DynamoDB.NewTable(fmt.Sprintf("%s-apps", cluster), pk)
	return table
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

func AppCreate(cluster string, name string) error {
	vpc, err := stackOutput(cluster, "Vpc")

	if err != nil {
		return err
	}

	app := &App{Name: name, Cluster: &Cluster{Name: cluster, Id: vpc}}

	subnets, err := stackOutputList(cluster, "Subnet")

	if err != nil {
		return err
	}

	app.Cluster.Subnets = make([]Subnet, len(subnets))

	for i, subnet := range subnets {
		app.Cluster.Subnets[i] = Subnet{Id: subnet}
	}

	azs, err := availabilityZones()

	if err != nil {
		return err
	}

	app.Cluster.AvailabilityZones = azs

	tags := map[string]string{
		"type":    "app",
		"cluster": cluster,
		"app":     name,
	}

	err = createStackFromTemplate("app", fmt.Sprintf("%s-%s", cluster, name), app, tags)

	if err != nil {
		return err
	}

	attributes := []dynamodb.Attribute{
		*dynamodb.NewStringAttribute("name", name),
		*dynamodb.NewStringAttribute("created-at", "now"),
	}

	_, err = AppsTable(cluster).PutItem(name, "", attributes)

	return err
}

func AppDelete(cluster string, name string) error {
	_, err := CloudFormation.DeleteStack(fmt.Sprintf("%s-%s", cluster, name))

	if err != nil {
		return err
	}

	_, err = AppsTable(cluster).DeleteItem(&dynamodb.Key{name, ""})

	return err
}

func ProcessList(cluster, app string) ([]Process, error) {
	return []Process{}, nil
}

func availabilityZones() ([]string, error) {
	res, err := EC2.DescribeAvailabilityZones(nil, nil)

	if err != nil {
		return nil, err
	}

	subnets := make([]string, len(res.AvailabilityZones))

	for i, zone := range res.AvailabilityZones {
		subnets[i] = zone.Name
	}

	return subnets, nil
}

func createStackFromTemplate(templateName, name string, object interface{}, tags map[string]string) error {
	funcs := template.FuncMap{
		"azList": func(ss []string) template.HTML {
			ms := make([]string, len(ss))
			for i, s := range ss {
				ms[i] = fmt.Sprintf("%q", s)
			}
			return template.HTML(strings.Join(ms, ", "))
		},
		"subnetList": func(ss []Subnet) template.HTML {
			ms := make([]string, len(ss))
			for i, s := range ss {
				ms[i] = fmt.Sprintf("%q", s.Id)
			}
			return template.HTML(strings.Join(ms, ", "))
		},
	}

	tmpl, err := template.New(templateName).Funcs(funcs).ParseFiles(fmt.Sprintf("templates/formation/%s.tmpl", templateName))

	if err != nil {
		return fmt.Errorf("could not parse template while creating vpc %s: %s", name, err)
	}

	var formation bytes.Buffer

	err = tmpl.Execute(&formation, object)

	// fmt.Printf("formation.String() %+v\n", formation.String())

	if err != nil {
		return fmt.Errorf("could not parse formation from template while creating stack %s: %s", name, err)
	}

	params := &cloudformation.CreateStackParams{
		StackName:    name,
		TemplateBody: formation.String(),
	}

	for key, value := range tags {
		params.Tags = append(params.Tags, cloudformation.Tag{Key: key, Value: value})
	}

	_, err = CloudFormation.CreateStack(params)

	return err
}

func flattenTags(tags []cloudformation.Tag) map[string]string {
	f := make(map[string]string)

	for _, tag := range tags {
		f[tag.Key] = tag.Value
	}

	return f
}

func humanStatus(original string) string {
	switch original {
	case "CREATE_IN_PROGRESS":
		return "creating"
	case "CREATE_COMPLETE":
		return "running"
	case "DELETE_FAILED":
		return "running"
	case "DELETE_IN_PROGRESS":
		return "deleting"
	case "ROLLBACK_COMPLETE":
		return "failed"
	default:
		fmt.Printf("unknown status: %s\n", original)
		return "unknown"
	}
}

func stackOutput(stackName string, outputKey string) (string, error) {
	res, err := CloudFormation.DescribeStacks(stackName, "")

	if err != nil {
		return "", err
	}

	if len(res.Stacks) != 1 {
		return "", fmt.Errorf("could not fetch stack %s", stackName)
	}

	for _, output := range res.Stacks[0].Outputs {
		if output.OutputKey == outputKey {
			return output.OutputValue, nil
		}
	}

	return "", fmt.Errorf("no such output key for stack %s: %s", stackName, outputKey)
}

func stackOutputList(stackName string, outputPrefix string) ([]string, error) {
	res, err := CloudFormation.DescribeStacks(stackName, "")

	if err != nil {
		return nil, err
	}

	if len(res.Stacks) != 1 {
		return nil, fmt.Errorf("could not fetch stack %s", stackName)
	}

	values := make([]string, 0)

	for _, output := range res.Stacks[0].Outputs {
		if strings.HasPrefix(output.OutputKey, outputPrefix) {
			values = append(values, output.OutputValue)
		}
	}

	return values, nil
}

func upperName(name string) string {
	return strings.ToUpper(name[0:1]) + name[1:]
}
