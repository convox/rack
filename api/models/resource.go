package models

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/api/cache"
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
	if resources, ok := cache.Get("ListResources", app).(Resources); ok {
		return resources, nil
	}

	stackName := shortNameToStackName(app)

	res, err := CloudFormation().DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(stackName),
	})

	if app != stackName && awsError(err) == "ValidationError" {
		res, err = CloudFormation().DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
			StackName: aws.String(app),
		})
	}

	if err != nil {
		return nil, err
	}

	resources := make(Resources, len(res.StackResources))

	for _, r := range res.StackResources {
		resources[*r.LogicalResourceId] = Resource{
			Id:     cs(r.PhysicalResourceId, ""),
			Name:   cs(r.LogicalResourceId, ""),
			Reason: cs(r.ResourceStatusReason, ""),
			Status: cs(r.ResourceStatus, ""),
			Type:   cs(r.ResourceType, ""),
			Time:   ct(r.Timestamp),
		}
	}

	err = cache.Set("ListResources", app, resources, 15*time.Second)

	if err != nil {
		return nil, err
	}

	return resources, nil
}
