package aws

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/s3"
	"github.com/convox/rack/api/crypt"
	"github.com/convox/rack/api/structs"
)

func releasesTable(app string) string {
	return os.Getenv("DYNAMO_RELEASES")
}

func (p *AWSProvider) ReleaseGet(app, id string) (*structs.Release, error) {
	req := &dynamodb.GetItemInput{
		ConsistentRead: aws.Bool(true),
		Key: map[string]*dynamodb.AttributeValue{
			"id": &dynamodb.AttributeValue{S: aws.String(id)},
		},
		TableName: aws.String(releasesTable(app)),
	}

	res, err := p.dynamodb().GetItem(req)

	if err != nil {
		return nil, err
	}

	if res.Item == nil {
		return nil, fmt.Errorf("no such release: %s", id)
	}

	release := releaseFromItem(res.Item)

	return release, nil
}

func (p *AWSProvider) ReleaseList(app string) (structs.Releases, error) {
	req := &dynamodb.QueryInput{
		KeyConditions: map[string]*dynamodb.Condition{
			"app": &dynamodb.Condition{
				AttributeValueList: []*dynamodb.AttributeValue{
					&dynamodb.AttributeValue{S: aws.String(app)},
				},
				ComparisonOperator: aws.String("EQ"),
			},
		},
		IndexName:        aws.String("app.created"),
		Limit:            aws.Int64(20),
		ScanIndexForward: aws.Bool(false),
		TableName:        aws.String(releasesTable(app)),
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
			"created": &dynamodb.AttributeValue{S: aws.String(r.Created.Format(SortableTime))},
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
	created, _ := time.Parse(SortableTime, coalesce(item["created"], ""))

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
