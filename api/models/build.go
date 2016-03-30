package models

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/api/provider"
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

// Test if another build container is running.
// This is a temporary workaround since the current Docker Registry does not
// handle pushing multiple images at the same time.
// Course grained locking will prevent subtle build errors until a better
// registry and/or Docker image subsystem is integrated
func (b *Build) IsRunning() bool {
	out, err := exec.Command("docker", "ps", "-q", "--filter", "name=build-B*").CombinedOutput()

	// log exec errors but optimistically consider builds unlocked
	if err != nil {
		fmt.Printf("ns=kernel cn=build at=IsRunning state=error step=exec.Command app=%q build=%q error=%q\n", b.App, b.Id, err)
		return true
	}

	// There are active build-* containers if `docker ps -q` returns a container id, e.g. "930b96f8f3dc\n"
	running := string(out) != ""

	fmt.Printf("ns=kernel cn=build at=IsRunning running=%s step=exec.Command app=%q build=%q\n", running, b.App, b.Id)
	return running
}

func (b *Build) Cleanup() error {
	return nil
}

func (b *Build) buildError(err error, ch chan error) {
	NotifyError("build:create", err, map[string]string{"id": b.Id, "app": b.App})
	fmt.Printf("ns=kernel cn=build state=error app=%q build=%q error=%q\n", b.App, b.Id, err)
	b.Fail(err)
	ch <- err
}

func (b *Build) copyError(err error) {
	NotifyError("build:copy", err, map[string]string{"id": b.Id, "app": b.App})
	b.Fail(err)
}

func (b *Build) buildArgs(cache bool, manifest string) ([]string, error) {
	app, err := GetApp(b.App)

	if err != nil {
		return nil, err
	}

	args := []string{"run", "-i", "--name", fmt.Sprintf("build-%s", b.Id), "-v", "/var/run/docker.sock:/var/run/docker.sock", os.Getenv("DOCKER_IMAGE_API"), "build", "-id", b.Id}

	endpoint, err := AppDockerLogin(*app)

	if err != nil {
		return nil, err
	}

	args = append(args, "-push", endpoint)

	if repository := app.Outputs["RegistryRepository"]; repository != "" {
		args = append(args, "-flatten", repository)
	}

	if manifest != "" {
		args = append(args, "-manifest", manifest)
	}

	if !cache {
		args = append(args, "-no-cache")
	}

	err = LoginPrivateRegistries()

	if err != nil {
		return nil, err
	}

	if dockercfg, err := ioutil.ReadFile("/root/.docker/config.json"); err == nil {
		args = append(args, "-dockercfg", string(dockercfg))
	}

	return args, nil
}

func (b *Build) ExecuteLocal(r io.Reader, cache bool, manifest string, ch chan error) {
	started := time.Now()

	b.Status = "building"
	err := b.Save()

	if err != nil {
		helpers.TrackError("build", err, map[string]interface{}{"type": "local", "at": "b.Save"})
		b.buildError(err, ch)
		return
	}

	args, err := b.buildArgs(cache, manifest)

	if err != nil {
		helpers.TrackError("build", err, map[string]interface{}{"type": "local", "at": "b.buildArgs"})
		b.buildError(err, ch)
		return
	}

	args = append(args, b.App, "-")

	err = b.execute(args, r, ch)

	if err != nil {
		helpers.TrackError("build", err, map[string]interface{}{"type": "local", "at": "b.execute"})
		b.buildError(err, ch)
		return
	}

	NotifySuccess("build:create", map[string]string{"id": b.Id, "app": b.App})
	fmt.Printf("ns=kernel cn=build at=ExecuteLocal state=success step=build.execute app=%q build=%q\n", b.App, b.Id)
	helpers.TrackSuccess("build", map[string]interface{}{"type": "local", "elapsed": time.Now().Sub(started).Nanoseconds() / 1000000})
}

func (b *Build) ExecuteRemote(repo string, cache bool, manifest string, ch chan error) {
	started := time.Now()

	b.Status = "building"
	err := b.Save()

	if err != nil {
		helpers.TrackError("build", err, map[string]interface{}{"type": "remote", "at": "b.Save"})
		b.buildError(err, ch)
		return
	}

	args, err := b.buildArgs(cache, manifest)

	if err != nil {
		helpers.TrackError("build", err, map[string]interface{}{"type": "remote", "at": "b.buildArgs"})
		b.buildError(err, ch)
		return
	}

	args = append(args, b.App, repo)

	err = b.execute(args, nil, ch)

	if err != nil {
		helpers.TrackError("build", err, map[string]interface{}{"type": "remote", "at": "b.execute"})
		b.buildError(err, ch)
		return
	}

	NotifySuccess("build:create", map[string]string{"id": b.Id, "app": b.App})
	fmt.Printf("ns=kernel cn=build at=ExecuteRemote state=success step=build.execute app=%q build=%q\n", b.App, b.Id)
	helpers.TrackSuccess("build", map[string]interface{}{"type": "remote", "elapsed": time.Now().Sub(started).Nanoseconds() / 1000000})
}

