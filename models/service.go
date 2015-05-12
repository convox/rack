package models

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/dynamodb"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/s3"
)

type Service struct {
	Name string
	Type string

	App string
}

type Services []Service

func ListServices(app string) (Services, error) {
	a, err := GetApp(app)

	if err != nil {
		if strings.Index(err.Error(), "does not exist") != -1 {
			return Services{}, nil
		}

		return nil, err
	}

	req := &s3.ListObjectsInput{
		Bucket: aws.String(a.Outputs["Settings"]),
		Prefix: aws.String("service/"),
	}

	res, err := S3().ListObjects(req)

	services := make(Services, len(res.Contents))

	for i, s := range res.Contents {
		name := strings.TrimPrefix(*s.Key, "service/")
		svc, err := GetService(app, name)

		if err != nil {
			return nil, err
		}

		services[i] = *svc
	}

	return services, nil
}

func GetService(app, name string) (*Service, error) {
	a, err := GetApp(app)

	if err != nil {
		return nil, err
	}

	value, err := s3Get(a.Outputs["Settings"], fmt.Sprintf("service/%s", name))

	if err != nil {
		return nil, err
	}

	var service *Service

	err = json.Unmarshal([]byte(value), &service)

	if err != nil {
		return nil, err
	}

	return service, nil
}

func (s *Service) Save() error {
	app, err := GetApp(s.App)

	if err != nil {
		return err
	}

	data, err := json.Marshal(s)

	if err != nil {
		return err
	}

	return s3Put(app.Outputs["Settings"], fmt.Sprintf("service/%s", s.Name), data, false)
}

func (s *Service) ManagementUrl() string {
	region := os.Getenv("AWS_REGION")

	resources, err := ListResources(s.App)

	if err != nil {
		panic(err)
	}

	switch s.Type {
	case "convox/postgres":
		id := resources[fmt.Sprintf("%sDatabase", upperName(s.Name))].Id
		return fmt.Sprintf("https://console.aws.amazon.com/rds/home?region=%s#dbinstances:id=%s;sf=all", region, id)
	case "convox/redis":
		id := resources[fmt.Sprintf("%sInstances", upperName(s.Name))].Id
		return fmt.Sprintf("https://console.aws.amazon.com/ec2/autoscaling/home?region=%s#AutoScalingGroups:id=%s;view=details", region, id)
	default:
		return ""
	}
}

func servicesTable(app string) string {
	return fmt.Sprintf("%s-services", app)
}

func serviceFromItem(item map[string]dynamodb.AttributeValue) *Service {
	return &Service{
		Name: coalesce(item["name"].S, ""),
		Type: coalesce(item["type"].S, ""),
		App:  coalesce(item["app"].S, ""),
	}
}
