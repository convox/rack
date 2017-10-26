package aws

import (
	"bytes"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/convox/rack/api/crypt"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/manifest"
)

// ReleaseDelete will delete all releases that belong to app and buildID
// This could includes the active release which implies this should be called with caution.
func (p *AWSProvider) ReleaseDelete(app, buildID string) error {
	qi := &dynamodb.QueryInput{
		KeyConditionExpression: aws.String("app = :app"),
		FilterExpression:       aws.String("build = :build"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":app":   {S: aws.String(app)},
			":build": {S: aws.String(buildID)},
		},
		IndexName: aws.String("app.created"),
		TableName: aws.String(p.DynamoReleases),
	}

	return p.deleteReleaseItems(qi, p.DynamoReleases)
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

	item, err := p.fetchRelease(app, id)
	if err != nil {
		return nil, err
	}

	r, err := releaseFromItem(item)
	if err != nil {
		return nil, err
	}

	settings, err := p.appResource(app, "Settings")
	if err != nil {
		return nil, err
	}

	data, err := p.s3Get(settings, fmt.Sprintf("releases/%s/env", r.Id))
	if err != nil {
		return nil, err
	}

	if a.Parameters["Key"] != "" {
		if d, err := crypt.New().Decrypt(a.Parameters["Key"], data); err == nil {
			data = d
		}
	}

	env := structs.Environment{}
	env.LoadEnvironment(data)

	r.Env = env.Raw()

	return r, nil
}

// ReleaseList returns a list of the latest releases, with the length specified in limit
func (p *AWSProvider) ReleaseList(app string, limit int64) (structs.Releases, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	req := &dynamodb.QueryInput{
		KeyConditions: map[string]*dynamodb.Condition{
			"app": {
				AttributeValueList: []*dynamodb.AttributeValue{
					{S: aws.String(a.Name)},
				},
				ComparisonOperator: aws.String("EQ"),
			},
		},
		IndexName:        aws.String("app.created"),
		Limit:            aws.Int64(limit),
		ScanIndexForward: aws.Bool(false),
		TableName:        aws.String(p.DynamoReleases),
	}

	res, err := p.dynamodb().Query(req)
	if err != nil {
		return nil, err
	}

	releases := make(structs.Releases, len(res.Items))

	for i, item := range res.Items {
		r, err := releaseFromItem(item)
		if err != nil {
			return nil, err
		}

		releases[i] = *r
	}

	return releases, nil
}

// ReleasePromote promotes a release
func (p *AWSProvider) ReleasePromote(r *structs.Release) error {
	if _, err := p.AppGet(r.App); err != nil {
		return err
	}

	stack := fmt.Sprintf("%s-%s", p.Rack, r.App)

	fmt.Printf("r = %+v\n", r)

	m, err := manifest.Load([]byte(r.Manifest), manifest.Environment{})
	if err != nil {
		return err
	}

	tp := map[string]interface{}{
		"App":      r.App,
		"Manifest": m,
		"Release":  r,
	}

	data, err := formationTemplate("app", tp)
	if err != nil {
		return err
	}

	fmt.Printf("string(data) = %+v\n", string(data))

	ou, err := p.ObjectStore("", bytes.NewReader(data), structs.ObjectOptions{})
	if err != nil {
		return err
	}

	fmt.Printf("ou = %+v\n", ou)

	updates := map[string]string{}

	fmt.Printf("updates = %+v\n", updates)

	if err := p.updateStack(stack, ou, updates); err != nil {
		return err
	}

	return nil
}

// ReleaseSave saves a Release
func (p *AWSProvider) ReleaseSave(r *structs.Release) error {
	a, err := p.AppGet(r.App)
	if err != nil {
		return err
	}

	if r.Id == "" {
		return fmt.Errorf("Id can not be blank")
	}

	if r.Created.IsZero() {
		r.Created = time.Now()
	}

	if p.IsTest() {
		r.Created = time.Unix(1473028693, 0).UTC()
	}

	req := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"id":      {S: aws.String(r.Id)},
			"app":     {S: aws.String(r.App)},
			"created": {S: aws.String(r.Created.Format(sortableTime))},
		},
		TableName: aws.String(p.DynamoReleases),
	}

	if r.Build != "" {
		req.Item["build"] = &dynamodb.AttributeValue{S: aws.String(r.Build)}
	}

	if r.Manifest != "" {
		req.Item["manifest"] = &dynamodb.AttributeValue{S: aws.String(r.Manifest)}
	}

	env := []byte(r.Env)
	key := a.Parameters["Key"]
	if key != "" {
		env, err = crypt.New().Encrypt(key, []byte(env))
		if err != nil {
			return err
		}
	}

	settings, err := p.appResource(r.App, "Settings")
	if err != nil {
		return err
	}

	_, err = p.s3().PutObject(&s3.PutObjectInput{
		ACL:           aws.String("public-read"),
		Body:          bytes.NewReader(env),
		Bucket:        aws.String(settings),
		ContentLength: aws.Int64(int64(len(env))),
		Key:           aws.String(fmt.Sprintf("releases/%s/env", r.Id)),
	})
	if err != nil {
		return err
	}

	_, err = p.dynamodb().PutItem(req)
	return err
}

func (p *AWSProvider) fetchRelease(app, id string) (map[string]*dynamodb.AttributeValue, error) {
	res, err := p.dynamodb().GetItem(&dynamodb.GetItemInput{
		ConsistentRead: aws.Bool(true),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {S: aws.String(id)},
		},
		TableName: aws.String(p.DynamoReleases),
	})
	if err != nil {
		return nil, err
	}
	if res.Item == nil {
		return nil, errorNotFound(fmt.Sprintf("no such release: %s", id))
	}
	if res.Item["app"] == nil || *res.Item["app"].S != app {
		return nil, fmt.Errorf("mismatched app and release")
	}

	return res.Item, nil
}

func releaseFromItem(item map[string]*dynamodb.AttributeValue) (*structs.Release, error) {
	created, err := time.Parse(sortableTime, coalesce(item["created"], ""))
	if err != nil {
		return nil, err
	}

	release := &structs.Release{
		Id:       coalesce(item["id"], ""),
		App:      coalesce(item["app"], ""),
		Build:    coalesce(item["build"], ""),
		Manifest: coalesce(item["manifest"], ""),
		Created:  created,
	}

	return release, nil
}

// releasesDeleteAll will delete all releases associate with app
// This includes the active release which implies this should only be called when deleting an app.
func (p *AWSProvider) releaseDeleteAll(app string) error {

	// query dynamo for all releases for this app
	qi := &dynamodb.QueryInput{
		KeyConditionExpression: aws.String("app = :app"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":app": {S: aws.String(app)},
		},
		IndexName: aws.String("app.created"),
		TableName: aws.String(p.DynamoReleases),
	}

	return p.deleteReleaseItems(qi, p.DynamoReleases)
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
		r, err := releaseFromItem(item)
		if err != nil {
			return err
		}

		wr := &dynamodb.WriteRequest{
			DeleteRequest: &dynamodb.DeleteRequest{
				Key: map[string]*dynamodb.AttributeValue{
					"id": {
						S: aws.String(r.Id),
					},
				},
			},
		}

		wrs = append(wrs, wr)
	}

	return p.dynamoBatchDeleteItems(wrs, tableName)
}
