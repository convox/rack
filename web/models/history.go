package models

import (
	"fmt"
	"time"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/cloudformation"
)

type History struct {
	Id   string
	Name string

	Reason string
	Status string
	Type   string

	Time time.Time
}

type Histories []History

func ListHistories(app string) (Histories, error) {
	histories := Histories{}

	next := ""

	for {
		req := &cloudformation.DescribeStackEventsInput{StackName: aws.String(fmt.Sprintf("convox-%s", app))}

		if next != "" {
			req.NextToken = aws.String(next)
		}

		res, err := CloudFormation.DescribeStackEvents(req)

		if err != nil {
			return nil, err
		}

		for _, event := range res.StackEvents {
			histories = append(histories, History{
				Id:     *event.EventID,
				Name:   coalesce(event.LogicalResourceID, ""),
				Status: coalesce(event.ResourceStatus, ""),
				Type:   coalesce(event.ResourceType, ""),
				Reason: coalesce(event.ResourceStatusReason, ""),
				Time:   event.Timestamp,
			})
		}

		if res.NextToken == nil {
			break
		}

		next = *res.NextToken
	}

	return histories, nil
}
