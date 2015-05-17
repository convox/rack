package models

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/cloudformation"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/dynamodb"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/convox/env/crypt"
)

type Release struct {
	Id string

	App string

	Active   bool
	Ami      string
	Env      string
	Manifest string

	Created time.Time
}

type Releases []Release

func ListReleases(app string) (Releases, error) {
	req := &dynamodb.QueryInput{
		KeyConditions: &map[string]*dynamodb.Condition{
			"app": &dynamodb.Condition{
				AttributeValueList: []*dynamodb.AttributeValue{
					&dynamodb.AttributeValue{S: aws.String(app)},
				},
				ComparisonOperator: aws.String("EQ"),
			},
		},
		IndexName:        aws.String("app.created"),
		Limit:            aws.Long(10),
		ScanIndexForward: aws.Boolean(false),
		TableName:        aws.String(releasesTable(app)),
	}

	a, err := GetApp(app)

	if err != nil {
		return nil, err
	}

	res, err := DynamoDB().Query(req)

	if err != nil {
		return nil, err
	}

	releases := make(Releases, len(res.Items))

	for i, item := range res.Items {
		releases[i] = *releaseFromItem(*item)
		releases[i].Active = (a.Release == releases[i].Id)
	}

	return releases, nil
}

func GetRelease(app, id string) (*Release, error) {
	req := &dynamodb.GetItemInput{
		ConsistentRead: aws.Boolean(true),
		Key: &map[string]*dynamodb.AttributeValue{
			"id": &dynamodb.AttributeValue{S: aws.String(id)},
		},
		TableName: aws.String(releasesTable(app)),
	}

	a, err := GetApp(app)

	if err != nil {
		return nil, err
	}

	res, err := DynamoDB().GetItem(req)

	if err != nil {
		return nil, err
	}

	release := releaseFromItem(*res.Item)
	release.Active = (a.Release == release.Id)

	return release, nil
}

func (r *Release) Cleanup() error {
	app, err := GetApp(r.App)

	if err != nil {
		return err
	}

	// delete ami
	req := &ec2.DeregisterImageInput{
		ImageID: aws.String(r.Ami),
	}

	_, err = EC2().DeregisterImage(req)

	if err != nil {
		return err
	}

	// delete env
	err = s3Delete(app.Outputs["Settings"], fmt.Sprintf("releases/%s/env", r.Id))

	if err != nil {
		return err
	}

	return nil
}

func (r *Release) Save() error {
	if r.Id == "" {
		r.Id = generateId("R", 10)
	}

	if r.Created.IsZero() {
		r.Created = time.Now()
	}

	req := &dynamodb.PutItemInput{
		Item: &map[string]*dynamodb.AttributeValue{
			"id":      &dynamodb.AttributeValue{S: aws.String(r.Id)},
			"app":     &dynamodb.AttributeValue{S: aws.String(r.App)},
			"created": &dynamodb.AttributeValue{S: aws.String(r.Created.Format(SortableTime))},
		},
		TableName: aws.String(releasesTable(r.App)),
	}

	if r.Ami != "" {
		(*req.Item)["ami"] = &dynamodb.AttributeValue{S: aws.String(r.Ami)}
	}

	if r.Env != "" {
		(*req.Item)["env"] = &dynamodb.AttributeValue{S: aws.String(r.Env)}
	}

	if r.Manifest != "" {
		(*req.Item)["manifest"] = &dynamodb.AttributeValue{S: aws.String(r.Manifest)}
	}

	_, err := DynamoDB().PutItem(req)

	if err != nil {
		return err
	}

	app, err := GetApp(r.App)

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

	return s3Put(app.Outputs["Settings"], fmt.Sprintf("releases/%s/env", r.Id), env, true)
}

func (r *Release) Promote() error {
	app, err := GetApp(r.App)

	if err != nil {
		return err
	}

	manifest, err := LoadManifest(r.Manifest)

	if err != nil {
		return err
	}

	formation, err := app.Formation()

	if err != nil {
		return err
	}

	params := app.Parameters

	params["AMI"] = r.Ami
	params["Environment"] = fmt.Sprintf("https://%s.s3.amazonaws.com/releases/%s/env", app.Outputs["Settings"], r.Id)
	params["Release"] = r.Id

	for _, p := range manifest.Processes() {
		params[fmt.Sprintf("%sCommand", upperName(p.Name))] = p.Command
	}

	stackParams := []*cloudformation.Parameter{}

	for key, value := range params {
		stackParams = append(stackParams, &cloudformation.Parameter{ParameterKey: aws.String(key), ParameterValue: aws.String(value)})
	}

	existing, err := formationParameters(formation)

	if err != nil {
		return err
	}

	finalParams := []*cloudformation.Parameter{}

	// remove any params that do not exist in the formation
	for _, sp := range stackParams {
		if _, ok := existing[*sp.ParameterKey]; ok {
			finalParams = append(finalParams, sp)
		}
	}

	req := &cloudformation.UpdateStackInput{
		StackName:    aws.String(r.App),
		TemplateBody: aws.String(formation),
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		Parameters:   finalParams,
	}

	manifest, err = LoadManifest(r.Manifest)

	if err != nil {
		return err
	}

	for _, process := range manifest {
		if len(process.Ports) > 0 {
			if pp := strings.Split(process.Ports[0], ":"); len(pp) == 2 {
				req.Parameters = append(req.Parameters, &cloudformation.Parameter{
					ParameterKey:   aws.String(fmt.Sprintf("%sPort", upperName(process.Name))),
					ParameterValue: aws.String(pp[1]),
				})
			}
		}
	}

	_, err = CloudFormation().UpdateStack(req)

	return err
}

func (r *Release) Services() (Services, error) {
	manifest, err := LoadManifest(r.Manifest)

	if err != nil {
		return nil, err
	}

	services := manifest.Services()

	for i := range services {
		services[i].App = r.App
	}

	return services, nil
}

func releasesTable(app string) string {
	return fmt.Sprintf("%s-releases", app)
}

func releaseFromItem(item map[string]*dynamodb.AttributeValue) *Release {
	created, _ := time.Parse(SortableTime, coalesce(item["created"], ""))

	return &Release{
		Id:       coalesce(item["id"], ""),
		App:      coalesce(item["app"], ""),
		Ami:      coalesce(item["ami"], ""),
		Env:      coalesce(item["env"], ""),
		Manifest: coalesce(item["manifest"], ""),
		Created:  created,
	}
}
