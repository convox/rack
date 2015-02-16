package models

import (
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/cloudformation"
)

type Resource struct {
	Name       string
	PhysicalId string
}

type Resources map[string]Resource

func ListResources(app string) (Resources, error) {
	res, err := CloudFormation.DescribeStackResources(&cloudformation.DescribeStackResourcesInput{StackName: aws.String(app)})

	if err != nil {
		return nil, err
	}

	resources := make(Resources)

	for _, r := range res.StackResources {
		resources[*r.LogicalResourceID] = Resource{
			Name:       *r.LogicalResourceID,
			PhysicalId: *r.PhysicalResourceID,
		}
	}

	return resources, nil
}
