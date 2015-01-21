package builder

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/convox/kernel/builder/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/builder/controllers"

	caws "github.com/convox/kernel/builder/Godeps/_workspace/src/github.com/crowdmob/goamz/aws"
	"github.com/convox/kernel/builder/Godeps/_workspace/src/github.com/crowdmob/goamz/dynamodb"
	"github.com/convox/kernel/builder/Godeps/_workspace/src/github.com/crowdmob/goamz/ec2"

	gaws "github.com/convox/kernel/builder/Godeps/_workspace/src/github.com/goamz/goamz/aws"
	"github.com/convox/kernel/builder/Godeps/_workspace/src/github.com/goamz/goamz/cloudformation"
)

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
	fmt.Printf("err %+v\n", err)
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
			logs += fmt.Sprintf("%s\n", parts[4])
			fmt.Printf("message: %s\n", parts[4])
		}

		if parts[1] == "amazon-ebs" && parts[2] == "artifact" && parts[3] == "0" && parts[4] == "id" {
			aparts := strings.Split(parts[5], ":")
			err = createRelease(cluster, app, aparts[1], manifest, logs)
			fmt.Printf("err %+v\n", err)
		}
	}

	err = cmd.Wait()
	fmt.Printf("err %+v\n", err)
}

func createRelease(cluster, app, ami, manifest, logs string) error {
	attributes := []dynamodb.Attribute{
		*dynamodb.NewStringAttribute("manifest", manifest),
		*dynamodb.NewStringAttribute("logs", logs),
	}

	_, err := releasesTable(cluster, app).PutItem(ami, "now", attributes)

	return err
}

func releasesTable(cluster, app string) *dynamodb.Table {
	pk := dynamodb.PrimaryKey{dynamodb.NewStringAttribute("ami", ""), dynamodb.NewStringAttribute("created-at", "")}
	table := DynamoDB.NewTable(fmt.Sprintf("%s-%s-releases", cluster, app), pk)
	return table
}
