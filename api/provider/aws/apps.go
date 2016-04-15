package aws

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/api/structs"
)

func (p *AWSProvider) AppGet(name string) (*structs.App, error) {
	var res *cloudformation.DescribeStacksOutput
	var err error

	if name == os.Getenv("RACK") {
		res, err = p.describeStacks(&cloudformation.DescribeStacksInput{
			StackName: aws.String(name),
		})
	} else {
		// try 'convox-myapp', and if not found try 'myapp'
		res, err = p.describeStacks(&cloudformation.DescribeStacksInput{
			StackName: aws.String(os.Getenv("RACK") + "-" + name),
		})

		if awsError(err) == "ValidationError" {
			res, err = p.describeStacks(&cloudformation.DescribeStacksInput{
				StackName: aws.String(name),
			})
		}
	}

	if err != nil {
		return nil, err
	}

	if len(res.Stacks) != 1 {
		return nil, fmt.Errorf("could not load stack for app: %s", name)
	}

	app := appFromStack(res.Stacks[0])

	if app.Tags["Rack"] != "" && app.Tags["Rack"] != os.Getenv("RACK") {
		return nil, fmt.Errorf("no such app on this rack: %s", name)
	}

	return &app, nil
}

func appFromStack(stack *cloudformation.Stack) structs.App {
	name := *stack.StackName
	tags := stackTags(stack)
	if value, ok := tags["Name"]; ok {
		// StackName probably includes the Rack prefix, prefer Name tag.
		name = value
	}

	return structs.App{
		Name:       name,
		Release:    stackParameters(stack)["Release"],
		Status:     humanStatus(*stack.StackStatus),
		Outputs:    stackOutputs(stack),
		Parameters: stackParameters(stack),
		Tags:       stackTags(stack),
	}
}
