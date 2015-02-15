package models

import (
	"fmt"
	"strings"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/dynamodb"
)

type Service struct {
	Name string
	Type string

	App string
}

type Services []Service

func ListServices(app string) (Services, error) {
	req := &dynamodb.ScanInput{
		TableName: aws.String(servicesTable(app)),
	}

	res, err := DynamoDB.Scan(req)

	if err != nil {
		return nil, err
	}

	services := make(Services, len(res.Items))

	for i, item := range res.Items {
		services[i] = *serviceFromItem(item)
	}

	fmt.Printf("services %+v\n", services)

	return services, nil
}

func (r *Service) Save() error {
	req := &dynamodb.PutItemInput{
		Item: map[string]dynamodb.AttributeValue{
			"name": dynamodb.AttributeValue{S: aws.String(r.Name)},
			"type": dynamodb.AttributeValue{S: aws.String(r.Type)},
			"app":  dynamodb.AttributeValue{S: aws.String(r.App)},
		},
		TableName: aws.String(servicesTable(r.App)),
	}

	_, err := DynamoDB.PutItem(req)

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
	parts := strings.Split(r.Type, "/")
	name := upperName(parts[1])
	return fmt.Sprintf("%s%s", name, upperName(r.Name))
}

func servicesTable(app string) string {
	return fmt.Sprintf("convox-%s-services", app)
}

func serviceFromItem(item map[string]dynamodb.AttributeValue) *Service {
	return &Service{
		Name: coalesce(item["name"].S, ""),
		Type: coalesce(item["type"].S, ""),
		App:  coalesce(item["app"].S, ""),
	}
}
