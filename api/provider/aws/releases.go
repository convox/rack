package aws

import (
	"fmt"
	"os"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/convox/rack/api/crypt"
	"github.com/convox/rack/api/structs"
)

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

func (p *AWSProvider) ReleaseSave(r *structs.Release) error {
	if r.Id == "" {
		return fmt.Errorf("Id must not be blank")
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

	_, err := p.dynamodb().PutItem(req)

	if err != nil {
		return err
	}

	app, err := p.AppGet(r.App)

	if err != nil {
		return err
	}

	env := []byte(r.Env)

	if app.Parameters["Key"] != "" {
		cr := crypt.New(os.Getenv("AWS_REGION"), os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"))

		env, err = cr.Encrypt(app.Parameters["Key"], []byte(env))

		if err != nil {
			return err
		}
	}

	p.NotifySuccess("release:create", map[string]string{
		"id":  r.Id,
		"app": r.App,
	})

	return p.s3Put(app.Outputs["Settings"], fmt.Sprintf("releases/%s/env", r.Id), env, true)
}

func (p *AWSProvider) ReleaseFork(app string) (*structs.Release, error) {
	release, err := p.appLatestRelease(app)

	if err != nil {
		return nil, err
	}

	if release == nil {
		release = &structs.Release{
			App: app,
		}
	}

	release.Id = generateId("R", 10)
	release.Created = time.Time{}

	return release, nil
}

func (p *AWSProvider) releaseCleanup(release structs.Release) error {
	app, err := p.AppGet(release.App)

	if err != nil {
		return err
	}

	// delete env
	err = p.s3Delete(app.Outputs["Settings"], fmt.Sprintf("releases/%s/env", release.Id))

	if err != nil {
		return err
	}

	return nil
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

func releasesTable(app string) string {
	return os.Getenv("DYNAMO_RELEASES")
}
