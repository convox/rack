package models

import (
	"fmt"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ec2"
)

type Process struct {
	Name    string
	Command string
	Count   int
	Ports   []int

	App string
}

type Processes []Process

func ListProcesses(app string) (Processes, error) {
	a, err := GetApp(app)

	if err != nil {
		return nil, err
	}

	if a.Release == "" {
		release, err := a.LatestRelease()

		if err != nil {
			return nil, err
		}

		manifest, err := LoadManifest(release.Manifest)

		if err != nil {
			return nil, err
		}

		return manifest.Processes(), nil
	}

	// TODO: change the last filter to tag:App eventually

	req := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{Name: aws.String("tag:System"), Values: []*string{aws.String("convox")}},
			&ec2.Filter{Name: aws.String("tag:Type"), Values: []*string{aws.String("app")}},
			&ec2.Filter{Name: aws.String("tag:aws:cloudformation:stack-name"), Values: []*string{aws.String(app)}},
		},
	}

	res, err := EC2().DescribeInstances(req)

	if err != nil {
		return nil, err
	}

	processes := map[string]Process{}

	for _, r := range res.Reservations {
		for _, i := range r.Instances {
			tags := map[string]string{}

			for _, t := range i.Tags {
				tags[*t.Key] = *t.Value
			}

			parts := strings.SplitN(tags["Name"], "-", 2)

			if len(parts) != 2 {
				continue
			}

			name := parts[1]

			if p, ok := processes[name]; ok {
				p.Count += 1
				processes[name] = p
			} else {
				processes[name] = Process{
					Name:  name,
					Count: 1,
					App:   app,
				}
			}
		}
	}

	pp := Processes{}

	for _, process := range processes {
		pp = append(pp, process)
	}

	return pp, nil
}

func GetProcess(app, name string) (*Process, error) {
	req := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{Name: aws.String("tag:System"), Values: []*string{aws.String("convox")}},
			&ec2.Filter{Name: aws.String("tag:Type"), Values: []*string{aws.String("app")}},
			&ec2.Filter{Name: aws.String("tag:App"), Values: []*string{aws.String(app)}},
		},
	}

	res, err := EC2().DescribeInstances(req)

	if err != nil {
		return nil, err
	}

	count := 0

	for _, r := range res.Reservations {
		count += len(r.Instances)
	}

	process := &Process{
		Name:  name,
		Count: count,
		App:   app,
	}

	return process, nil
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
