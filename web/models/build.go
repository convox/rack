package models

import (
	"fmt"
	"time"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/dynamodb"
)

type Build struct {
	Id string

	Logs    string
	Release string
	Status  string

	Created time.Time
	Ended   time.Time
}

type Builds []Build

func ListBuilds(app string) (Builds, error) {
	req := &dynamodb.QueryInput{
		KeyConditions: map[string]dynamodb.Condition{
			"app": dynamodb.Condition{
				AttributeValueList: []dynamodb.AttributeValue{
					dynamodb.AttributeValue{S: aws.String(app)},
				},
				ComparisonOperator: aws.String("EQ"),
			},
		},
		IndexName:        aws.String("app.created"),
		Limit:            aws.Integer(10),
		ScanIndexForward: aws.Boolean(false),
		TableName:        aws.String(buildsTable(app)),
	}

	res, err := DynamoDB.Query(req)

	if err != nil {
		return nil, err
	}

	builds := make(Builds, len(res.Items))

	for i, item := range res.Items {
		builds[i] = *buildFromItem(item)
	}

	return builds, nil
}

func buildsTable(app string) string {
	return fmt.Sprintf("%s-builds", app)
}

func buildFromItem(item map[string]dynamodb.AttributeValue) *Build {
	created, _ := time.Parse(SortableTime, coalesce(item["created"].S, ""))
	ended, _ := time.Parse(SortableTime, coalesce(item["ended"].S, ""))

	return &Build{
		Id:      coalesce(item["id"].S, ""),
		Logs:    coalesce(item["logs"].S, ""),
		Release: coalesce(item["release"].S, ""),
		Status:  coalesce(item["status"].S, ""),
		Created: created,
		Ended:   ended,
	}
}
