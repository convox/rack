package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/dynamodb"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/s3"
)

type Process struct {
	Name  string
	Count string
	Ports []int

	App string
}

type Processes []Process

func ListProcesses(app string) (Processes, error) {
	a, err := GetApp(app)

	if err != nil {
		return nil, err
	}

	req := &s3.ListObjectsRequest{
		Bucket: aws.String(a.Outputs["Settings"]),
		Prefix: aws.String("process/"),
	}

	res, err := S3.ListObjects(req)

	processes := make(Processes, len(res.Contents))

	for i, p := range res.Contents {
		name := strings.TrimPrefix(*p.Key, "process/")
		ps, err := GetProcess(app, name)

		if err != nil {
			return nil, err
		}

		processes[i] = *ps
	}

	return processes, nil
}

func GetProcess(app, name string) (*Process, error) {
	a, err := GetApp(app)

	if err != nil {
		return nil, err
	}

	req := &s3.GetObjectRequest{
		Bucket: aws.String(a.Outputs["Settings"]),
		Key:    aws.String(fmt.Sprintf("process/%s", name)),
	}

	res, err := S3.GetObject(req)

	if err != nil {
		return nil, err
	}

	value, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	var process *Process

	err = json.Unmarshal([]byte(value), &process)

	if err != nil {
		return nil, err
	}

	return process, nil
}

func (p *Process) Save() error {
	app, err := GetApp(p.App)

	if err != nil {
		return err
	}

	data, err := json.Marshal(p)

	if err != nil {
		return err
	}

	req := &s3.PutObjectRequest{
		Body:          ioutil.NopCloser(bytes.NewReader(data)),
		Bucket:        aws.String(app.Outputs["Settings"]),
		ContentLength: aws.Long(int64(len(data))),
		Key:           aws.String(fmt.Sprintf("process/%s", p.Name)),
	}

	_, err = S3.PutObject(req)

	return err
}

func (p *Process) SubscribeLogs(output chan []byte, quit chan bool) error {
	resources, err := ListResources(p.App)
	fmt.Printf("err %+v\n", err)

	if err != nil {
		return err
	}

	done := make(chan bool)
	go subscribeKinesis(p.Name, resources[fmt.Sprintf("%sKinesis", upperName(p.Name))].Id, output, done)

	return nil
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

func (p *Process) Resources() Resources {
	resources, err := ListProcessResources(p.App, p.Name)

	if err != nil {
		panic(err)
	}

	return resources
}

func (p *Process) Userdata() string {
	return `""`
}

func processesTable(app string) string {
	return fmt.Sprintf("%s-processes", app)
}

func processFromItem(item map[string]dynamodb.AttributeValue) *Process {
	return &Process{
		Name:  coalesce(item["name"].S, ""),
		Count: coalesce(item["count"].S, "0"),
		App:   coalesce(item["app"].S, ""),
	}
}
