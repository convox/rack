package models

import (
	"fmt"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/dynamodb"
)

type Resource struct {
	Name string
	Type string

	App string
}

type Resources []Resource

func ListResources(app string) (Resources, error) {
	rows, err := resourcesTable(app).Scan(nil)

	if err != nil {
		return nil, err
	}

	resources := make(Resources, len(rows))

	for i, row := range rows {
		resources[i] = *resourceFromRow(row)
	}

	return resources, nil
}

func (r *Resource) Save() error {
	resource := []dynamodb.Attribute{
		*dynamodb.NewStringAttribute("name", r.Name),
		*dynamodb.NewStringAttribute("type", r.Type),
		*dynamodb.NewStringAttribute("app", r.App),
	}

	_, err := resourcesTable(r.App).PutItem(r.Name, "", resource)

	return err
}

func (r *Resource) Env() (string, error) {
	env, err := buildTemplate(r.Type, "env", r)

	if err != nil {
		return "", err
	}

	return env, nil
}

func (r *Resource) Formation() (string, error) {
	formation, err := buildTemplate(r.Type, "formation", r)

	if err != nil {
		return "", err
	}

	return formation, nil
}

func (r Resource) AvailabilityZones() []string {
	azs := []string{}

	for _, subnet := range ListSubnets() {
		azs = append(azs, subnet.AvailabilityZone)
	}

	return azs
}

func (r Resource) FormationName() string {
	return fmt.Sprintf("%s%s", upperName(r.Type), upperName(r.Name))
}

func resourceFromRow(row map[string]*dynamodb.Attribute) *Resource {
	return &Resource{
		Name: coalesce(row["name"], ""),
		Type: coalesce(row["type"], ""),
		App:  coalesce(row["app"], ""),
	}
}

func resourcesTable(app string) *dynamodb.Table {
	pk := dynamodb.PrimaryKey{dynamodb.NewStringAttribute("name", ""), nil}
	table := DynamoDB.NewTable(fmt.Sprintf("convox-%s-resources", app), pk)
	return table
}
