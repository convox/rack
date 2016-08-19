package aws

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/convox/rack/api/crypt"
	"github.com/convox/rack/api/structs"
)

func releasesTable(app string) string {
	return os.Getenv("DYNAMO_RELEASES")
}

// ReleaseGet returns a release
func (p *AWSProvider) ReleaseGet(app, id string) (*structs.Release, error) {
	if id == "" {
		return nil, fmt.Errorf("release id must not be empty")
	}

	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	req := &dynamodb.GetItemInput{
		ConsistentRead: aws.Bool(true),
		Key: map[string]*dynamodb.AttributeValue{
			"id": &dynamodb.AttributeValue{S: aws.String(id)},
		},
		TableName: aws.String(releasesTable(a.Name)),
	}

	res, err := p.dynamodb().GetItem(req)
	if err != nil {
		return nil, err
	}
	if res.Item == nil {
		return nil, ErrorNotFound(fmt.Sprintf("no such release: %s", id))
	}

	release := releaseFromItem(res.Item)

	return release, nil
}

// ReleaseList returns a list of the latest releases, with the length specified in limit
func (p *AWSProvider) ReleaseList(app string, limit int64) (structs.Releases, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	req := &dynamodb.QueryInput{
		KeyConditions: map[string]*dynamodb.Condition{
			"app": &dynamodb.Condition{
				AttributeValueList: []*dynamodb.AttributeValue{
					&dynamodb.AttributeValue{S: aws.String(a.Name)},
				},
				ComparisonOperator: aws.String("EQ"),
			},
		},
		IndexName:        aws.String("app.created"),
		Limit:            aws.Int64(limit),
		ScanIndexForward: aws.Bool(false),
		TableName:        aws.String(releasesTable(a.Name)),
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

func (p *AWSProvider) ReleasePromote(app, id string) (*structs.Release, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	if !a.IsBound() {
		return nil, fmt.Errorf("unbound apps are no longer supported for promotion")
	}

	return &structs.Release{}, fmt.Errorf("promote not yet implemented for AWS provider")
}

func (p *AWSProvider) ReleaseSave(r *structs.Release, bucket, key string) error {
	if r.Id == "" {
		return fmt.Errorf("Id can not be blank")
	}

	if r.Created.IsZero() {
		r.Created = time.Now()
	}

	req := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"id":      &dynamodb.AttributeValue{S: aws.String(r.Id)},
			"app":     &dynamodb.AttributeValue{S: aws.String(r.App)},
			"created": &dynamodb.AttributeValue{S: aws.String(r.Created.Format(sortableTime))},
		},
		TableName: aws.String(releasesTable(r.App)),
	}

	if r.Build != "" {
		req.Item["build"] = &dynamodb.AttributeValue{S: aws.String(r.Build)}
	}

	if r.Env != "" {
		req.Item["env"] = &dynamodb.AttributeValue{S: aws.String(r.Env)}
	}

	if r.Manifest != "" {
		req.Item["manifest"] = &dynamodb.AttributeValue{S: aws.String(r.Manifest)}
	}

	var err error
	env := []byte(r.Env)

	if key != "" {
		cr := crypt.New(os.Getenv("AWS_REGION"), os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"))

		env, err = cr.Encrypt(key, []byte(env))
		if err != nil {
			return err
		}
	}

	_, err = p.s3().PutObject(&s3.PutObjectInput{
		ACL:           aws.String("public-read"),
		Body:          bytes.NewReader(env),
		Bucket:        aws.String(bucket),
		ContentLength: aws.Int64(int64(len(env))),
		Key:           aws.String(fmt.Sprintf("releases/%s/env", r.Id)),
	})
	if err != nil {
		return err
	}

	_, err = p.dynamodb().PutItem(req)
	return err
}

func releaseFromItem(item map[string]*dynamodb.AttributeValue) *structs.Release {
	created, _ := time.Parse(sortableTime, coalesce(item["created"], ""))

	release := &structs.Release{
		Id:       coalesce(item["id"], ""),
		App:      coalesce(item["app"], ""),
		Build:    coalesce(item["build"], ""),
		Env:      coalesce(item["env"], ""),
		Manifest: coalesce(item["manifest"], ""),
		Created:  created,
	}

	return release
}

// ReleaseDelete will delete all releases that belong to app and buildID
// This could includes the active release which implies this should be called with caution.
func (p *AWSProvider) ReleaseDelete(app, buildID string) error {

	// query dynamo for all releases for this build
	qi := &dynamodb.QueryInput{
		KeyConditionExpression: aws.String("app = :app"),
		FilterExpression:       aws.String("build = :build"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":app":   &dynamodb.AttributeValue{S: aws.String(app)},
			":build": &dynamodb.AttributeValue{S: aws.String(buildID)},
		},
		IndexName: aws.String("app.created"),
		TableName: aws.String(releasesTable(app)),
	}

	return p.deleteReleaseItems(qi, releasesTable(app))
}

// releasesDeleteAll will delete all releases associate with app
// This includes the active release which implies this should only be called when deleting an app.
func (p *AWSProvider) releaseDeleteAll(app string) error {

	// query dynamo for all releases for this app
	qi := &dynamodb.QueryInput{
		KeyConditionExpression: aws.String("app = :app"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":app": &dynamodb.AttributeValue{S: aws.String(app)},
		},
		IndexName: aws.String("app.created"),
		TableName: aws.String(releasesTable(app)),
	}

	return p.deleteReleaseItems(qi, releasesTable(app))
}

// deleteReleaseItems deletes release items from Dynamodb based on query input and the tableName
func (p *AWSProvider) deleteReleaseItems(qi *dynamodb.QueryInput, tableName string) error {

	res, err := p.dynamodb().Query(qi)
	if err != nil {
		return err
	}

	// collect release IDs to delete
	wrs := []*dynamodb.WriteRequest{}
	for _, item := range res.Items {
		r := releaseFromItem(item)

		wr := &dynamodb.WriteRequest{
			DeleteRequest: &dynamodb.DeleteRequest{
				Key: map[string]*dynamodb.AttributeValue{
					"id": &dynamodb.AttributeValue{
						S: aws.String(r.Id),
					},
				},
			},
		}

		wrs = append(wrs, wr)
	}

	return p.dynamoBatchDeleteItems(wrs, tableName)
}
