package aws

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/convox/rack/api/structs"
)

func (p *AWSProvider) SystemGet() (*structs.System, error) {
	res, err := p.describeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(p.Rack),
	})
	if ae, ok := err.(awserr.Error); ok && ae.Code() == "ValidationError" {
		return nil, ErrorNotFound(fmt.Sprintf("%s not found", p.Rack))
	}
	if err != nil {
		return nil, err
	}
	if len(res.Stacks) != 1 {
		return nil, fmt.Errorf("could not load stack for app: %s", p.Rack)
	}

	stack := res.Stacks[0]
	params := stackParameters(stack)

	count, err := strconv.Atoi(params["InstanceCount"])

	if err != nil {
		return nil, err
	}

	r := &structs.System{
		Count:   count,
		Name:    p.Rack,
		Region:  p.Region,
		Status:  humanStatus(*stack.StackStatus),
		Type:    params["InstanceType"],
		Version: params["Version"],
	}

	return r, nil
}

// SystemLogs streams logs for the Rack
func (p *AWSProvider) SystemLogs(w io.Writer, opts structs.LogStreamOptions) error {
	system, err := p.describeStack(p.Rack)
	if err != nil {
		return err
	}

	// if strings.HasSuffix(err.Error(), "write: broken pipe") {
	//   return nil
	// }

	return p.subscribeLogs(w, stackOutputs(system)["LogGroup"], opts)
}

// SystemReleases lists the latest releases of the rack
func (p *AWSProvider) SystemReleases() (structs.Releases, error) {
	req := &dynamodb.QueryInput{
		KeyConditions: map[string]*dynamodb.Condition{
			"app": &dynamodb.Condition{
				AttributeValueList: []*dynamodb.AttributeValue{
					&dynamodb.AttributeValue{S: aws.String(p.Rack)},
				},
				ComparisonOperator: aws.String("EQ"),
			},
		},
		IndexName:        aws.String("app.created"),
		Limit:            aws.Int64(20),
		ScanIndexForward: aws.Bool(false),
		TableName:        aws.String(p.DynamoReleases),
	}

	res, err := p.dynamodb().Query(req)
	if err != nil {
		return nil, err
	}

	releases := make(structs.Releases, len(res.Items))

	for i, item := range res.Items {
		releases[i] = *releaseFromItem(item)
	}

	return releases, nil
}

func (p *AWSProvider) SystemSave(system structs.System) error {
	typeValid := false
	// Better search method could work here if needed
	// sort.SearchString() return value doesn't indicate if string is not in slice
	for _, itype := range instanceTypes {
		if itype == system.Type {
			typeValid = true
			break
		}
	}
	if !typeValid {
		return fmt.Errorf("invalid instance type: %s", system.Type)
	}

	// FIXME
	// mac, err := maxAppConcurrency()

	// // dont scale the rack below the max concurrency plus one
	// // see formation.go for more details
	// if err == nil && r.Count < (mac+1) {
	//   return fmt.Errorf("max process concurrency is %d, can't scale rack below %d instances", mac, mac+1)
	// }

	template := fmt.Sprintf("https://convox.s3.amazonaws.com/release/%s/formation.json", system.Version)

	params := map[string]string{
		"InstanceCount": strconv.Itoa(system.Count),
		"InstanceType":  system.Type,
		"Version":       system.Version,
	}

	stack, err := p.describeStack(p.Rack)
	if err != nil {
		return err
	}

	// if there is a version update then record it
	if system.Version != stackParameters(stack)["Version"] {
		_, err := p.dynamodb().PutItem(&dynamodb.PutItemInput{
			Item: map[string]*dynamodb.AttributeValue{
				"id":      &dynamodb.AttributeValue{S: aws.String(system.Version)},
				"app":     &dynamodb.AttributeValue{S: aws.String(p.Rack)},
				"created": &dynamodb.AttributeValue{S: aws.String(p.createdTime())},
			},
			TableName: aws.String(p.DynamoReleases),
		})

		if err != nil {
			return err
		}
	}

	err = p.updateStack(p.Rack, template, params)
	if awsError(err) == "ValidationError" {
		switch {
		case strings.Contains(err.Error(), "No updates are to be performed"):
			return fmt.Errorf("no system updates are to be performed")
		case strings.Contains(err.Error(), "can not be updated"):
			return fmt.Errorf("system is already updating")
		}
	}

	return err
}
