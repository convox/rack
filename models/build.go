package models

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/dynamodb"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/kinesis"
)

type Build struct {
	Id string

	App string

	Logs    string
	Release string
	Status  string

	Started time.Time
	Ended   time.Time

	kinesis string
}

type Builds []Build

func NewBuild(app string) Build {
	return Build{
		Id:  generateId("B", 10),
		App: app,

		Status: "created",
	}
}

func ListBuilds(app string, last map[string]string) (Builds, error) {
	req := &dynamodb.QueryInput{
		KeyConditions: &map[string]*dynamodb.Condition{
			"app": &dynamodb.Condition{
				AttributeValueList: []*dynamodb.AttributeValue{&dynamodb.AttributeValue{S: aws.String(app)}},
				ComparisonOperator: aws.String("EQ"),
			},
		},
		IndexName:        aws.String("app.created"),
		Limit:            aws.Long(10),
		ScanIndexForward: aws.Boolean(false),
		TableName:        aws.String(buildsTable(app)),
	}

	if last["id"] != "" {
		req.ExclusiveStartKey = &map[string]*dynamodb.AttributeValue{
			"app":     &dynamodb.AttributeValue{S: aws.String(app)},
			"id":      &dynamodb.AttributeValue{S: aws.String(last["id"])},
			"created": &dynamodb.AttributeValue{S: aws.String(last["created"])},
		}
	}

	res, err := DynamoDB().Query(req)

	if err != nil {
		return nil, err
	}

	builds := make(Builds, len(res.Items))

	for i, item := range res.Items {
		builds[i] = *buildFromItem(*item)
	}

	return builds, nil
}

func GetBuild(app, id string) (*Build, error) {
	req := &dynamodb.GetItemInput{
		ConsistentRead: aws.Boolean(true),
		Key: &map[string]*dynamodb.AttributeValue{
			"id": &dynamodb.AttributeValue{S: aws.String(id)},
		},
		TableName: aws.String(buildsTable(app)),
	}

	res, err := DynamoDB().GetItem(req)

	if err != nil {
		return nil, err
	}

	build := buildFromItem(*res.Item)

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
		Item: &map[string]*dynamodb.AttributeValue{
			"id":      &dynamodb.AttributeValue{S: aws.String(b.Id)},
			"app":     &dynamodb.AttributeValue{S: aws.String(b.App)},
			"status":  &dynamodb.AttributeValue{S: aws.String(b.Status)},
			"created": &dynamodb.AttributeValue{S: aws.String(b.Started.Format(SortableTime))},
		},
		TableName: aws.String(buildsTable(b.App)),
	}

	if b.Logs != "" {
		(*req.Item)["logs"] = &dynamodb.AttributeValue{S: aws.String(b.Logs)}
	}

	if b.Release != "" {
		(*req.Item)["release"] = &dynamodb.AttributeValue{S: aws.String(b.Release)}
	}

	if !b.Ended.IsZero() {
		(*req.Item)["ended"] = &dynamodb.AttributeValue{S: aws.String(b.Ended.Format(SortableTime))}
	}

	_, err := DynamoDB().PutItem(req)

	return err
}

func (b *Build) Cleanup() error {
	// TODO: store Ami on build and clean up from here
	// and remove the ami cleanup in release.Cleanup()

	// app, err := GetApp(b.App)

	// if err != nil {
	//   return err
	// }

	// // delete ami
	// req := &ec2.DeregisterImageRequest{
	//   ImageID: aws.String(b.Ami),
	// }

	// return EC2.DeregisterImage(req)

	return nil
}

func (b *Build) ExecuteLocal(r io.Reader) {
	b.Status = "building"
	b.Save()

	name := b.App

	args := []string{"run", "-i", "-v", "/var/run/docker.sock:/var/run/docker.sock", fmt.Sprintf("convox/build:%s", os.Getenv("RELEASE")), "-id", b.Id, "-push", os.Getenv("REGISTRY_HOST"), "-auth", os.Getenv("REGISTRY_PASSWORD"), name, "-"}

	err := b.execute(args, r)

	if err != nil {
		b.Fail(err)
	}
}

