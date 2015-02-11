package models

import (
	"fmt"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/dynamodb"
)

type Service struct {
	Name string
	Type string

	App string
}

type Services []Service

func ListServices(app string) (Services, error) {
	rows, err := servicesTable(app).Scan(nil)

	if err != nil {
		return nil, err
	}

	services := make(Services, len(rows))

	for i, row := range rows {
		services[i] = *serviceFromRow(row)
	}

	return services, nil
}

func (r *Service) Save() error {
	service := []dynamodb.Attribute{
		*dynamodb.NewStringAttribute("name", r.Name),
		*dynamodb.NewStringAttribute("type", r.Type),
		*dynamodb.NewStringAttribute("app", r.App),
	}

	_, err := servicesTable(r.App).PutItem(r.Name, "", service)

	return err
}

func (r *Service) Env() (string, error) {
	env, err := buildFormationTemplate(r.Type, "env", r)

	if err != nil {
		return "", err
	}

	return env, nil
}

func (r *Service) Formation() (string, error) {
	formation, err := buildFormationTemplate(r.Type, "formation", r)

	if err != nil {
		return "", err
	}

	return formation, nil
}

func (r Service) AvailabilityZones() []string {
	azs := []string{}

	for _, subnet := range ListSubnets() {
		azs = append(azs, subnet.AvailabilityZone)
	}

	return azs
}

func (r Service) FormationName() string {
	return fmt.Sprintf("%s%s", upperName(r.Type), upperName(r.Name))
}

func serviceFromRow(row map[string]*dynamodb.Attribute) *Service {
	return &Service{
		Name: coalesce(row["name"], ""),
		Type: coalesce(row["type"], ""),
		App:  coalesce(row["app"], ""),
	}
}

func servicesTable(app string) *dynamodb.Table {
	pk := dynamodb.PrimaryKey{dynamodb.NewStringAttribute("name", ""), nil}
	table := DynamoDB.NewTable(fmt.Sprintf("convox-%s-resources", app), pk)
	return table
}
