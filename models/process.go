package models

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/s3"
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
		if strings.Index(err.Error(), "does not exist") != -1 {
			return Processes{}, nil
		}

		return nil, err
	}

	req := &s3.ListObjectsInput{
		Bucket: aws.String(a.Outputs["Settings"]),
		Prefix: aws.String("process/"),
	}

	res, err := S3().ListObjects(req)

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

	value, err := s3Get(a.Outputs["Settings"], fmt.Sprintf("process/%s", name))

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

	return s3Put(app.Outputs["Settings"], fmt.Sprintf("process/%s", p.Name), data, false)
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
