package models

import (
	"fmt"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/dynamodb"
)

type Process struct {
	Name  string
	Count string
	Ports []int

	App string
}

type Processes []Process

func ListProcesses(app string) (Processes, error) {
	rows, err := processesTable(app).Scan(nil)

	if err != nil {
		return nil, err
	}

	processes := make(Processes, len(rows))

	for i, row := range rows {
		processes[i] = *processFromRow(row)
	}

	return processes, nil
}

func (p *Process) Save() error {
	process := []dynamodb.Attribute{
		*dynamodb.NewStringAttribute("name", p.Name),
		*dynamodb.NewStringAttribute("count", p.Count),
		*dynamodb.NewStringAttribute("app", p.App),
	}

	_, err := processesTable(p.App).PutItem(p.Name, "", process)

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

func (p *Process) Metrics() *Metrics {
	metrics, err := ProcessMetrics(p.App, p.Name)

	if err != nil {
		panic(err)
	}

	return metrics
}

func processFromRow(row map[string]*dynamodb.Attribute) *Process {
	return &Process{
		Name:  coalesce(row["name"], ""),
		Count: coalesce(row["count"], "0"),
		App:   coalesce(row["app"], ""),
	}
}

func processesTable(app string) *dynamodb.Table {
	pk := dynamodb.PrimaryKey{dynamodb.NewStringAttribute("name", ""), nil}
	table := DynamoDB.NewTable(fmt.Sprintf("convox-%s-processes", app), pk)
	return table
}
