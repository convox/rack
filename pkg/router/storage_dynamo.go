package router

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/convox/rack/pkg/helpers"
)

type StorageDynamo struct {
	activity string
	ddb      *dynamodb.DynamoDB
	routes   string
}

func NewStorageDynamo(routes, activity string) *StorageDynamo {
	fmt.Printf("ns=storage.dynamo at=new routes=%s activity=%s\n", routes, activity)

	return &StorageDynamo{
		activity: activity,
		ddb:      dynamodb.New(session.New()),
		routes:   routes,
	}
}

func (b *StorageDynamo) IdleGet(target string) (bool, error) {
	fmt.Printf("ns=storage.dynamo at=idle.get target=%q\n", target)

	res, err := b.ddb.GetItem(&dynamodb.GetItemInput{
		Key:       map[string]*dynamodb.AttributeValue{"target": {S: aws.String(target)}},
		TableName: aws.String(b.activity),
	})
	if err != nil {
		return false, err
	}
	if res.Item == nil || res.Item["idle"] == nil || res.Item["idle"].S == nil {
		return false, nil
	}

	return (*res.Item["idle"].S == "true"), nil
}

func (b *StorageDynamo) IdleSet(target string, idle bool) error {
	fmt.Printf("ns=storage.dynamo at=idle.get target=%q idle=%t\n", target, idle)

	_, err := b.ddb.UpdateItem(&dynamodb.UpdateItemInput{
		ExpressionAttributeNames:  map[string]*string{"#idle": aws.String("idle")},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":idle": {S: aws.String(fmt.Sprintf("%t", idle))}},
		Key:                       map[string]*dynamodb.AttributeValue{"target": &dynamodb.AttributeValue{S: aws.String(target)}},
		TableName:                 aws.String(b.activity),
		UpdateExpression:          aws.String("SET #idle = :idle"),
	})
	if err != nil {
		return err
	}
	return nil
}

func (b *StorageDynamo) RequestBegin(target string) error {
	fmt.Printf("ns=storage.dynamo at=request.begin target=%q\n", target)

	activity := time.Now().UTC().Format(helpers.SortableTime)

	_, err := b.ddb.UpdateItem(&dynamodb.UpdateItemInput{
		ExpressionAttributeNames:  map[string]*string{"#activity": aws.String("activity"), "#active": aws.String("active")},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":activity": {S: aws.String(activity)}, ":n": {N: aws.String("1")}},
		Key:                       map[string]*dynamodb.AttributeValue{"target": &dynamodb.AttributeValue{S: aws.String(target)}},
		TableName:                 aws.String(b.activity),
		UpdateExpression:          aws.String("SET #activity = :activity ADD #active :n"),
	})
	if err != nil {
		return err
	}

	return nil
}

func (b *StorageDynamo) RequestEnd(target string) error {
	fmt.Printf("ns=storage.dynamo at=request.end target=%q\n", target)

	activity := time.Now().UTC().Format(helpers.SortableTime)

	_, err := b.ddb.UpdateItem(&dynamodb.UpdateItemInput{
		ExpressionAttributeNames:  map[string]*string{"#activity": aws.String("activity"), "#active": aws.String("active")},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":activity": {S: aws.String(activity)}, ":n": {N: aws.String("-1")}},
		Key:                       map[string]*dynamodb.AttributeValue{"target": &dynamodb.AttributeValue{S: aws.String(target)}},
		TableName:                 aws.String(b.activity),
		UpdateExpression:          aws.String("SET #activity = :activity ADD #active :n"),
	})
	if err != nil {
		return err
	}

	return nil
}

func (b *StorageDynamo) Stale(cutoff time.Time) ([]string, error) {
	fmt.Printf("ns=storage.dynamo at=stale cutoff=%s\n", cutoff)

	return []string{}, nil
}

func (b *StorageDynamo) TargetAdd(host, target string, idles bool) error {
	fmt.Printf("ns=storage.dynamo at=target.add host=%q target=%q\n", host, target)

	_, err := b.ddb.UpdateItem(&dynamodb.UpdateItemInput{
		ExpressionAttributeNames:  map[string]*string{"#targets": aws.String("targets")},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":targets": {SS: []*string{aws.String(target)}}},
		Key:                       map[string]*dynamodb.AttributeValue{"host": {S: aws.String(host)}},
		TableName:                 aws.String(b.routes),
		UpdateExpression:          aws.String("ADD #targets :targets"),
	})
	if err != nil {
		return err
	}

	return nil
}

func (b *StorageDynamo) TargetList(host string) ([]string, error) {
	fmt.Printf("ns=storage.dynamo at=target.list\n")

	res, err := b.ddb.GetItem(&dynamodb.GetItemInput{
		Key:       map[string]*dynamodb.AttributeValue{"host": {S: aws.String(host)}},
		TableName: aws.String(b.routes),
	})
	if err != nil {
		return nil, err
	}
	if res.Item == nil || res.Item["targets"] == nil {
		return []string{}, nil
	}

	ts := []string{}

	for _, t := range res.Item["targets"].SS {
		ts = append(ts, *t)
	}

	return ts, nil
}

func (b *StorageDynamo) TargetRemove(host, target string) error {
	fmt.Printf("ns=storage.dynamo at=target.remove host=%q target=%q\n", host, target)

	_, err := b.ddb.UpdateItem(&dynamodb.UpdateItemInput{
		ExpressionAttributeNames:  map[string]*string{"#targets": aws.String("targets")},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":targets": {SS: []*string{aws.String(target)}}},
		Key:                       map[string]*dynamodb.AttributeValue{"host": {S: aws.String(host)}},
		TableName:                 aws.String(b.routes),
		UpdateExpression:          aws.String("DELETE #targets :targets"),
	})
	if err != nil {
		return err
	}

	return nil
}
