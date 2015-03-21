package models

import (
	"fmt"
	"time"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	// "github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/cloudformation"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/dynamodb"
)

type Event struct {
	App     string
	Created time.Time

	Metadata string
	State    string
	Type     string
	User     string
}

type Events []Event

func ListEvents(app string) (Events, error) {
	req := &dynamodb.QueryInput{
		KeyConditions: map[string]dynamodb.Condition{
			"app": dynamodb.Condition{
				AttributeValueList: []dynamodb.AttributeValue{
					dynamodb.AttributeValue{S: aws.String(app)},
				},
				ComparisonOperator: aws.String("EQ"),
			},
		},
		Limit:            aws.Integer(10),
		ScanIndexForward: aws.Boolean(false),
		TableName:        aws.String(eventsTable(app)),
	}

	res, err := DynamoDB.Query(req)

	if err != nil {
		return nil, err
	}

	events := make(Events, len(res.Items))

	for i, item := range res.Items {
		events[i] = *eventFromItem(item)
	}

	return events, nil
}

func (e *Event) Save() error {
	req := &dynamodb.PutItemInput{
		Item: map[string]dynamodb.AttributeValue{
			"app":      dynamodb.AttributeValue{S: aws.String(e.App)},
			"created":  dynamodb.AttributeValue{S: aws.String(e.Created.Format(SortableTime))},
			"metadata": dynamodb.AttributeValue{S: aws.String(e.Metadata)},
			"state":    dynamodb.AttributeValue{S: aws.String(e.State)},
			"type":     dynamodb.AttributeValue{S: aws.String(e.Type)},
			"user":     dynamodb.AttributeValue{S: aws.String(e.User)},
		},
		TableName: aws.String(eventsTable(e.App)),
	}

	_, err := DynamoDB.PutItem(req)

	return err
}

func eventsTable(app string) string {
	return fmt.Sprintf("%s-events", app)
}

func eventFromItem(item map[string]dynamodb.AttributeValue) *Event {
	created, _ := time.Parse(SortableTime, coalesce(item["created"].S, ""))

	return &Event{
		App:      coalesce(item["app"].S, ""),
		Created:  created,
		Metadata: coalesce(item["metadata"].S, ""),
		State:    coalesce(item["state"].S, ""),
		Type:     coalesce(item["type"].S, ""),
		User:     coalesce(item["user"].S, ""),
	}
}
