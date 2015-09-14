package models

import (
	"encoding/json"
	"os"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/dynamodb"
)

type Change struct {
	App     string
	Created time.Time

	Metadata string
	Logs     string
	Status   string
	TargetId string
	Type     string
	User     string
	M        ChangeMetadata
}

type Changes []Change

type ChangeMetadata struct {
	Events       []Event       `json:"events"`
	Transactions []Transaction `json:"transactions"`
	Error        string        `json:"error"`
}

func ListChanges(app string) (Changes, error) {
	req := &dynamodb.QueryInput{
		KeyConditions: &map[string]*dynamodb.Condition{
			"app": &dynamodb.Condition{
				AttributeValueList: []*dynamodb.AttributeValue{
					&dynamodb.AttributeValue{S: aws.String(app)},
				},
				ComparisonOperator: aws.String("EQ"),
			},
		},
		Limit:            aws.Long(10),
		ScanIndexForward: aws.Boolean(false),
		TableName:        aws.String(changesTable(app)),
	}

	res, err := DynamoDB().Query(req)

	if err != nil {
		return nil, err
	}

	changes := make(Changes, len(res.Items))

	for i, item := range res.Items {
		changes[i] = *changeFromItem(*item)
	}

	return changes, nil
}

func (e *Change) Save() error {
	req := &dynamodb.PutItemInput{
		Item: &map[string]*dynamodb.AttributeValue{
			"app":       &dynamodb.AttributeValue{S: aws.String(e.App)},
			"created":   &dynamodb.AttributeValue{S: aws.String(e.Created.Format(SortableTime))},
			"metadata":  &dynamodb.AttributeValue{S: aws.String(e.Metadata)},
			"status":    &dynamodb.AttributeValue{S: aws.String(e.Status)},
			"target_id": &dynamodb.AttributeValue{S: aws.String(e.TargetId)},
			"type":      &dynamodb.AttributeValue{S: aws.String(e.Type)},
			"user":      &dynamodb.AttributeValue{S: aws.String(e.User)},
		},
		TableName: aws.String(changesTable(e.App)),
	}

	_, err := DynamoDB().PutItem(req)

	if err != nil {
		panic(err)
	}

	return err
}

func changesTable(app string) string {
	return os.Getenv("DYNAMO_CHANGES")
}

func changeFromItem(item map[string]*dynamodb.AttributeValue) *Change {
	created, _ := time.Parse(SortableTime, coalesce(item["created"], ""))

	metadata := ChangeMetadata{}

	err := json.Unmarshal([]byte(coalesce(item["metadata"], "{}")), &metadata)
	if err != nil {
		panic(err)
	}

	return &Change{
		App:      coalesce(item["app"], ""),
		Created:  created,
		Metadata: coalesce(item["metadata"], ""),
		M:        metadata,
		Status:   coalesce(item["status"], ""),
		Type:     coalesce(item["type"], ""),
		TargetId: coalesce(item["target_id"], ""),
		User:     coalesce(item["user"], ""),
	}
}
