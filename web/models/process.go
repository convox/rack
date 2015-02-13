package models

import (
	"fmt"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/dynamodb"
)

type Process struct {
	Name  string
	Count string
	Ports []int

	App string
}

type Processes []Process

func ListProcesses(app string) (Processes, error) {
	res, err := DynamoDB.Scan(&dynamodb.ScanInput{TableName: aws.String(processesTable(app))})

	if err != nil {
		return nil, err
	}

	processes := make(Processes, len(res.Items))

	for i, item := range res.Items {
		processes[i] = *processFromItem(item)
	}

	return processes, nil
}

func GetProcess(app, name string) (*Process, error) {
	req := &dynamodb.GetItemInput{
		ConsistentRead: aws.Boolean(true),
		Key: map[string]dynamodb.AttributeValue{
			"name": dynamodb.AttributeValue{S: aws.String(name)},
		},
		TableName: aws.String(processesTable(app)),
	}

	res, err := DynamoDB.GetItem(req)

	if err != nil {
		return nil, err
	}

	return processFromItem(res.Item), nil
}

func (p *Process) Save() error {
	req := &dynamodb.PutItemInput{
		Item: map[string]dynamodb.AttributeValue{
			"name":  dynamodb.AttributeValue{S: aws.String(p.Name)},
			"count": dynamodb.AttributeValue{S: aws.String(p.Count)},
			"app":   dynamodb.AttributeValue{S: aws.String(p.App)},
		},
		TableName: aws.String(processesTable(p.App)),
	}

	_, err := DynamoDB.PutItem(req)

	return err
}

func (p *Process) Balancer() bool {
	return p.Name == "web"
}

func (p *Process) BalancerUrl() string {
	app, err := GetApp(p.App)

	if err != nil {
		return ""
	}

	return app.Outputs[upperName(p.Name)+"BalancerHost"]
}

func (p *Process) Formation(env string) (string, error) {
	p.Ports = []int{3000}

	params := map[string]interface{}{
		"Env":     env,
		"Process": p,
	}

	formation, err := buildFormationTemplate("process", "formation", params)

	if err != nil {
		return "", err
	}

	return formation, nil
}

func (p *Process) AvailabilityZones() []string {
	azs := []string{}

	for _, subnet := range ListSubnets() {
		azs = append(azs, subnet.AvailabilityZone)
	}

	return azs
}

func (p *Process) Userdata() string {
	return `""`
}

func (p *Process) Instances() Instances {
	instances, err := ListInstances(p.App, p.Name)

	if err != nil {
		panic(err)
	}

	return instances
}

func (p *Process) Metrics() *Metrics {
	metrics, err := ProcessMetrics(p.App, p.Name)

	if err != nil {
		panic(err)
	}

	return metrics
}

func (p *Process) SubscribeLogs(output chan []byte, quit chan bool) error {
	resources, err := ListResources(p.App)

	if err != nil {
		return err
	}

	done := make(chan bool)
	go subscribeKinesis(p.Name, resources[fmt.Sprintf("%sKinesis", upperName(p.Name))].PhysicalId, output, done)

	return nil
}

func processesTable(app string) string {
	return fmt.Sprintf("convox-%s-processes", app)
}

func processFromItem(item map[string]dynamodb.AttributeValue) *Process {
	return &Process{
		Name:  coalesce(item["name"].S, ""),
		Count: coalesce(item["count"].S, "0"),
		App:   coalesce(item["app"].S, ""),
	}
}
