package models

import (
	"fmt"
	"time"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/cloudformation"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/dynamodb"
)

type Release struct {
	Id string

	Ami      string
	Manifest string
	Status   string

	App string

	Created time.Time
}

type Releases []Release

func ListReleases(app string) (Releases, error) {
	req := &dynamodb.QueryInput{
		KeyConditions: map[string]dynamodb.Condition{
			"app": dynamodb.Condition{
				AttributeValueList: []dynamodb.AttributeValue{
					dynamodb.AttributeValue{S: aws.String(app)},
				},
				ComparisonOperator: aws.String("EQ"),
			},
		},
		IndexName:        aws.String("app.created"),
		Limit:            aws.Integer(10),
		ScanIndexForward: aws.Boolean(false),
		TableName:        aws.String(releasesTable(app)),
	}

	res, err := DynamoDB.Query(req)

	if err != nil {
		return nil, err
	}

	releases := make(Releases, len(res.Items))

	for i, item := range res.Items {
		releases[i] = *releaseFromItem(item)
	}

	return releases, nil
}

func GetRelease(app, id string) (*Release, error) {
	req := &dynamodb.GetItemInput{
		ConsistentRead: aws.Boolean(true),
		Key: map[string]dynamodb.AttributeValue{
			"id": dynamodb.AttributeValue{S: aws.String(id)},
		},
		TableName: aws.String(releasesTable(app)),
	}

	res, err := DynamoDB.GetItem(req)

	if err != nil {
		return nil, err
	}

	return releaseFromItem(res.Item), nil
}

func (r *Release) Formation() (string, error) {
	app, err := GetApp(r.App)

	if err != nil {
		return "", err
	}

	app.Release = r.Id

	manifest, err := LoadManifest(r.Manifest)

	if err != nil {
		return "", err
	}

	err = manifest.Apply(app)

	if err != nil {
		return "", err
	}

	formation, err := app.Formation()

	if err != nil {
		return "", err
	}

	return formation, nil
}

func (r *Release) Promote() error {
	app, err := GetApp(r.App)

	if err != nil {
		return err
	}

	formation, err := r.Formation()

	if err != nil {
		return err
	}

	req := &cloudformation.UpdateStackInput{
		StackName:    aws.String(fmt.Sprintf("convox-%s", r.App)),
		TemplateBody: aws.String(formation),
		Capabilities: []string{"CAPABILITY_IAM"},
		Parameters: []cloudformation.Parameter{
			cloudformation.Parameter{ParameterKey: aws.String("Release"), ParameterValue: aws.String(r.Id)},
			cloudformation.Parameter{ParameterKey: aws.String("Repository"), ParameterValue: aws.String(app.Repository)},
		},
	}

	_, err = CloudFormation.UpdateStack(req)

	return err
}

func releasesTable(app string) string {
	return fmt.Sprintf("convox-%s-releases", app)
}

func releaseFromItem(item map[string]dynamodb.AttributeValue) *Release {
	created, _ := time.Parse(SortableTime, coalesce(item["created"].S, ""))

	return &Release{
		Id:       coalesce(item["id"].S, ""),
		Ami:      coalesce(item["ami"].S, ""),
		Manifest: coalesce(item["manifest"].S, ""),
		Status:   coalesce(item["status"].S, ""),
		App:      coalesce(item["app"].S, ""),
		Created:  created,
	}
}
