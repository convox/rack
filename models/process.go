package models

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ecs"
)

type Process struct {
	Name    string
	Command string
	Count   int

	App string
}

type Processes []Process

func ListProcesses(app string) (Processes, error) {
	a, err := GetApp(app)

	if err != nil {
		return nil, err
	}

	res, err := ECS().ListServices(&ecs.ListServicesInput{Cluster: aws.String(a.Cluster)})

	if err != nil {
		return nil, err
	}

	req := &ecs.DescribeServicesInput{
		Cluster:  aws.String(a.Cluster),
		Services: res.ServiceARNs,
	}

	sres, err := ECS().DescribeServices(req)

	if err != nil {
		return nil, err
	}

	ps := Processes{}

	for _, s := range sres.Services {
		parts := strings.Split(*s.ServiceName, "-")
		app := strings.Join(parts[0:len(parts)-1], "-")
		name := parts[len(parts)-1]

		if app == a.Name {
			ps = append(ps, Process{
				App:   app,
				Name:  name,
				Count: int(*s.DesiredCount),
			})
		}
	}

	return ps, nil
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

func (p *Process) Ports() map[string]string {
	app, err := GetApp(p.App)

	if err != nil {
		return map[string]string{}
	}

	ports := map[string]string{}

	for key, value := range app.Outputs {
		r := regexp.MustCompile(fmt.Sprintf("%sPort([0-9]+)Balancer", upperName(p.Name)))

		if matches := r.FindStringSubmatch(key); len(matches) == 2 {
			ports[matches[1]] = value
		}
	}

	return ports
}

func (p *Process) BalancerHost() string {
	app, err := GetApp(p.App)

	if err != nil {
		return ""
	}

	return app.Outputs[fmt.Sprintf("%sBalancerHost", upperName(p.Name))]
}

func (p *Process) BalancerPorts() map[string]string {
	host := p.BalancerHost()

	bp := map[string]string{}

	for original, current := range p.Ports() {
		bp[original] = fmt.Sprintf("%s:%s", host, current)
	}

	return bp
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