func (b *Build) ExecuteRemote(repo string) {
	b.Status = "building"
	b.Save()

	name := b.App

	args := []string{"run", "-v", "/var/run/docker.sock:/var/run/docker.sock", fmt.Sprintf("convox/build:%s", os.Getenv("RELEASE")), "-id", b.Id, "-push", os.Getenv("REGISTRY_HOST"), "-auth", os.Getenv("REGISTRY_PASSWORD"), name}

	parts := strings.Split(repo, "#")

	if len(parts) > 1 {
		args = append(args, strings.Join(parts[0:len(parts)-1], "#"), parts[len(parts)-1])
	} else {
		args = append(args, repo)
	}

	err := b.execute(args, nil)

	if err != nil {
		b.Fail(err)
	}
}

func (b *Build) execute(args []string, r io.Reader) error {
	app, err := GetApp(b.App)

	if err != nil {
		return err
	}

	cmd := exec.Command("docker", args...)

	stdin, err := cmd.StdinPipe()

	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout

	if err != nil {
		return err
	}

	if err = cmd.Start(); err != nil {
		return err
	}

	if r != nil {
		_, err := io.Copy(stdin, r)

		if err != nil {
			return err
		}

		stdin.Close()
	}

	// Every 2 seconds check for new logs and save
	ticker := time.Tick(1 * time.Second)

	logs := ""

	go func() {
		for _ = range ticker {
			if b.Logs != logs {
				b.Save()
				logs = b.Logs
			}
		}
	}()

	manifest := ""
	success := true
	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "|", 2)

		if len(parts) < 2 {
			b.log(parts[0])
			continue
		}

		switch parts[0] {
		case "manifest":
			manifest += fmt.Sprintf("%s\n", parts[1])
		case "error":
			success = false
			fmt.Println(parts[1])
			b.log(parts[1])
		default:
			fmt.Println(parts[1])
			b.log(parts[1])
		}
	}

	err = cmd.Wait()

	// close(quit)

	if err != nil {
		return err
	}

	if !success {
		return fmt.Errorf("error from builder")
	}

	release, err := app.ForkRelease()

	if err != nil {
		return err
	}

	release.Build = b.Id
	release.Manifest = manifest

	err = release.Save()

	if err != nil {
		return err
	}

	b.Release = release.Id
	b.Status = "complete"
	b.Ended = time.Now()
	b.Save()

	return nil
}

func (b *Build) Fail(err error) {
	b.Status = "failed"
	b.Ended = time.Now()
	b.log(fmt.Sprintf("Build Error: %s", err))
	b.Save()
}

func (b *Build) Image(process string) string {
	return fmt.Sprintf("%s/%s-%s:%s", os.Getenv("REGISTRY"), b.App, process, b.Id)
}

func (b *Build) log(line string) {
	b.Logs += fmt.Sprintf("%s\n", line)

	if b.kinesis == "" {
		app, err := GetApp(b.App)

		if err != nil {
			panic(err)
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
		panic(err)
	}

	// record to kinesis
}

func buildsTable(app string) string {
	return os.Getenv("DYNAMO_BUILDS")
}

func buildFromItem(item map[string]*dynamodb.AttributeValue) *Build {
	started, _ := time.Parse(SortableTime, coalesce(item["created"], ""))
	ended, _ := time.Parse(SortableTime, coalesce(item["ended"], ""))

	return &Build{
		Id:      coalesce(item["id"], ""),
		App:     coalesce(item["app"], ""),
		Logs:    coalesce(item["logs"], ""),
		Release: coalesce(item["release"], ""),
		Status:  coalesce(item["status"], ""),
		Started: started,
		Ended:   ended,
	}
}
