package models

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/convox/rack/api/crypt"
)

type Release struct {
	Id       string    `json:"id"`
	App      string    `json:"app"`
	Build    string    `json:"build"`
	Env      string    `json:"env"`
	Manifest string    `json:"manifest"`
	Created  time.Time `json:"created"`
}

type Releases []Release

func NewRelease(app string) Release {
	return Release{
		Id:  generateId("R", 10),
		App: app,
	}
}

func ListReleases(app string) (Releases, error) {
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

	res, err := DynamoDB().Query(req)

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
		ConsistentRead: aws.Bool(true),
		Key: map[string]*dynamodb.AttributeValue{
			"id": &dynamodb.AttributeValue{S: aws.String(id)},
		},
		TableName: aws.String(releasesTable(app)),
	}

	res, err := DynamoDB().GetItem(req)

	if err != nil {
		return nil, err
	}

	if res.Item == nil {
		return nil, fmt.Errorf("no such release: %s", id)
	}

	release := releaseFromItem(res.Item)

	return release, nil
}

func (r *Release) Cleanup() error {
	app, err := GetApp(r.App)

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

	NotifySuccess("release:create", map[string]string{"id": r.Id, "app": r.App})

	return S3Put(app.Outputs["Settings"], fmt.Sprintf("releases/%s/env", r.Id), env, true)
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

	// If release formation was saved in S3, get that instead
	f, err := s3Get(app.Outputs["Settings"], fmt.Sprintf("templates/%s", r.Id))

	if err != nil && awserrCode(err) != "NoSuchKey" {
		return err
	}

	if err == nil {
		formation = string(f)
	}

	fmt.Printf("ns=kernel at=release.promote at=s3Get found=%t\n", err == nil)

	existing, err := formationParameters(formation)

	if err != nil {
		return err
	}

	app.Parameters["Environment"] = r.EnvironmentUrl()
	app.Parameters["Kernel"] = CustomTopic
	app.Parameters["Release"] = r.Id
	app.Parameters["Version"] = os.Getenv("RELEASE")

	if os.Getenv("ENCRYPTION_KEY") != "" {
		app.Parameters["Key"] = os.Getenv("ENCRYPTION_KEY")
	}

	params := []*cloudformation.Parameter{}

	for key, value := range app.Parameters {
		if _, ok := existing[key]; ok {
			params = append(params, &cloudformation.Parameter{ParameterKey: aws.String(key), ParameterValue: aws.String(value)})
		}
	}

	err = S3Put(app.Outputs["Settings"], fmt.Sprintf("templates/%s", r.Id), []byte(formation), false)

	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://s3.amazonaws.com/%s/templates/%s", app.Outputs["Settings"], r.Id)

	req := &cloudformation.UpdateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(r.App),
		TemplateURL:  aws.String(url),
		Parameters:   params,
	}

	_, err = CloudFormation().UpdateStack(req)

	NotifySuccess("release:promote", map[string]string{
		"app": r.App,
		"id":  r.Id,
	})

	return err
}

func (r *Release) EnvironmentUrl() string {
	app, err := GetApp(r.App)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return ""
	}

	return fmt.Sprintf("https://%s.s3.amazonaws.com/releases/%s/env", app.Outputs["Settings"], r.Id)
}

func (r *Release) Formation() (string, error) {
	manifest, err := LoadManifest(r.Manifest)

	if err != nil {
		return "", err
	}

	// try to figure out which process to map to the main load balancer
	primary, err := primaryProcess(r.App)

	if err != nil {
		return "", err
	}

	// if we dont have a primary default to a process named web
	if primary == "" && manifest.Entry("web") != nil {
		primary = "web"
	}

	// if we still dont have a primary try the first process with external ports
	if primary == "" && manifest.HasExternalPorts() {
		for _, entry := range manifest {
			if len(entry.ExternalPorts()) > 0 {
				primary = entry.Name
				break
			}
		}
	}

	for i, entry := range manifest {
		if entry.Name == primary {
			manifest[i].primary = true
		}

		manifest[i].Image = fmt.Sprintf("%s/%s-%s:%s", os.Getenv("REGISTRY_HOST"), r.App, entry.Name, r.Build)
	}

	return manifest.Formation()
}

var regexpPrimaryProcess = regexp.MustCompile(`\[":",\["TCP",\{"Ref":"([A-Za-z]+)Port\d+Host`)

// try to determine which process to map to the main load balancer
func primaryProcess(app string) (string, error) {
	res, err := CloudFormation().GetTemplate(&cloudformation.GetTemplateInput{
		StackName: aws.String(app),
	})

	if err != nil {
		return "", err
	}

	/* bounce through json marshaling to make whitespace predictable */

	var body interface{}

	err = json.Unmarshal([]byte(*res.TemplateBody), &body)

	if err != nil {
		return "", err
	}

	data, err := json.Marshal(body)

	process := regexpPrimaryProcess.FindStringSubmatch(string(data))

	if len(process) > 1 {
		return DashName(process[1]), nil
	}

	return "", nil
}

func releasesTable(app string) string {
	return os.Getenv("DYNAMO_RELEASES")
}

func releaseFromItem(item map[string]*dynamodb.AttributeValue) *Release {
	created, _ := time.Parse(SortableTime, coalesce(item["created"], ""))

	release := &Release{
		Id:       coalesce(item["id"], ""),
		App:      coalesce(item["app"], ""),
		Build:    coalesce(item["build"], ""),
		Env:      coalesce(item["env"], ""),
		Manifest: coalesce(item["manifest"], ""),
		Created:  created,
	}

	return release
}
