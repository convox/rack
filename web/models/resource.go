package models

import (
	"fmt"
	"time"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/cloudformation"
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
	res, err := CloudFormation.DescribeStackResources(&cloudformation.DescribeStackResourcesInput{StackName: aws.String(app)})

	if err != nil {
		return nil, err
	}

	resources := make(Resources)

	for _, r := range res.StackResources {
		fmt.Printf("r %+v\n", r)
		resources[*r.LogicalResourceID] = Resource{
			Id:     coalesce(r.PhysicalResourceID, ""),
			Name:   coalesce(r.LogicalResourceID, ""),
			Reason: coalesce(r.ResourceStatusReason, ""),
			Status: coalesce(r.ResourceStatus, ""),
			Type:   coalesce(r.ResourceType, ""),
			Time:   r.Timestamp,
		}
	}

	return resources, nil
}
