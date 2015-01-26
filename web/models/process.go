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
	return nil, nil
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

func (p *Process) Balancer() string {
	if p.Name == "web" {
		return "foo"
	} else {
		return ""
	}
}

func (p *Process) Formation(env string) (string, error) {
	p.Ports = []int{3000}

	params := map[string]interface{}{
		"Env":     env,
		"Process": p,
	}

	formation, err := buildTemplate("process", "formation", params)

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

func processFromRow(row map[string]*dynamodb.Attribute) *Process {
	return &Process{
		Name:  coalesce(row["id"], ""),
		Count: coalesce(row["release"], ""),
	}
}

func processesTable(app string) *dynamodb.Table {
	pk := dynamodb.PrimaryKey{dynamodb.NewStringAttribute("name", ""), nil}
	table := DynamoDB.NewTable(fmt.Sprintf("convox-%s-processes", app), pk)
	return table
}
