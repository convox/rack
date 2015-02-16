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

	"github.com/convox/kernel/builder/Godeps/_workspace/src/github.com/gorilla/mux"

	caws "github.com/convox/kernel/builder/Godeps/_workspace/src/github.com/crowdmob/goamz/aws"
	"github.com/convox/kernel/builder/Godeps/_workspace/src/github.com/crowdmob/goamz/dynamodb"
	"github.com/convox/kernel/builder/Godeps/_workspace/src/github.com/crowdmob/goamz/ec2"

	gaws "github.com/convox/kernel/builder/Godeps/_workspace/src/github.com/goamz/goamz/aws"
	"github.com/convox/kernel/builder/Godeps/_workspace/src/github.com/goamz/goamz/cloudformation"
)

var SortableTime = "20060102.150405.000000000"

var (
	cauth = caws.Auth{AccessKey: os.Getenv("AWS_ACCESS"), SecretKey: os.Getenv("AWS_SECRET")}
	gauth = gaws.Auth{AccessKey: os.Getenv("AWS_ACCESS"), SecretKey: os.Getenv("AWS_SECRET")}
)

var (
	CloudFormation = cloudformation.New(gauth, gaws.Regions[os.Getenv("AWS_REGION")])
	DynamoDB       = dynamodb.New(cauth, caws.Regions[os.Getenv("AWS_REGION")])
	EC2            = ec2.New(cauth, caws.Regions[os.Getenv("AWS_REGION")])
)

func init() {
}

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

	build := []dynamodb.Attribute{
		*dynamodb.NewStringAttribute("app", app),
		*dynamodb.NewStringAttribute("created", created),
		*dynamodb.NewStringAttribute("id", id),
		*dynamodb.NewStringAttribute("status", "building"),
	}

	_, err := buildsTable(app).PutItem(id, "", build)

	if err != nil {
		return "", err
	}

	return id, nil
}

func updateBuild(app, id, status, release, logs string) error {
	row, err := buildsTable(app).GetItem(&dynamodb.Key{HashKey: id})

	if err != nil {
		return err
	}

	build := []dynamodb.Attribute{}

	for key, attr := range row {
		build = append(build, *dynamodb.NewStringAttribute(key, attr.Value))
	}

	ended := time.Now().Format(SortableTime)

	build = append(build, *dynamodb.NewStringAttribute("ended", ended))
	build = append(build, *dynamodb.NewStringAttribute("status", status))
	build = append(build, *dynamodb.NewStringAttribute("logs", logs))

	if release != "" {
		build = append(build, *dynamodb.NewStringAttribute("release", release))
	}

	_, err = buildsTable(app).PutItem(id, "", build)

	return err
}

func createRelease(app, ami, manifest string) (string, error) {
	id := generateId("R", 9)
	created := time.Now().Format(SortableTime)

	release := []dynamodb.Attribute{
		*dynamodb.NewStringAttribute("app", app),
		*dynamodb.NewStringAttribute("created", created),
		*dynamodb.NewStringAttribute("id", id),
		*dynamodb.NewStringAttribute("ami", ami),
		*dynamodb.NewStringAttribute("manifest", manifest),
	}

	_, err := releasesTable(app).PutItem(id, "", release)

	return id, err
}

func coalesce(att *dynamodb.Attribute, def string) string {
	if att != nil {
		return att.Value
	} else {
		return def
	}
}

func buildsTable(app string) *dynamodb.Table {
	pk := dynamodb.PrimaryKey{dynamodb.NewStringAttribute("id", ""), nil}
	table := DynamoDB.NewTable(fmt.Sprintf("%s-builds", app), pk)
	return table
}

func releasesTable(app string) *dynamodb.Table {
	pk := dynamodb.PrimaryKey{dynamodb.NewStringAttribute("id", ""), nil}
	table := DynamoDB.NewTable(fmt.Sprintf("%s-releases", app), pk)
	return table
}

var idAlphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func generateId(prefix string, size int) string {
	b := make([]rune, size)
	for i := range b {
		b[i] = idAlphabet[rand.Intn(len(idAlphabet))]
	}
	return prefix + string(b)
}
