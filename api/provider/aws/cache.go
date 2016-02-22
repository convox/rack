package aws

import (
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/api/cache"
)

func (p *AWSProvider) CachedDescribeStacks(input *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
	key := ""

	if input != nil && input.StackName != nil {
		key = *input.StackName
	}

	res, ok := cache.Get("DescribeStacks", key).(*cloudformation.DescribeStacksOutput)

	if ok {
		return res, nil
	}

	res, err := p.cloudformation().DescribeStacks(input)

	if err != nil {
		return nil, err
	}

	err = cache.Set("DescribeStacks", key, res, 5*time.Second)

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (p *AWSProvider) CachedListContainerInstances(input *ecs.ListContainerInstancesInput) (*ecs.ListContainerInstancesOutput, error) {
	res, ok := cache.Get("ListContainerInstances", input).(*ecs.ListContainerInstancesOutput)

	if ok {
		return res, nil
	}

	res, err := p.ecs().ListContainerInstances(input)

	if err != nil {
		return nil, err
	}

	err = cache.Set("ListContainerInstances", input, res, 10*time.Second)

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (p *AWSProvider) CachedUpdateStack(input *cloudformation.UpdateStackInput) (*cloudformation.UpdateStackOutput, error) {
	cache.Clear("DescribeStacks", "")
	cache.Clear("DescribeStacks", *input.StackName)

	return p.cloudformation().UpdateStack(input)
}
