package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/dynamodb"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/s3"
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
		return nil, err
	}

	req := &s3.ListObjectsRequest{
		Bucket: aws.String(a.Outputs["Settings"]),
		Prefix: aws.String("service/"),
	}

	res, err := S3.ListObjects(req)

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

	req := &s3.GetObjectRequest{
		Bucket: aws.String(a.Outputs["Settings"]),
		Key:    aws.String(fmt.Sprintf("service/%s", name)),
	}

	res, err := S3.GetObject(req)

	if err != nil {
		return nil, err
	}

	value, err := ioutil.ReadAll(res.Body)

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

	req := &s3.PutObjectRequest{
		Body:          ioutil.NopCloser(bytes.NewReader(data)),
		Bucket:        aws.String(app.Outputs["Settings"]),
		ContentLength: aws.Long(int64(len(data))),
		Key:           aws.String(fmt.Sprintf("service/%s", s.Name)),
	}

	_, err = S3.PutObject(req)

	return err
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
