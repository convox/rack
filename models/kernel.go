package models

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/cloudformation"
)

func KernelUpdate() error {
	stackName := os.Getenv("STACK_NAME")

	if stackName != "" {
		res, err := CloudFormation.DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String(stackName)})

		if err != nil {
			return err
		}

		if len(res.Stacks) != 1 {
			return fmt.Errorf("could not find stack: %s", stackName)
		}

		stack := res.Stacks[0]

		fmt.Printf("stack %+v\n", stack)

		req := &cloudformation.UpdateStackInput{
			StackName:    aws.String(stackName),
			TemplateURL:  aws.String("http://convox.s3.amazonaws.com/formation.json"),
			Capabilities: []string{"CAPABILITY_IAM"},
			Parameters:   stack.Parameters,
		}

		for _, p := range req.Parameters {
			if *p.ParameterKey == "AMI" {
				latest, err := latestAmi()

				if err != nil {
					return err
				}

				p.ParameterValue = aws.String(latest)
			}
		}

		_, err = CloudFormation.UpdateStack(req)

		if err != nil {
			return err
		}
	}

	return nil
}

func latestAmi() (string, error) {
	res, err := http.Get("http://convox.s3.amazonaws.com/ami.latest")

	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return "", err
	}

	return string(data), nil
}
