package router

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"golang.org/x/crypto/acme/autocert"
)

type CacheDynamo struct {
	ddb   *dynamodb.DynamoDB
	table string
}

func NewCacheDynamo(table string) *CacheDynamo {
	fmt.Printf("ns=cache.dynamo at=new table=%s\n", table)

	return &CacheDynamo{
		ddb:   dynamodb.New(session.New()),
		table: table,
	}
}

func (c *CacheDynamo) Delete(ctx context.Context, key string) error {
	fmt.Printf("ns=cache.dynamo at=delete key=%s\n", key)

	_, err := c.ddb.DeleteItem(&dynamodb.DeleteItemInput{
		Key:       map[string]*dynamodb.AttributeValue{"key": {S: aws.String(key)}},
		TableName: aws.String(c.table),
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *CacheDynamo) Get(ctx context.Context, key string) ([]byte, error) {
	fmt.Printf("ns=cache.dynamo at=get key=%s\n", key)

	res, err := c.ddb.GetItem(&dynamodb.GetItemInput{
		Key:       map[string]*dynamodb.AttributeValue{"key": {S: aws.String(key)}},
		TableName: aws.String(c.table),
	})
	if err != nil {
		return nil, err
	}
	if res.Item == nil || res.Item["value"] == nil || res.Item["value"].B == nil {
		return nil, autocert.ErrCacheMiss
	}

	return res.Item["value"].B, nil
}

func (c *CacheDynamo) Put(ctx context.Context, key string, data []byte) error {
	fmt.Printf("ns=cache.dynamo at=put key=%s\n", key)

	_, err := c.ddb.PutItem(&dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"key":   {S: aws.String(key)},
			"value": {B: data},
		},
		TableName: aws.String(c.table),
	})
	if err != nil {
		return err
	}

	return nil
}
