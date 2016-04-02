package models

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/convox/rack/api/helpers"
)

type Build struct {
	Id       string `json:"id"`
	App      string `json:"app"`
	Logs     string `json:"logs"`
	Manifest string `json:"manifest"`
	Release  string `json:"release"`
	Status   string `json:"status"`

	Description string `json:"description"`

	Started time.Time `json:"started"`
	Ended   time.Time `json:"ended"`

	kinesis string `json:"-"`
}

type Builds []Build

func NewBuild(app string) Build {
	return Build{
		App:    app,
		Id:     generateId("B", 10),
		Status: "created",
	}
}

func ListBuilds(app string) (Builds, error) {
	req := &dynamodb.QueryInput{
		KeyConditions: map[string]*dynamodb.Condition{
			"app": &dynamodb.Condition{
				AttributeValueList: []*dynamodb.AttributeValue{&dynamodb.AttributeValue{S: aws.String(app)}},
				ComparisonOperator: aws.String("EQ"),
			},
		},
		IndexName:        aws.String("app.created"),
		Limit:            aws.Int64(20),
		ScanIndexForward: aws.Bool(false),
		TableName:        aws.String(buildsTable(app)),
	}

	res, err := DynamoDB().Query(req)

	if err != nil {
		return nil, err
	}

	builds := make(Builds, len(res.Items))

	for i, item := range res.Items {
		builds[i] = *buildFromItem(item)
	}

	return builds, nil
}

func GetBuild(app, id string) (*Build, error) {
	req := &dynamodb.GetItemInput{
		ConsistentRead: aws.Bool(true),
		Key: map[string]*dynamodb.AttributeValue{
			"id": &dynamodb.AttributeValue{S: aws.String(id)},
		},
		TableName: aws.String(buildsTable(app)),
	}

	res, err := DynamoDB().GetItem(req)

	if err != nil {
		return nil, err
	}

	if res.Item == nil {
		return nil, fmt.Errorf("no such build: %s", id)
	}

	build := buildFromItem(res.Item)

	return build, nil
}

func (b *Build) Save() error {
	if b.Id == "" {
		return fmt.Errorf("Id can not be blank")
	}

	if b.Started.IsZero() {
		b.Started = time.Now()
	}

	req := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"id":      &dynamodb.AttributeValue{S: aws.String(b.Id)},
			"app":     &dynamodb.AttributeValue{S: aws.String(b.App)},
			"status":  &dynamodb.AttributeValue{S: aws.String(b.Status)},
			"created": &dynamodb.AttributeValue{S: aws.String(b.Started.Format(SortableTime))},
		},
		TableName: aws.String(buildsTable(b.App)),
	}

	if b.Description != "" {
		req.Item["description"] = &dynamodb.AttributeValue{S: aws.String(b.Description)}
	}

	if b.Manifest != "" {
		req.Item["manifest"] = &dynamodb.AttributeValue{S: aws.String(b.Manifest)}
	}

	if b.Release != "" {
		req.Item["release"] = &dynamodb.AttributeValue{S: aws.String(b.Release)}
	}

	if !b.Ended.IsZero() {
		req.Item["ended"] = &dynamodb.AttributeValue{S: aws.String(b.Ended.Format(SortableTime))}
	}

	a, err := GetApp(b.App)

	if err != nil {
		return err
	}

	err = S3Put(a.Outputs["Settings"], fmt.Sprintf("builds/%s.log", b.Id), []byte(b.Logs), true)

	if err != nil {
		return err
	}

	_, err = DynamoDB().PutItem(req)

	return err
}

func (b *Build) Cleanup() error {
	return nil
}

func (b *Build) copyError(err error) {
	NotifyError("build:copy", err, map[string]string{"id": b.Id, "app": b.App})
	b.Fail(err)
}

