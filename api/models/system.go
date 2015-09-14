package models

import (
	"fmt"
	"os"
	"strconv"

	"/github.com/awslabs/aws-sdk-go/aws"
	"/github.com/awslabs/aws-sdk-go/service/cloudformation"
)

type System struct {
	Count   int    `json:"count"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Type    string `json:"type"`
	Version string `json:"version"`
}

func GetSystem() (*System, error) {
	rack := os.Getenv("RACK")

	res, err := CloudFormation().DescribeStacks(&cloudformation.DescribeStacksInput{StackName: aws.String(rack)})

	if err != nil {
		return nil, err
	}

	if len(res.Stacks) != 1 {
		return nil, fmt.Errorf("could not load stack for app: %s", rack)
	}

	stack := res.Stacks[0]
	params := stackParameters(stack)

	count, err := strconv.Atoi(params["InstanceCount"])

	if err != nil {
		return nil, err
	}

	r := &System{
		Count:   count,
		Name:    rack,
		Status:  humanStatus(*stack.StackStatus),
		Type:    params["InstanceType"],
		Version: os.Getenv("RELEASE"),
	}

	return r, nil
}

func (r *System) Save() error {
	rack := os.Getenv("RACK")

	app, err := GetApp(rack)

	if err != nil {
		return err
	}

	params := map[string]string{
		"InstanceCount": strconv.Itoa(r.Count),
		"InstanceType":  r.Type,
		"Version":       r.Version,
	}

	template := fmt.Sprintf("https://convox.s3.amazonaws.com/release/%s/formation.json", r.Version)

	return app.UpdateParamsAndTemplate(params, template)
}
