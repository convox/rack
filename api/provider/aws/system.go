package aws

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/convox/rack/api/structs"
)

func (p *AWSProvider) SystemGet() (*structs.System, error) {
	rack := os.Getenv("RACK")

	res, err := p.CachedDescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(rack),
	})

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

	r := &structs.System{
		Count:   count,
		Name:    rack,
		Status:  humanStatus(*stack.StackStatus),
		Type:    params["InstanceType"],
		Version: os.Getenv("RELEASE"),
	}

	return r, nil
}

func (p *AWSProvider) SystemSave(system *structs.System) error {
	rack := os.Getenv("RACK")

	if system.Count < 2 {
		return fmt.Errorf("rack cannot be scaled below 2 instances")
	}

	capacity, err := p.CapacityGet()

	if err != nil {
		return err
	}

	requiredInstances := int(capacity.ProcessWidth) + 1

	if system.Count < requiredInstances {
		return fmt.Errorf("your process concurrency requires at least %d instances in the rack", requiredInstances)
	}

	params := map[string]string{
		"InstanceCount": strconv.Itoa(system.Count),
		"InstanceType":  system.Type,
		"Version":       system.Version,
	}

	template := fmt.Sprintf("https://convox.s3.amazonaws.com/release/%s/formation.json", system.Version)

	if system.Version != os.Getenv("RELEASE") {
		_, err := p.dynamodb().PutItem(&dynamodb.PutItemInput{
			Item: map[string]*dynamodb.AttributeValue{
				"id":      &dynamodb.AttributeValue{S: aws.String(system.Version)},
				"app":     &dynamodb.AttributeValue{S: aws.String(rack)},
				"created": &dynamodb.AttributeValue{S: aws.String(time.Now().Format(SortableTime))},
			},
			TableName: aws.String(releasesTable(rack)),
		})

		if err != nil {
			return err
		}
	}

	err = p.stackUpdate(rack, template, params)

	if awsError(err) == "ValidationError" {
		switch {
		case strings.Index(err.Error(), "No updates are to be performed") > -1:
			return fmt.Errorf("no system updates are to be performed")
		case strings.Index(err.Error(), "can not be updated") > -1:
			return fmt.Errorf("system is already updating")
		}
	}

	if err != nil {
		return err
	}

	return nil
}
