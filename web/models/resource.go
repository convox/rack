package models

import "fmt"

type Resource struct {
	Name       string
	PhysicalId string
}

type Resources map[string]Resource

func ListResources(app string) (Resources, error) {
	res, err := CloudFormation.DescribeStackResources(fmt.Sprintf("convox-%s", app), "", "")

	if err != nil {
		return nil, err
	}

	resources := make(Resources)

	for _, r := range res.StackResources {
		resources[r.LogicalResourceId] = Resource{
			Name:       r.LogicalResourceId,
			PhysicalId: r.PhysicalResourceId,
		}
	}

	return resources, nil
}
