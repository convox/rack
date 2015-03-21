package controllers

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/convox/kernel/builder/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/builder/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/dynamodb"
	"github.com/convox/kernel/builder/Godeps/_workspace/src/github.com/gorilla/mux"
)

var SortableTime = "20060102.150405.000000000"

var (
	DynamoDB = dynamodb.New(aws.Creds(os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"), ""), os.Getenv("AWS_REGION"), nil)
)

func Build(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	form := ParseForm(r)

	go executeBuild(vars["app"], form["repo"])

	RenderText(rw, `{"status":"ok"}`)
}

func awsEnvironment() string {
	env := []string{
		fmt.Sprintf("AWS_REGION=%s", os.Getenv("AWS_REGION")),
		fmt.Sprintf("AWS_ACCESS=%s", os.Getenv("AWS_ACCESS")),
		fmt.Sprintf("AWS_SECRET=%s", os.Getenv("AWS_SECRET")),
	}
	return strings.Join(env, "\n")
}

func recoverBuild(app, id string) {
	if r := recover(); r != nil {
		err := updateBuild(app, id, "failed", "", r.(error).Error())

		if err != nil {
			fmt.Printf("error during recovery: %v\n", err)
		}
	}
}

func executeBuild(app, repo string) {
	id, err := createBuild(app)
	fmt.Printf("err %+v\n", err)

	defer recoverBuild(app, id)

	base, err := ioutil.TempDir("", "build")

	if err != nil {
		panic(err)
	}

	env := filepath.Join(base, ".env")

	if err = ioutil.WriteFile(env, []byte(awsEnvironment()), 0400); err != nil {
		panic(err)
	}

	cmd := exec.Command("docker", "run", "--env-file", env, "convox/builder", repo, app)
	stdout, err := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr

	if err = cmd.Start(); err != nil {
		panic(err)
	}

	manifest := ""
	logs := ""
	success := false

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "|", 2)

		if len(parts) < 2 {
			fmt.Printf("unknown | %s\n", scanner.Text())
			continue
		}

		switch parts[0] {
		case "manifest":
			manifest += fmt.Sprintf("%s\n", parts[1])
		case "packer":
			fmt.Printf("packer | %s\n", parts[1])
		case "build":
			fmt.Printf("build | %s\n", parts[1])
			logs += fmt.Sprintf("%s\n", parts[1])
		case "error":
			fmt.Printf("error| %s\n", parts[1])
		case "ami":
			release, err := createRelease(app, parts[1], manifest)

			if err != nil {
				panic(err)
			}

			err = updateBuild(app, id, "complete", release, logs)

			if err != nil {
				panic(err)
			}

			success = true
		default:
			fmt.Printf("unknown | %s\n", parts[1])
		}
	}

	err = cmd.Wait()

	if !success || err != nil {
		fmt.Printf("build failed\n")
		err = updateBuild(app, id, "failed", "", logs)
	}
}

func createBuild(app string) (string, error) {
	id := generateId("B", 9)

	defer recoverBuild(app, id)

	created := time.Now().Format(SortableTime)

	req := &dynamodb.PutItemInput{
		Item: map[string]dynamodb.AttributeValue{
			"app":     dynamodb.AttributeValue{S: aws.String(app)},
			"created": dynamodb.AttributeValue{S: aws.String(created)},
			"id":      dynamodb.AttributeValue{S: aws.String(id)},
			"status":  dynamodb.AttributeValue{S: aws.String("building")},
		},
		TableName: aws.String(buildsTable(app)),
	}

	_, err := DynamoDB.PutItem(req)

	if err != nil {
		return "", err
	}

	return id, nil
}

func updateBuild(app, id, status, release, logs string) error {
	req := &dynamodb.GetItemInput{
		ConsistentRead: aws.Boolean(true),
		Key: map[string]dynamodb.AttributeValue{
			"id": dynamodb.AttributeValue{S: aws.String(id)},
		},
		TableName: aws.String(buildsTable(app)),
	}

	row, err := DynamoDB.GetItem(req)

	if err != nil {
		return err
	}

	if len(row.Item) == 0 {
		return fmt.Errorf("no such build: %s", id)
	}

	build := row.Item

	ended := time.Now().Format(SortableTime)

	build["ended"] = dynamodb.AttributeValue{S: aws.String(ended)}
	build["status"] = dynamodb.AttributeValue{S: aws.String(status)}
	build["logs"] = dynamodb.AttributeValue{S: aws.String(logs)}

	if release != "" {
		build["release"] = dynamodb.AttributeValue{S: aws.String(release)}
	}

	preq := &dynamodb.PutItemInput{
		Item:      build,
		TableName: aws.String(buildsTable(app)),
	}

	_, err = DynamoDB.PutItem(preq)

	return err
}

func createRelease(app, ami, manifest string) (string, error) {
	id := generateId("R", 9)
	created := time.Now().Format(SortableTime)

	req := &dynamodb.PutItemInput{
		Item: map[string]dynamodb.AttributeValue{
			"app":      dynamodb.AttributeValue{S: aws.String(app)},
			"created":  dynamodb.AttributeValue{S: aws.String(created)},
			"id":       dynamodb.AttributeValue{S: aws.String(id)},
			"ami":      dynamodb.AttributeValue{S: aws.String(ami)},
			"manifest": dynamodb.AttributeValue{S: aws.String(manifest)},
		},
		TableName: aws.String(buildsTable(app)),
	}

	_, err := DynamoDB.PutItem(req)

	if err != nil {
		return "", err
	}

	return id, nil
}

func buildsTable(app string) string {
	return fmt.Sprintf("%s-builds", app)
}

func releasesTable(app string) string {
	return fmt.Sprintf("%s-releases", app)
}

var idAlphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func generateId(prefix string, size int) string {
	b := make([]rune, size)
	for i := range b {
		b[i] = idAlphabet[rand.Intn(len(idAlphabet))]
	}
	return prefix + string(b)
}
