package models

import (
	"strings"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/cloudformation"
)

type Resource struct {
	Id   string
	Name string

	Reason string
	Status string
	Type   string

	Time time.Time
}

type Resources map[string]Resource

func ListResources(app string) (Resources, error) {
	res, err := CloudFormation().DescribeStackResources(&cloudformation.DescribeStackResourcesInput{StackName: aws.String(app)})

	if err != nil {
		return nil, err
	}

	resources := make(Resources)

	for _, r := range res.StackResources {
		resources[*r.LogicalResourceID] = Resource{
			Id:     cs(r.PhysicalResourceID, ""),
			Name:   cs(r.LogicalResourceID, ""),
			Reason: cs(r.ResourceStatusReason, ""),
			Status: cs(r.ResourceStatus, ""),
			Type:   cs(r.ResourceType, ""),
			Time:   ct(r.Timestamp),
		}
	}

	return resources, nil
}

func ListProcessResources(app, process string) (Resources, error) {
	res, err := ListResources(app)

	if err != nil {
		return nil, err
	}

	resources := make(Resources)

	prefix := UpperName(process)

	for name, resource := range res {
		if strings.HasPrefix(name, prefix) {
			resources[name] = resource
		}
	}

	return resources, nil
}
