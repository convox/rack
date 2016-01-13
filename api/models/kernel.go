package models

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
)

const KERNEL_URL = "https://convox.s3.amazonaws.com/kernel.json";

func KernelUpdate() error {
	stackName := os.Getenv("STACK_NAME")

	if stackName != "" {
		res, err := CloudFormation().DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String(stackName)})

		if err != nil {
			return err
		}

		if len(res.Stacks) != 1 {
			return fmt.Errorf("could not find stack: %s", stackName)
		}

		stack := res.Stacks[0]

		params := map[string]string{}

		for _, p := range stack.Parameters {
			params[*p.ParameterKey] = *p.ParameterValue
		}

		latest, err := latestAmi()

		if err != nil {
			return err
		}

		params["AMI"] = latest

		// backwards compatibility
		delete(params, "WebCommand")
		delete(params, "WebPorts")

		stackParams := []*cloudformation.Parameter{}

		for key, value := range params {
			stackParams = append(stackParams, &cloudformation.Parameter{
				ParameterKey:   aws.String(key),
				ParameterValue: aws.String(value),
			})
		}

		gr, err := http.Get(KERNEL_URL)

		if err != nil {
			return err
		}

		defer gr.Body.Close()

		formation, err := ioutil.ReadAll(gr.Body)

		if err != nil {
			return err
		}

		existing, err := formationParameters(string(formation))

		if err != nil {
			return err
		}

		finalParams := []*cloudformation.Parameter{}

		// remove any params that do not exist in the formation
		for _, sp := range stackParams {
			if _, ok := existing[*sp.ParameterKey]; ok {
				finalParams = append(finalParams, sp)
			}
		}

		req := &cloudformation.UpdateStackInput{
			StackName:    aws.String(stackName),
			TemplateURL:  aws.String(KERNEL_URL),
			Capabilities: []*string{aws.String("CAPABILITY_IAM")},
			Parameters:   finalParams,
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

		_, err = CloudFormation().UpdateStack(req)

		if err != nil {
			return err
		}
	}

	return nil
}

func latestAmi() (string, error) {
	res, err := http.Get("https://convox.s3.amazonaws.com/ami.latest")

	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return "", err
	}

	return string(data), nil
}
