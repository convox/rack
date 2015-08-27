package models

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/dynamodb"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/kinesis"
)

type Build struct {
	Id string

	App string

	Logs     string
	Manifest string
	Release  string
	Status   string

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
		logMax := 1024 * 395 // Dynamo attribute can be 400k max

		var logs string
		var key string
		remainder := b.Logs
		counter := 0

		for len(remainder) > 0 {
			if len(remainder) > logMax {
				logs = remainder[0:logMax]
				remainder = remainder[logMax:]
			} else {
				logs = remainder
				remainder = ""
			}

			if counter == 0 {
				key = "logs"
			} else {
				key = fmt.Sprintf("logs%d", counter)
			}

			(*req.Item)[key] = &dynamodb.AttributeValue{S: aws.String(logs)}

			counter += 1
		}
	}

	if b.Manifest != "" {
		(*req.Item)["manifest"] = &dynamodb.AttributeValue{S: aws.String(b.Manifest)}
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

func (b *Build) ExecuteLocal(r io.Reader, ch chan error) {
	b.Status = "building"
	b.Save()

	name := b.App

	args := []string{"run", "-i", "--name", fmt.Sprintf("build-%s", b.Id), "-v", "/var/run/docker.sock:/var/run/docker.sock", fmt.Sprintf("convox/build:%s", os.Getenv("RELEASE")), "-id", b.Id, "-push", os.Getenv("REGISTRY_HOST"), "-auth", os.Getenv("PASSWORD"), name, "-"}

	err := b.execute(args, r, ch)

	if err != nil {
		fmt.Printf("ns=kernel cn=build at=ExecuteLocal state=error step=build.execute app=%q build=%q error=%q\n", b.App, b.Id, err)
		b.Fail(err)
		ch <- err
	} else {
		fmt.Printf("ns=kernel cn=build at=ExecuteLocal state=success step=build.execute app=%q build=%q\n", b.App, b.Id)
	}
}

func (b *Build) ExecuteRemote(repo string, ch chan error) {
	b.Status = "building"
	b.Save()

	name := b.App

	args := []string{"run", "--name", fmt.Sprintf("build-%s", b.Id), "-v", "/var/run/docker.sock:/var/run/docker.sock", fmt.Sprintf("convox/build:%s", os.Getenv("RELEASE")), "-id", b.Id, "-push", os.Getenv("REGISTRY_HOST"), "-auth", os.Getenv("PASSWORD"), name}

	parts := strings.Split(repo, "#")

	if len(parts) > 1 {
		args = append(args, strings.Join(parts[0:len(parts)-1], "#"), parts[len(parts)-1])
	} else {
		args = append(args, repo)
	}

	err := b.execute(args, nil, ch)

	if err != nil {
		fmt.Printf("ns=kernel cn=build at=ExecuteRemote state=error step=build.execute app=%q build=%q error=%q\n", b.App, b.Id, err)
		b.Fail(err)
		ch <- err
	} else {
		fmt.Printf("ns=kernel cn=build at=ExecuteRemote state=success step=build.execute app=%q build=%q\n", b.App, b.Id)
	}
}

func (b *Build) execute(args []string, r io.Reader, ch chan error) error {
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

	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()

	if err != nil {
		return err
	}

	if err = cmd.Start(); err != nil {
		return err
	}

	for {
		err := exec.Command("docker", "logs", fmt.Sprintf("build-%s", b.Id)).Run()

		time.Sleep(200 * time.Millisecond)

		if err == nil {
			break
		}
	}

	ch <- nil // notify that start was ok

	if r != nil {
		_, err := io.Copy(stdin, r)

		if err != nil {
			return err
		}
	}

	stdin.Close()

	var wg sync.WaitGroup

	wg.Add(2)
	go b.scanLines(stdout, &wg)
	go b.scanLines(stderr, &wg)
	wg.Wait()

	if err = cmd.Wait(); err != nil {
		return err
	}

	err = b.Save()

	if err != nil {
		return err
	}

	if b.Status == "failed" {
		return fmt.Errorf("error from builder")
	}

	release, err := app.ForkRelease()

	if err != nil {
		return err
	}

	release.Build = b.Id
	release.Manifest = b.Manifest

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

func (b *Build) scanLines(r io.Reader, wg *sync.WaitGroup) {
	defer wg.Done()

	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		b.log(scanner.Text())

		parts := strings.SplitN(scanner.Text(), "|", 2)

		switch parts[0] {
		case "manifest":
			b.Manifest += fmt.Sprintf("%s\n", parts[1])
		case "error":
			b.Status = "failed"
		}
	}
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
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}
}

func buildsTable(app string) string {
	return os.Getenv("DYNAMO_BUILDS")
}

func buildFromItem(item map[string]*dynamodb.AttributeValue) *Build {
	started, _ := time.Parse(SortableTime, coalesce(item["created"], ""))
	ended, _ := time.Parse(SortableTime, coalesce(item["ended"], ""))

	return &Build{
		Id:       coalesce(item["id"], ""),
		App:      coalesce(item["app"], ""),
		Logs:     coalesce(item["logs"], ""),
		Manifest: coalesce(item["manifest"], ""),
		Release:  coalesce(item["release"], ""),
		Status:   coalesce(item["status"], ""),
		Started:  started,
		Ended:    ended,
	}
}
