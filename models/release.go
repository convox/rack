package models

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/cloudformation"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/dynamodb"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/convox/env/crypt"
)

type Release struct {
	App      string    `json:"app"`
	Build    string    `json:"build"`
	Env      string    `json:"env"`
	Id       string    `json:"id"`
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
		KeyConditions: &map[string]*dynamodb.Condition{
			"app": &dynamodb.Condition{
				AttributeValueList: []*dynamodb.AttributeValue{
					&dynamodb.AttributeValue{S: aws.String(app)},
				},
				ComparisonOperator: aws.String("EQ"),
			},
		},
		IndexName:        aws.String("app.created"),
		Limit:            aws.Long(20),
		ScanIndexForward: aws.Boolean(false),
		TableName:        aws.String(releasesTable(app)),
	}

	res, err := DynamoDB().Query(req)

	if err != nil {
		return nil, err
	}

	releases := make(Releases, len(res.Items))

	for i, item := range res.Items {
		releases[i] = *releaseFromItem(*item)
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

	res, err := DynamoDB().GetItem(req)

	if err != nil {
		return nil, err
	}

	if res.Item == nil {
		return nil, fmt.Errorf("no such release: %s", id)
	}

	release := releaseFromItem(*res.Item)

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
		Item: &map[string]*dynamodb.AttributeValue{
			"id":      &dynamodb.AttributeValue{S: aws.String(r.Id)},
			"app":     &dynamodb.AttributeValue{S: aws.String(r.App)},
			"created": &dynamodb.AttributeValue{S: aws.String(r.Created.Format(SortableTime))},
		},
		TableName: aws.String(releasesTable(r.App)),
	}

	if r.Build != "" {
		(*req.Item)["build"] = &dynamodb.AttributeValue{S: aws.String(r.Build)}
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

	return S3Put(app.Outputs["Settings"], fmt.Sprintf("releases/%s/env", r.Id), env, true)
}

func (r *Release) Promote() error {
	formation, err := r.Formation()

	if err != nil {
		return err
	}

	existing, err := formationParameters(formation)

	if err != nil {
		return err
	}

	app, err := GetApp(r.App)

	if err != nil {
		return err
	}

	pss, err := r.Processes()

	if err != nil {
		return err
	}

	for _, ps := range pss {
		app.Parameters[fmt.Sprintf("%sCommand", UpperName(ps.Name))] = ps.Command
		app.Parameters[fmt.Sprintf("%sImage", UpperName(ps.Name))] = fmt.Sprintf("%s/%s-%s:%s", os.Getenv("REGISTRY_HOST"), r.App, ps.Name, r.Build)
		app.Parameters[fmt.Sprintf("%sScale", UpperName(ps.Name))] = strconv.Itoa(ps.Count)
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

	req := &cloudformation.UpdateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(r.App),
		TemplateBody: aws.String(formation),
		Parameters:   params,
	}

	_, err = CloudFormation().UpdateStack(req)

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
	args := []string{"run", "-i", os.Getenv("DOCKER_IMAGE_APP"), "-mode", "staging"}

	cmd := exec.Command("docker", args...)
	cmd.Stderr = os.Stderr

	in, err := cmd.StdinPipe()

	if err != nil {
		return "", err
	}

	out, err := cmd.StdoutPipe()

	if err != nil {
		return "", err
	}

	err = cmd.Start()

	if err != nil {
		return "", err
	}

	io.WriteString(in, r.Manifest)
	in.Close()

	data, err := ioutil.ReadAll(out)

	if err != nil {
		return "", err
	}

	err = cmd.Wait()

	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (r *Release) Processes() (Processes, error) {
	manifest, err := LoadManifest(r.Manifest)

	fmt.Printf("%+v\n", manifest)

	if err != nil {
		return nil, err
	}

	return manifest.Processes(), nil
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