func (srcBuild *Build) CopyTo(destApp App) (*Build, error) {
	started := time.Now()

	srcApp, err := GetApp(srcBuild.App)

	if err != nil {
		return nil, err
	}

	// generate a new build ID
	destBuild := NewBuild(destApp.Name)

	// copy src build properties
	destBuild.Manifest = srcBuild.Manifest

	// set copy properties
	destBuild.Description = fmt.Sprintf("Copy of %s %s", srcBuild.App, srcBuild.Id)
	destBuild.Status = "copying"

	err = destBuild.Save()

	if err != nil {
		destBuild.copyError(err)
		return nil, err
	}

	// pull, tag, push images
	manifest, err := LoadManifest(destBuild.Manifest, &destApp)

	if err != nil {
		destBuild.copyError(err)
		return nil, err
	}

	for _, entry := range manifest {
		var srcImg, destImg string

		srcImg = entry.RegistryImage(srcApp, srcBuild.Id)
		destImg = entry.RegistryImage(&destApp, destBuild.Id)

		destBuild.Logs += fmt.Sprintf("RUNNING: docker pull %s\n", srcImg)

		out, err := exec.Command("docker", "pull", srcImg).CombinedOutput()

		if err != nil {
			destBuild.copyError(err)
			return nil, err
		}

		destBuild.Logs += string(out)

		destBuild.Logs += fmt.Sprintf("RUNNING: docker tag -f %s %s\n", srcImg, destImg)

		out, err = exec.Command("docker", "tag", "-f", srcImg, destImg).CombinedOutput()

		if err != nil {
			destBuild.copyError(err)
			return nil, err
		}

		destBuild.Logs += string(out)

		destBuild.Logs += fmt.Sprintf("RUNNING: docker push %s\n", destImg)

		out, err = exec.Command("docker", "push", destImg).CombinedOutput()

		if err != nil {
			destBuild.copyError(err)
			return nil, err
		}

		destBuild.Logs += string(out)

	}

	// make release for new build
	release, err := destApp.ForkRelease()

	if err != nil {
		destBuild.copyError(err)
		return nil, err
	}

	release.Build = destBuild.Id
	release.Manifest = destBuild.Manifest

	err = release.Save()

	if err != nil {
		destBuild.copyError(err)
		return nil, err
	}

	// finalize new build
	destBuild.Ended = time.Now()
	destBuild.Release = release.Id
	destBuild.Status = "complete"

	err = destBuild.Save()

	if err != nil {
		destBuild.copyError(err)
		return nil, err
	}

	NotifySuccess("build:copy", map[string]string{"id": destBuild.Id, "app": destBuild.App})
	helpers.TrackSuccess("build-copy", map[string]interface{}{"elapsed": time.Now().Sub(started).Nanoseconds() / 1000000})

	return &destBuild, nil
}

func (b *Build) Fail(err error) {
	b.Status = "failed"
	b.Ended = time.Now()
	b.log(fmt.Sprintf("Build Error: %s", err))
	b.Save()
}

func (b *Build) log(line string) {
	b.Logs += fmt.Sprintf("%s\n", line)

	if b.kinesis == "" {
		app, err := GetApp(b.App)

		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s", err)
			return
		}

		b.kinesis = app.Outputs["Kinesis"]
	}

	_, err := Kinesis().PutRecords(&kinesis.PutRecordsInput{
		StreamName: aws.String(b.kinesis),
		Records: []*kinesis.PutRecordsRequestEntry{
			&kinesis.PutRecordsRequestEntry{
				Data:         []byte(fmt.Sprintf("build: %s", line)),
				PartitionKey: aws.String(string(time.Now().UnixNano())),
			},
		},
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}
}

func buildsTable(app string) string {
	return os.Getenv("DYNAMO_BUILDS")
}

func buildFromItem(item map[string]*dynamodb.AttributeValue) *Build {
	started, _ := time.Parse(SortableTime, coalesce(item["created"], ""))
	ended, _ := time.Parse(SortableTime, coalesce(item["ended"], ""))

	logs := ""
	var err error

	if item["logs"] == nil {
		logs, err = getS3BuildLogs(coalesce(item["app"], ""), coalesce(item["id"], ""))

		if err != nil {
			logs = ""
		}
	}

	return &Build{
		Id:          coalesce(item["id"], ""),
		App:         coalesce(item["app"], ""),
		Description: coalesce(item["description"], ""),
		Logs:        coalesce(item["logs"], logs),
		Manifest:    coalesce(item["manifest"], ""),
		Release:     coalesce(item["release"], ""),
		Status:      coalesce(item["status"], ""),
		Started:     started,
		Ended:       ended,
	}
}

func getS3BuildLogs(app, build_id string) (string, error) {
	a, err := GetApp(app)

	if err != nil {
		return "", err
	}

	logs, err := s3Get(a.Outputs["Settings"], fmt.Sprintf("builds/%s.log", build_id))

	if err != nil {
		return "", err
	}

	return string(logs), nil
}
