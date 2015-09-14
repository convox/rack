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

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/dynamodb"
	"github.com/awslabs/aws-sdk-go/service/kinesis"
)

type Build struct {
	Id       string `json:"id"`
	App      string `json:"app"`
	Logs     string `json:"logs"`
	Manifest string `json:"manifest"`
	Release  string `json:"release"`
	Status   string `json:"status"`

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
		KeyConditions: &map[string]*dynamodb.Condition{
			"app": &dynamodb.Condition{
				AttributeValueList: []*dynamodb.AttributeValue{&dynamodb.AttributeValue{S: aws.String(app)}},
				ComparisonOperator: aws.String("EQ"),
			},
		},
		IndexName:        aws.String("app.created"),
		Limit:            aws.Long(20),
		ScanIndexForward: aws.Boolean(false),
		TableName:        aws.String(buildsTable(app)),
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

	if res.Item == nil {
		return nil, fmt.Errorf("no such build: %s", id)
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

	if b.Manifest != "" {
		(*req.Item)["manifest"] = &dynamodb.AttributeValue{S: aws.String(b.Manifest)}
	}

	if b.Release != "" {
		(*req.Item)["release"] = &dynamodb.AttributeValue{S: aws.String(b.Release)}
	}

	if !b.Ended.IsZero() {
		(*req.Item)["ended"] = &dynamodb.AttributeValue{S: aws.String(b.Ended.Format(SortableTime))}
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

func (b *Build) ExecuteLocal(r io.Reader, ch chan error) {
	b.Status = "building"
	b.Save()

	name := b.App

	args := []string{"run", "-i", "--name", fmt.Sprintf("build-%s", b.Id), "-v", "/var/run/docker.sock:/var/run/docker.sock", os.Getenv("DOCKER_IMAGE_BUILD"), "-id", b.Id, "-push", os.Getenv("REGISTRY_HOST")}

	if pw := os.Getenv("PASSWORD"); pw != "" {
		args = append(args, "-auth", pw)
	}

	args = append(args, name, "-")

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

	args := []string{"run", "--name", fmt.Sprintf("build-%s", b.Id), "-v", "/var/run/docker.sock:/var/run/docker.sock", os.Getenv("DOCKER_IMAGE_BUILD"), "-id", b.Id, "-push", os.Getenv("REGISTRY_HOST")}

	if pw := os.Getenv("PASSWORD"); pw != "" {
		args = append(args, "-auth", pw)
	}

	args = append(args, name)

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
		parts := strings.SplitN(scanner.Text(), "|", 2)

		if len(parts) < 2 {
			b.log(parts[0])
			continue
		}

		switch parts[0] {
		case "manifest":
			b.Manifest += fmt.Sprintf("%s\n", parts[1])
		case "error":
			b.log(fmt.Sprintf("ERROR: %s", parts[1]))
			b.Status = "failed"
		default:
			b.log(parts[1])
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

	logs := ""
	var err error

	if item["logs"] == nil {
		logs, err = getS3BuildLogs(coalesce(item["app"], ""), coalesce(item["id"], ""))

		if err != nil {
			logs = ""
		}
	}

	return &Build{
		Id:       coalesce(item["id"], ""),
		App:      coalesce(item["app"], ""),
		Logs:     coalesce(item["logs"], logs),
		Manifest: coalesce(item["manifest"], ""),
		Release:  coalesce(item["release"], ""),
		Status:   coalesce(item["status"], ""),
		Started:  started,
		Ended:    ended,
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
