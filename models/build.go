package models

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/dynamodb"
)

type Build struct {
	Id string

	Cluster string
	App     string

	Logs    string
	Release string
	Status  string

	Started time.Time
	Ended   time.Time
}

type Builds []Build

func NewBuild(cluster, app string) Build {
	return Build{
		Id:      generateId("B", 10),
		Cluster: cluster,
		App:     app,

		Status: "created",
	}
}

func ListBuilds(cluster, app string) (Builds, error) {
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
		TableName:        aws.String(buildsTable(cluster, app)),
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
			"cluster": &dynamodb.AttributeValue{S: aws.String(b.Cluster)},
			"app":     &dynamodb.AttributeValue{S: aws.String(b.App)},
			"status":  &dynamodb.AttributeValue{S: aws.String(b.Status)},
			"created": &dynamodb.AttributeValue{S: aws.String(b.Started.Format(SortableTime))},
		},
		TableName: aws.String(buildsTable(b.Cluster, b.App)),
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

func (b *Build) Execute(repo string) {
	b.Status = "building"
	b.Save()

	name := fmt.Sprintf("%s-%s", b.Cluster, b.App)

	cmd := exec.Command("docker", "run", "-v", "/var/run/docker.sock:/var/run/docker.sock", "convox/build", "-id", b.Id, "-push", os.Getenv("REGISTRY"), name, repo)
	stdout, err := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout

	if err = cmd.Start(); err != nil {
		// TODO log error
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return
	}

	manifest := ""
	success := true
	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "|", 2)

		if len(parts) < 2 {
			b.Logs += fmt.Sprintf("%s\n", parts[0])
			continue
		}

		switch parts[0] {
		case "manifest":
			manifest += fmt.Sprintf("%s\n", parts[1])
		case "error":
			success = false
			fmt.Println(parts[1])
			b.Logs += fmt.Sprintf("%s\n", parts[1])
		default:
			fmt.Println(parts[1])
			b.Logs += fmt.Sprintf("%s\n", parts[1])
		}
	}

	err = cmd.Wait()

	if err != nil {
		b.Fail(err)
		return
	}

	if !success {
		b.Fail(fmt.Errorf("error from builder"))
		return
	}

	app, err := GetApp(b.Cluster, b.App)

	if err != nil {
		b.Fail(err)
		return
	}

	release, err := app.ForkRelease()

	if err != nil {
		b.Fail(err)
		return
	}

	release.Build = b.Id
	release.Manifest = manifest

	err = release.Save()

	if err != nil {
		b.Fail(err)
		return
	}

	b.Release = release.Id
	b.Status = "complete"
	b.Ended = time.Now()
	b.Save()
}

func (b *Build) Fail(err error) {
	b.Status = "failed"
	b.Ended = time.Now()
	b.Logs += fmt.Sprintf("Build Error: %s\n", err)
	b.Save()
}

func (b *Build) Image(process string) string {
	return fmt.Sprintf("%s/%s-%s-%s:%s", os.Getenv("REGISTRY"), b.Cluster, b.App, process, b.Id)
}

func buildsTable(cluster, app string) string {
	return fmt.Sprintf("%s-%s-builds", cluster, app)
}

func buildFromItem(item map[string]*dynamodb.AttributeValue) *Build {
	started, _ := time.Parse(SortableTime, coalesce(item["created"], ""))
	ended, _ := time.Parse(SortableTime, coalesce(item["ended"], ""))

	return &Build{
		Id:      coalesce(item["id"], ""),
		App:     coalesce(item["app"], ""),
		Cluster: coalesce(item["cluster"], ""),
		Logs:    coalesce(item["logs"], ""),
		Release: coalesce(item["release"], ""),
		Status:  coalesce(item["status"], ""),
		Started: started,
		Ended:   ended,
	}
}
