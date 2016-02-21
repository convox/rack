package aws

import (
	"fmt"
	"os"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/api/structs"
)

func (p *AWSProvider) AppGet(name string) (*structs.App, error) {
	res, err := p.CachedDescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(name),
	})

	if err != nil {
		return nil, err
	}

	if len(res.Stacks) != 1 {
		return nil, fmt.Errorf("could not load stack for app: %s", name)
	}

	tags := stackTags(res.Stacks[0])

	if tags["Rack"] != "" && tags["Rack"] != os.Getenv("RACK") {
		return nil, fmt.Errorf("no such app: %s", name)
	}

	app := appFromStack(res.Stacks[0])

	return &app, nil
}

func appFromStack(stack *cloudformation.Stack) structs.App {
	return structs.App{
		Name:       *stack.StackName,
		Release:    stackParameters(stack)["Release"],
		Status:     humanStatus(*stack.StackStatus),
		Outputs:    stackOutputs(stack),
		Parameters: stackParameters(stack),
		Tags:       stackTags(stack),
	}
}
