package aws

import (
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/api/cache"
)

func (p *AWSProvider) describeStacks(input *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
	res, ok := cache.Get("describeStacks", input.StackName).(*cloudformation.DescribeStacksOutput)

	if ok {
		return res, nil
	}

	res, err := p.cloudformation().DescribeStacks(input)

	if err != nil {
		return nil, err
	}

	err = cache.Set("describeStacks", input.StackName, res, 5*time.Second)

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (p *AWSProvider) listContainerInstances(input *ecs.ListContainerInstancesInput) (*ecs.ListContainerInstancesOutput, error) {
	res, ok := cache.Get("listContainerInstances", input).(*ecs.ListContainerInstancesOutput)

	if ok {
		return res, nil
	}

	res, err := p.ecs().ListContainerInstances(input)

	if err != nil {
		return nil, err
	}

	err = cache.Set("listContainerInstances", input, res, 10*time.Second)

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (p *AWSProvider) updateStack(input *cloudformation.UpdateStackInput) (*cloudformation.UpdateStackOutput, error) {
	cache.Clear("describeStacks", nil)
	cache.Clear("describeStacks", input.StackName)

	return p.cloudformation().UpdateStack(input)
}
