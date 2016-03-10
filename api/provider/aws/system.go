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

	res, err := p.describeStacks(&cloudformation.DescribeStacksInput{
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
		Region:  os.Getenv("AWS_REGION"),
		Status:  humanStatus(*stack.StackStatus),
		Type:    params["InstanceType"],
		Version: os.Getenv("RELEASE"),
	}

	return r, nil
}

func (p *AWSProvider) SystemSave(system structs.System) error {
	rack := os.Getenv("RACK")

	// FIXME
	// mac, err := maxAppConcurrency()

	// // dont scale the rack below the max concurrency plus one
	// // see formation.go for more details
	// if err == nil && r.Count < (mac+1) {
	//   return fmt.Errorf("max process concurrency is %d, can't scale rack below %d instances", mac, mac+1)
	// }

	app, err := p.AppGet(rack)

	if err != nil {
		return err
	}

	params := map[string]string{
		"InstanceCount": strconv.Itoa(system.Count),
		"InstanceType":  system.Type,
		"Version":       system.Version,
	}

	template := fmt.Sprintf("https://convox.s3.amazonaws.com/release/%s/formation.json", system.Version)

	if system.Version != app.Parameters["Version"] {
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

	err = p.stackUpdate(app.Name, template, params)

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
