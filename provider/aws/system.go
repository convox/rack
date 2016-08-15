package aws

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/dynamodb"
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
		case strings.Contains(err.Error(), "No updates are to be performed"):
			return fmt.Errorf("no system updates are to be performed")
		case strings.Contains(err.Error(), "can not be updated"):
			return fmt.Errorf("system is already updating")
		}
	}
	return err
}

// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-types.html
var instanceTypes = []string{
	"c1.medium",
	"c1.xlarge",
	"c3.2xlarge",
	"c3.4xlarge",
	"c3.8xlarge",
	"c3.large",
	"c3.xlarge",
	"c4.2xlarge",
	"c4.4xlarge",
	"c4.8xlarge",
	"c4.large",
	"c4.xlarge",
	"cc1.4xlarge",
	"cc2.8xlarge",
	"cg1.4xlarge",
	"cr1.8xlarge",
	"d2.2xlarge",
	"d2.4xlarge",
	"d2.8xlarge",
	"d2.xlarge",
	"g2.2xlarge",
	"g2.8xlarge",
	"hi1.4xlarge",
	"hs1.8xlarge",
	"i2.2xlarge",
	"i2.4xlarge",
	"i2.8xlarge",
	"i2.xlarge",
	"m1.large",
	"m1.medium",
	"m1.small",
	"m1.xlarge",
	"m2.2xlarge",
	"m2.4xlarge",
	"m2.xlarge",
	"m3.2xlarge",
	"m3.large",
	"m3.medium",
	"m3.xlarge",
	"m4.10xlarge",
	"m4.2xlarge",
	"m4.4xlarge",
	"m4.large",
	"m4.xlarge",
	"r3.2xlarge",
	"r3.4xlarge",
	"r3.8xlarge",
	"r3.large",
	"r3.xlarge",
	"t1.micro",
	"t2.large",
	"t2.medium",
	"t2.micro",
	"t2.nano",
	"t2.small",
	"x1.16xlarge",
	"x1.32xlarge",
	"x1.4xlarge",
	"x1.8xlarge",
}
