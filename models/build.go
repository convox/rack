package models

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/dynamodb"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"
)

var (
	log = logger.New("ns=kernel md=build")
)

type Build struct {
	Id string

	App string

	Logs    string
	Release string
	Status  string

	Started time.Time
	Ended   time.Time
}

type Builds []Build

func ListBuilds(app string) (Builds, error) {
	req := &dynamodb.QueryInput{
		KeyConditions: map[string]dynamodb.Condition{
			"app": dynamodb.Condition{
				AttributeValueList: []dynamodb.AttributeValue{dynamodb.AttributeValue{S: aws.String(app)}},
				ComparisonOperator: aws.String("EQ"),
			},
		},
		IndexName:        aws.String("app.created"),
		Limit:            aws.Integer(10),
		ScanIndexForward: aws.Boolean(false),
		TableName:        aws.String(buildsTable(app)),
	}

	res, err := DynamoDB.Query(req)

	if err != nil {
		return nil, err
	}

	builds := make(Builds, len(res.Items))

	for i, item := range res.Items {
		builds[i] = *buildFromItem(item)
	}

	return builds, nil
}

func (b *Build) Save() error {
	if b.Id == "" {
		b.Id = generateId("B", 10)
	}

	if b.Started.IsZero() {
		b.Started = time.Now()
	}

	req := &dynamodb.PutItemInput{
		Item: map[string]dynamodb.AttributeValue{
			"id":      dynamodb.AttributeValue{S: aws.String(b.Id)},
			"app":     dynamodb.AttributeValue{S: aws.String(b.App)},
			"status":  dynamodb.AttributeValue{S: aws.String(b.Status)},
			"created": dynamodb.AttributeValue{S: aws.String(b.Started.Format(SortableTime))},
		},
		TableName: aws.String(buildsTable(b.App)),
	}

	if b.Logs != "" {
		req.Item["logs"] = dynamodb.AttributeValue{S: aws.String(b.Logs)}
	}

	if b.Release != "" {
		req.Item["release"] = dynamodb.AttributeValue{S: aws.String(b.Release)}
	}

	if !b.Ended.IsZero() {
		req.Item["ended"] = dynamodb.AttributeValue{S: aws.String(b.Ended.Format(SortableTime))}
	}

	_, err := DynamoDB.PutItem(req)

	return err
}

func (b *Build) Execute(repo string) {
	log = log.At("execute").Start()

	defer b.recoverBuild(log)

	base, err := ioutil.TempDir("", "build")

	if err != nil {
		log.Error(err)
		return
	}

	env := filepath.Join(base, ".env")

	if err = ioutil.WriteFile(env, []byte(awsEnvironment()), 0400); err != nil {
		log.Error(err)
		return
	}

	cmd := exec.Command("docker", "run", "--env-file", env, "convox/builder:release", repo, b.App)
	stdout, err := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr

	if err = cmd.Start(); err != nil {
		log.Error(err)
		return
	}

	release := &Release{App: b.App}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "|", 2)

		if len(parts) < 2 {
			log.Log("type=unknown text=%q", scanner.Text())
			continue
		}

		switch parts[0] {
		case "manifest":
			release.Manifest += fmt.Sprintf("%s\n", parts[1])
		case "packer":
			log.Log("type=packer text=%q", parts[1])
		case "build":
			log.Log("type=build text=%q", parts[1])
			b.Logs += fmt.Sprintf("%s\n", parts[1])
		case "error":
			log.Log("type=error text=%q", parts[1])
		case "ami":
			release.Ami = parts[1]

			if err := release.Save(); err != nil {
				log.Error(err)
				b.Fail(err)
				return
			}

			b.Release = release.Id
			b.Status = "complete"
			b.Ended = time.Now()
			b.Save()

			if err != nil {
				log.Error(err)
				return
			}
		default:
			log.Log("type=unknown text=%q", parts[1])
		}
	}

	err = cmd.Wait()

	if err != nil {
		log.Error(err)
		b.Fail(err)
		return
	}

	if release.Ami == "" {
		err = fmt.Errorf("build did not create ami")
		log.Error(err)
		b.Fail(err)
		return
	}
}

func (b *Build) Fail(err error) {
	b.Status = "failed"
	b.Logs += fmt.Sprintf("\nBuild Error:\n %s", err)
	b.Save()
}

func (b *Build) recoverBuild(log *logger.Logger) {
}

func buildsTable(app string) string {
	return fmt.Sprintf("%s-builds", app)
}

func buildFromItem(item map[string]dynamodb.AttributeValue) *Build {
	started, _ := time.Parse(SortableTime, coalesce(item["created"].S, ""))
	ended, _ := time.Parse(SortableTime, coalesce(item["ended"].S, ""))

	return &Build{
		Id:      coalesce(item["id"].S, ""),
		App:     coalesce(item["app"].S, ""),
		Logs:    coalesce(item["logs"].S, ""),
		Release: coalesce(item["release"].S, ""),
		Status:  coalesce(item["status"].S, ""),
		Started: started,
		Ended:   ended,
	}
}
