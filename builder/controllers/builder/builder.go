package builder

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
	"github.com/convox/kernel/builder/controllers"

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
	form := controllers.ParseForm(r)

	go executeBuild(vars["cluster"], vars["app"], form["repo"])

	controllers.RenderText(rw, `{"status":"ok"}`)
}

func awsEnvironment() string {
	env := []string{
		fmt.Sprintf("AWS_REGION=%s", os.Getenv("AWS_REGION")),
		fmt.Sprintf("AWS_ACCESS=%s", os.Getenv("AWS_ACCESS")),
		fmt.Sprintf("AWS_SECRET=%s", os.Getenv("AWS_SECRET")),
	}
	return strings.Join(env, "\n")
}

func executeBuild(cluster, app, repo string) {
	id, err := createBuild(cluster, app)
	fmt.Printf("err %+v\n", err)

	ami := fmt.Sprintf("%s-%s", cluster, app)

	base, err := ioutil.TempDir("", "build")
	fmt.Printf("err %+v\n", err)

	env := filepath.Join(base, ".env")

	err = ioutil.WriteFile(env, []byte(awsEnvironment()), 0400)
	fmt.Printf("err %+v\n", err)

	fmt.Printf("repo %+v\n", repo)
	fmt.Printf("ami %+v\n", ami)

	cmd := exec.Command("docker", "run", "--env-file", env, "convox/builder", repo, ami)
	stdout, err := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	fmt.Printf("err %+v\n", err)

	manifest := ""
	logs := ""

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ",", 6)

		if len(parts) < 5 {
			fmt.Println(scanner.Text())
			continue
		}

		if parts[2] == "manifest" {
			manifest += parts[4] + "\n"
		}

		if parts[2] == "ui" && parts[3] == "say" {
			fmt.Printf("system: %s\n", parts[4])
		}

		if parts[2] == "ui" && parts[3] == "message" {
			line := parts[4]
			message := strings.Replace(strings.SplitN(line, ": ", 2)[1], "%!(PACKER_COMMA)", ",", -1)
			logs += fmt.Sprintf("%s\n", message)
			fmt.Printf("message: %s\n", message)
		}

		if parts[1] == "amazon-ebs" && parts[2] == "artifact" && parts[3] == "0" && parts[4] == "id" {
			ami := strings.Split(parts[5], ":")[1]
			release, err := createRelease(cluster, app, ami, manifest)
			fmt.Printf("release %+v\n", release)
			fmt.Printf("err %+v\n", err)

			err = updateBuild(cluster, app, id, release, logs)
			fmt.Printf("err %+v\n", err)
		}
	}

	err = cmd.Wait()
	fmt.Printf("err %+v\n", err)

	// err = createRelease(cluster, app, "ami-foo", "example-manifest")
	// fmt.Printf("err1 %+v\n", err)

	// err = updateBuild(cluster, app, id, "ami-foo", "example-logs")
	// fmt.Printf("err2 %+v\n", err)
}

func createBuild(cluster, app string) (string, error) {
	id := generateId("B", 9)
	created := time.Now().Format(SortableTime)

	build := []dynamodb.Attribute{
		*dynamodb.NewStringAttribute("app", app),
		*dynamodb.NewStringAttribute("created", created),
		*dynamodb.NewStringAttribute("id", id),
		*dynamodb.NewStringAttribute("status", "building"),
	}

	_, err := buildsTable(cluster, app).PutItem(id, "", build)

	if err != nil {
		return "", err
	}

	return id, nil
}

func updateBuild(cluster, app, id, release, logs string) error {
	dbuild, err := buildsTable(cluster, app).GetItem(&dynamodb.Key{HashKey: id})

	fmt.Printf("erra %+v\n", err)

	if err != nil {
		return err
	}

	build := []dynamodb.Attribute{}

	for key, attr := range dbuild {
		build = append(build, *dynamodb.NewStringAttribute(key, attr.Value))
	}

	ended := time.Now().Format(SortableTime)

	build = append(build, *dynamodb.NewStringAttribute("ended", ended))
	build = append(build, *dynamodb.NewStringAttribute("status", "complete"))
	build = append(build, *dynamodb.NewStringAttribute("release", release))
	build = append(build, *dynamodb.NewStringAttribute("logs", logs))

	_, err = buildsTable(cluster, app).PutItem(id, "", build)

	return err
}

func createRelease(cluster, app, ami, manifest string) (string, error) {
	dapp, err := appsTable(cluster).GetItem(&dynamodb.Key{HashKey: app})

	if err != nil {
		return "", err
	}

	id := generateId("R", 9)
	created := time.Now().Format(SortableTime)

	release := []dynamodb.Attribute{
		*dynamodb.NewStringAttribute("app", app),
		*dynamodb.NewStringAttribute("created", created),
		*dynamodb.NewStringAttribute("id", id),
		*dynamodb.NewStringAttribute("ami", ami),
		*dynamodb.NewStringAttribute("manifest", manifest),
		*dynamodb.NewStringAttribute("env", coalesce(dapp["env"], "{}")),
	}

	_, err = releasesTable(cluster, app).PutItem(id, "", release)

	return id, err
}

func coalesce(att *dynamodb.Attribute, def string) string {
	if att != nil {
		return att.Value
	} else {
		return def
	}
}

func appsTable(cluster string) *dynamodb.Table {
	pk := dynamodb.PrimaryKey{dynamodb.NewStringAttribute("name", ""), nil}
	table := DynamoDB.NewTable(fmt.Sprintf("%s-apps", cluster), pk)
	return table
}

func buildsTable(cluster, app string) *dynamodb.Table {
	pk := dynamodb.PrimaryKey{dynamodb.NewStringAttribute("id", ""), nil}
	table := DynamoDB.NewTable(fmt.Sprintf("%s-%s-builds", cluster, app), pk)
	return table
}

func releasesTable(cluster, app string) *dynamodb.Table {
	pk := dynamodb.PrimaryKey{dynamodb.NewStringAttribute("id", ""), nil}
	table := DynamoDB.NewTable(fmt.Sprintf("%s-%s-releases", cluster, app), pk)
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
