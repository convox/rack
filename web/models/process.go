package models

import (
	"fmt"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/dynamodb"
)

type Process struct {
	Name  string
	Count string

	App string
}

type Processes []Process

func ListProcesses(app string) (Processes, error) {
	return nil, nil
}

func (p *Process) Save() error {
	return nil
}

func (p *Process) Balancer() string {
	if p.Name == "web" {
		return "foo"
	} else {
		return ""
	}
}

func (p *Process) Formation() (string, error) {
	formation, err := buildTemplate("process", "formation", p)

	if err != nil {
		return "", err
	}

	fmt.Printf("formation %+v\n", formation)

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

func processTable(app string) *dynamodb.Table {
	pk := dynamodb.PrimaryKey{dynamodb.NewStringAttribute("name", ""), nil}
	table := DynamoDB.NewTable(fmt.Sprintf("convox-%s-processs", app), pk)
	return table
}
