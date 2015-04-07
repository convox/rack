package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/cloudformation"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/dynamodb"
)

type Release struct {
	Id string

	App string

	Active   bool
	Ami      string
	Manifest string

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

	a, err := GetApp(app)

	if err != nil {
		return nil, err
	}

	res, err := DynamoDB.Query(req)

	if err != nil {
		return nil, err
	}

	releases := make(Releases, len(res.Items))

	for i, item := range res.Items {
		releases[i] = *releaseFromItem(item)
		releases[i].Active = (a.Release == releases[i].Id)
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

	a, err := GetApp(app)

	if err != nil {
		return nil, err
	}

	res, err := DynamoDB.GetItem(req)

	if err != nil {
		return nil, err
	}

	release := releaseFromItem(res.Item)
	release.Active = (a.Release == release.Id)

	return release, nil
}

func (r *Release) Save() error {
	if r.Id == "" {
		r.Id = generateId("R", 10)
	}

	if r.Created.IsZero() {
		r.Created = time.Now()
	}

	req := &dynamodb.PutItemInput{
		Item: map[string]dynamodb.AttributeValue{
			"id":      dynamodb.AttributeValue{S: aws.String(r.Id)},
			"app":     dynamodb.AttributeValue{S: aws.String(r.App)},
			"created": dynamodb.AttributeValue{S: aws.String(r.Created.Format(SortableTime))},
		},
		TableName: aws.String(releasesTable(r.App)),
	}

	if r.Ami != "" {
		req.Item["ami"] = dynamodb.AttributeValue{S: aws.String(r.Ami)}
	}

	if r.Manifest != "" {
		req.Item["manifest"] = dynamodb.AttributeValue{S: aws.String(r.Manifest)}
	}

	_, err := DynamoDB.PutItem(req)

	return err
}

func (r *Release) Formation() (string, error) {
	app, err := GetApp(r.App)

	if err != nil {
		return "", err
	}

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

	azs, err := ListAvailabilityZones()

	if err != nil {
		return err
	}

	formation, err := r.Formation()

	if err != nil {
		return err
	}

	// TODO: remove hardcoded Environment
	req := &cloudformation.UpdateStackInput{
		StackName:    aws.String(r.App),
		TemplateBody: aws.String(formation),
		Capabilities: []string{"CAPABILITY_IAM"},
		Parameters: []cloudformation.Parameter{
			cloudformation.Parameter{ParameterKey: aws.String("AMI"), ParameterValue: aws.String(r.Ami)},
			cloudformation.Parameter{ParameterKey: aws.String("AvailabilityZones"), ParameterValue: aws.String(strings.Join(azs.Names(), ","))},
			cloudformation.Parameter{ParameterKey: aws.String("Environment"), ParameterValue: aws.String("http://convox-temp-ui8ae2rie8ie.s3.amazonaws.com/env")},
			cloudformation.Parameter{ParameterKey: aws.String("Release"), ParameterValue: aws.String(r.Id)},
			cloudformation.Parameter{ParameterKey: aws.String("Repository"), ParameterValue: aws.String(app.Repository)},
		},
	}

	manifest, err := LoadManifest(r.Manifest)

	if err != nil {
		return err
	}

	for _, process := range manifest {
		if len(process.Ports) > 0 {
			req.Parameters = append(req.Parameters, cloudformation.Parameter{
				ParameterKey:   aws.String(fmt.Sprintf("%sPorts", upperName(process.Name))),
				ParameterValue: aws.String(strings.Join(process.Ports, ",")),
			})
		}
	}

	_, err = CloudFormation.UpdateStack(req)

	return err
}

func releasesTable(app string) string {
	return fmt.Sprintf("%s-releases", app)
}

func releaseFromItem(item map[string]dynamodb.AttributeValue) *Release {
	created, _ := time.Parse(SortableTime, coalesce(item["created"].S, ""))

	return &Release{
		Id:       coalesce(item["id"].S, ""),
		Ami:      coalesce(item["ami"].S, ""),
		Manifest: coalesce(item["manifest"].S, ""),
		App:      coalesce(item["app"].S, ""),
		Created:  created,
	}
}
