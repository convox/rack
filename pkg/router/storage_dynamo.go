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
	ddb   *dynamodb.DynamoDB
	table string
}

func NewStorageDynamo(table string) *StorageDynamo {
	fmt.Printf("ns=storage.dynamo at=new table=%s\n", table)

	return &StorageDynamo{
		ddb:   dynamodb.New(session.New()),
		table: table,
	}
}

func (b *StorageDynamo) IdleGet(host string) (bool, error) {
	fmt.Printf("ns=storage.dynamo at=idle.get host=%q\n", host)

	res, err := b.ddb.GetItem(&dynamodb.GetItemInput{
		Key:       map[string]*dynamodb.AttributeValue{"host": {S: aws.String(host)}},
		TableName: aws.String(b.table),
	})
	if err != nil {
		return false, err
	}
	if res.Item == nil || res.Item["idle"] == nil || res.Item["idle"].S == nil {
		return false, nil
	}

	return (*res.Item["idle"].S == "true"), nil
}

func (b *StorageDynamo) IdleSet(host string, idle bool) error {
	fmt.Printf("ns=storage.dynamo at=idle.get host=%q idle=%t\n", host, idle)

	_, err := b.ddb.UpdateItem(&dynamodb.UpdateItemInput{
		ExpressionAttributeNames:  map[string]*string{"#idle": aws.String("idle")},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":idle": {S: aws.String(fmt.Sprintf("%t", idle))}},
		Key:                       map[string]*dynamodb.AttributeValue{"host": &dynamodb.AttributeValue{S: aws.String(host)}},
		TableName:                 aws.String(b.table),
		UpdateExpression:          aws.String("SET #idle = :idle"),
	})
	if err != nil {
		return err
	}
	return nil
}

func (b *StorageDynamo) RequestBegin(host string) error {
	fmt.Printf("ns=storage.dynamo at=request.begin host=%q\n", host)

	activity := time.Now().UTC().Format(helpers.SortableTime)

	_, err := b.ddb.UpdateItem(&dynamodb.UpdateItemInput{
		ExpressionAttributeNames:  map[string]*string{"#activity": aws.String("activity"), "#active": aws.String("active")},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":activity": {S: aws.String(activity)}, ":n": {N: aws.String("1")}},
		Key:                       map[string]*dynamodb.AttributeValue{"host": &dynamodb.AttributeValue{S: aws.String(host)}},
		TableName:                 aws.String(b.table),
		UpdateExpression:          aws.String("SET #activity = :activity ADD #active :n"),
	})
	if err != nil {
		return err
	}

	return nil
}

func (b *StorageDynamo) RequestEnd(host string) error {
	fmt.Printf("ns=storage.dynamo at=request.end host=%q\n", host)

	activity := time.Now().UTC().Format(helpers.SortableTime)

	_, err := b.ddb.UpdateItem(&dynamodb.UpdateItemInput{
		ExpressionAttributeNames:  map[string]*string{"#activity": aws.String("activity"), "#active": aws.String("active")},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":activity": {S: aws.String(activity)}, ":n": {N: aws.String("-1")}},
		Key:                       map[string]*dynamodb.AttributeValue{"host": &dynamodb.AttributeValue{S: aws.String(host)}},
		TableName:                 aws.String(b.table),
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

func (b *StorageDynamo) TargetAdd(host, target string) error {
	fmt.Printf("ns=storage.dynamo at=target.add host=%q target=%q\n", host, target)

	_, err := b.ddb.UpdateItem(&dynamodb.UpdateItemInput{
		ExpressionAttributeNames:  map[string]*string{"#targets": aws.String("targets")},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":targets": {SS: []*string{aws.String(target)}}},
		Key:                       map[string]*dynamodb.AttributeValue{"host": {S: aws.String(host)}},
		TableName:                 aws.String(b.table),
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
		TableName: aws.String(b.table),
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
		TableName:                 aws.String(b.table),
		UpdateExpression:          aws.String("DELETE #targets :targets"),
	})
	if err != nil {
		return err
	}

	return nil
}