func (b *Build) ExecuteIndex(index Index, cache bool, manifest string, ch chan error) {
	started := time.Now()

	b.Status = "building"
	err := b.Save()

	if err != nil {
		helpers.TrackError("build", err, map[string]interface{}{"type": "remote", "at": "b.Save"})
		b.buildError(err, ch)
		return
	}

	args, err := b.buildArgs(cache, manifest)

	if err != nil {
		helpers.TrackError("build", err, map[string]interface{}{"type": "index", "at": "b.buildArgs"})
		b.buildError(err, ch)
		return
	}

	dir, err := ioutil.TempDir("", "source")

	if err != nil {
		helpers.TrackError("build", err, map[string]interface{}{"type": "index", "at": "ioutil.TempDir"})
		b.buildError(err, ch)
		return
	}

	err = os.Chmod(dir, 0755)

	if err != nil {
		helpers.TrackError("build", err, map[string]interface{}{"type": "index", "at": "os.Chmod"})
		b.buildError(err, ch)
		return
	}

	err = index.Download(dir)

	if err != nil {
		helpers.TrackError("build", err, map[string]interface{}{"type": "index", "at": "index.Download"})
		b.buildError(err, ch)
		return
	}

	tgz, err := createTarball(dir)

	if err != nil {
		helpers.TrackError("build", err, map[string]interface{}{"type": "index", "at": "createTarball"})
		b.buildError(err, ch)
		return
	}

	args = append(args, b.App, "-")

	err = b.execute(args, bytes.NewReader(tgz), ch)

	if err != nil {
		helpers.TrackError("build", err, map[string]interface{}{"type": "index", "at": "b.execute"})
		b.buildError(err, ch)
		return
	}

	NotifySuccess("build:create", map[string]string{"id": b.Id, "app": b.App})
	fmt.Printf("ns=kernel cn=build at=ExecuteIndex state=success step=build.execute app=%q build=%q\n", b.App, b.Id)
	helpers.TrackSuccess("build", map[string]interface{}{"type": "index", "elapsed": time.Now().Sub(started).Nanoseconds() / 1000000})
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

func (b *Build) Delete() error {
	// delete ECR images
	// delete dynamo record for build?
	// delete release records for build?
	return fmt.Errorf("Can not delete active build")
}

// Images returns a list of fully qualified URLs for images for every process type
// in the build manifest. These may point to the convox-hosted registry or ECR, e.g.
// {convox-826133048.us-east-1.elb.amazonaws.com:5000/myapp-web:BSUSBFCUCSA} or
// {826133048.dkr.ecr.us-east-1.amazonaws.com/myapp-zridvyqapp:web.BSUSBFCUCSA} respectively.
func (b *Build) Images() ([]string, error) {
	app, err := provider.AppGet(b.App)

	if err != nil {
		return nil, err
	}

	var entries ManifestEntries

	err = yaml.Unmarshal([]byte(b.Manifest), &entries)

	if err != nil {
		return nil, err
	}

	imgs := []string{}

	for name, _ := range entries {
		img := fmt.Sprintf("%s/%s-%s:%s", os.Getenv("REGISTRY_HOST"), app.Name, name, b.Id)

		if registryId := app.Outputs["RegistryId"]; registryId != "" {
			img = fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s.%s", registryId, os.Getenv("AWS_REGION"), app.Outputs["RegistryRepository"], name, b.Id)
		}

		imgs = append(imgs, img)
	}

	return imgs, nil
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

func createTarball(base string) ([]byte, error) {
	cwd, err := os.Getwd()

	if err != nil {
		return nil, err
	}

	err = os.Chdir(base)

	if err != nil {
		return nil, err
	}

	args := []string{"cz"}

	// If .dockerignore exists, use it to exclude files from the tarball
	if _, err = os.Stat(".dockerignore"); err == nil {
		args = append(args, "--exclude-from", ".dockerignore")
	}

	args = append(args, ".")

	cmd := exec.Command("tar", args...)

	out, err := cmd.StdoutPipe()

	if err != nil {
		return nil, err
	}

	cmd.Start()

	bytes, err := ioutil.ReadAll(out)

	if err != nil {
		return nil, err
	}

	err = cmd.Wait()

	if err != nil {
		return nil, err
	}

	err = os.Chdir(cwd)

	if err != nil {
		return nil, err
	}

	return bytes, nil
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
