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
	ddb     *dynamodb.DynamoDB
	hosts   string
	targets string
}

func NewStorageDynamo(hosts, targets string) *StorageDynamo {
	fmt.Printf("ns=storage.dynamo at=new hosts=%s targets=%s\n", hosts, targets)

	return &StorageDynamo{
		ddb:     dynamodb.New(session.New()),
		hosts:   hosts,
		targets: targets,
	}
}

func (s *StorageDynamo) IdleGet(target string) (bool, error) {
	fmt.Printf("ns=storage.dynamo at=idle.get target=%q\n", target)

	res, err := s.ddb.GetItem(&dynamodb.GetItemInput{
		Key:       map[string]*dynamodb.AttributeValue{"target": {S: aws.String(target)}},
		TableName: aws.String(s.targets),
	})
	if err != nil {
		return false, err
	}
	if res.Item == nil || res.Item["idle"] == nil || res.Item["idle"].S == nil {
		return false, nil
	}

	return (*res.Item["idle"].S == "true"), nil
}

func (s *StorageDynamo) IdleSet(target string, idle bool) error {
	fmt.Printf("ns=storage.dynamo at=idle.get target=%q idle=%t\n", target, idle)

	_, err := s.ddb.UpdateItem(&dynamodb.UpdateItemInput{
		ExpressionAttributeNames:  map[string]*string{"#idle": aws.String("idle")},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":idle": {S: aws.String(fmt.Sprintf("%t", idle))}},
		Key:              map[string]*dynamodb.AttributeValue{"target": &dynamodb.AttributeValue{S: aws.String(target)}},
		TableName:        aws.String(s.targets),
		UpdateExpression: aws.String("SET #idle = :idle"),
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *StorageDynamo) RequestBegin(target string) error {
	fmt.Printf("ns=storage.dynamo at=request.begin target=%q\n", target)

	activity := time.Now().UTC().Format(helpers.SortableTime)

	_, err := s.ddb.UpdateItem(&dynamodb.UpdateItemInput{
		ExpressionAttributeNames:  map[string]*string{"#activity": aws.String("activity"), "#active": aws.String("active")},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":activity": {S: aws.String(activity)}, ":n": {N: aws.String("1")}},
		Key:              map[string]*dynamodb.AttributeValue{"target": &dynamodb.AttributeValue{S: aws.String(target)}},
		TableName:        aws.String(s.targets),
		UpdateExpression: aws.String("SET #activity = :activity ADD #active :n"),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *StorageDynamo) RequestEnd(target string) error {
	fmt.Printf("ns=storage.dynamo at=request.end target=%q\n", target)

	activity := time.Now().UTC().Format(helpers.SortableTime)

	_, err := s.ddb.UpdateItem(&dynamodb.UpdateItemInput{
		ExpressionAttributeNames:  map[string]*string{"#activity": aws.String("activity"), "#active": aws.String("active")},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":activity": {S: aws.String(activity)}, ":n": {N: aws.String("-1")}},
		Key:              map[string]*dynamodb.AttributeValue{"target": &dynamodb.AttributeValue{S: aws.String(target)}},
		TableName:        aws.String(s.targets),
		UpdateExpression: aws.String("SET #activity = :activity ADD #active :n"),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *StorageDynamo) Stale(cutoff time.Time) ([]string, error) {
	fmt.Printf("ns=storage.dynamo at=stale cutoff=%s\n", cutoff)

	return []string{}, nil
}

func (s *StorageDynamo) TargetAdd(host, target string, idles bool) error {
	fmt.Printf("ns=storage.dynamo at=target.add host=%q target=%q\n", host, target)

	_, err := s.ddb.UpdateItem(&dynamodb.UpdateItemInput{
		ExpressionAttributeNames:  map[string]*string{"#targets": aws.String("targets")},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":targets": {SS: []*string{aws.String(target)}}},
		Key:              map[string]*dynamodb.AttributeValue{"host": {S: aws.String(host)}},
		TableName:        aws.String(s.hosts),
		UpdateExpression: aws.String("ADD #targets :targets"),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *StorageDynamo) TargetList(host string) ([]string, error) {
	fmt.Printf("ns=storage.dynamo at=target.list\n")

	res, err := s.ddb.GetItem(&dynamodb.GetItemInput{
		Key:       map[string]*dynamodb.AttributeValue{"host": {S: aws.String(host)}},
		TableName: aws.String(s.hosts),
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

func (s *StorageDynamo) TargetRemove(host, target string) error {
	fmt.Printf("ns=storage.dynamo at=target.remove host=%q target=%q\n", host, target)

	_, err := s.ddb.UpdateItem(&dynamodb.UpdateItemInput{
		ExpressionAttributeNames:  map[string]*string{"#targets": aws.String("targets")},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":targets": {SS: []*string{aws.String(target)}}},
		Key:              map[string]*dynamodb.AttributeValue{"host": {S: aws.String(host)}},
		TableName:        aws.String(s.hosts),
		UpdateExpression: aws.String("DELETE #targets :targets"),
	})
	if err != nil {
		return err
	}

	return nil
}
