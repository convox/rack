package aws

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecr"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/s3"
	"github.com/convox/rack/Godeps/_workspace/src/gopkg.in/yaml.v2"

	"github.com/convox/rack/api/structs"
)

type ManifestEntries map[string]interface{}

var regexpECR = regexp.MustCompile(`(\d+)\.dkr\.ecr\.([^.]+)\.amazonaws\.com\/([^:]+):([^ ]+)`)

func buildsTable(app string) string {
	return os.Getenv("DYNAMO_BUILDS")
}

func (p *AWSProvider) BuildCreateTar(app string, src io.Reader, manifest, description string, cache bool) (*structs.Build, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	b := structs.NewBuild(app)
	b.Description = description
	err = p.BuildSave(b, "")

	// save the tarball in s3?
	// TODO: retry pushes w/ backoff

	args := p.buildArgs(a, b, "-")

	env, err := p.buildEnv(a, b, manifest, cache)
	if err != nil {
		return b, err
	}

	cmd := exec.Command("docker", args...)
	cmd.Env = env
	cmd.Stdin = src

	// build create is now complete; background waiting for command to finish
	// and saving command stdout/stderr logs and exit status
	go func() {
		out, err := cmd.CombinedOutput()

		// reload build item to get data from BuildUpdate callback
		b, berr := p.BuildGet(app, b.Id)
		if berr != nil {
			fmt.Printf("TODO ROLLBAR: %+v\n", berr)
			return
		}

		b.Logs = string(out)

		if err != nil {
			b.Status = "failed"
		}

		err = p.BuildSave(b, a.Outputs["Settings"]) // PUT logs in S3
		if err != nil {
			fmt.Printf("TODO ROLLBAR: %+v\n", err)
			return
		}
	}()

	return b, err
}

func (p *AWSProvider) BuildGet(app, id string) (*structs.Build, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	req := &dynamodb.GetItemInput{
		ConsistentRead: aws.Bool(true),
		Key: map[string]*dynamodb.AttributeValue{
			"id": &dynamodb.AttributeValue{S: aws.String(id)},
		},
		TableName: aws.String(buildsTable(app)),
	}

	res, err := p.dynamodb().GetItem(req)
	if err != nil {
		return nil, err
	}

	if res.Item == nil {
		return nil, fmt.Errorf("no such build: %s", id)
	}

	build := p.buildFromItem(res.Item, a.Outputs["Settings"])

	return build, nil
}

func (p *AWSProvider) BuildDelete(app, id string) (*structs.Build, error) {
	b, err := p.BuildGet(app, id)
	if err != nil {
		return b, err
	}

	a, err := p.AppGet(app)
	if err != nil {
		return b, err
	}

	// scan dynamo for all releases for this build
	res, err := p.dynamodb().Query(&dynamodb.QueryInput{
		KeyConditionExpression: aws.String("app = :app"),
		FilterExpression:       aws.String("build = :build"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":app":   &dynamodb.AttributeValue{S: aws.String(app)},
			":build": &dynamodb.AttributeValue{S: aws.String(id)},
		},
		IndexName: aws.String("app.created"),
		TableName: aws.String(releasesTable(app)),
	})

	if err != nil {
		return nil, err
	}

	// collect release IDs to delete
	// and validate the build doesn't belong to the app's current release
	wrs := []*dynamodb.WriteRequest{}
	for _, item := range res.Items {
		r := releaseFromItem(item)

		if a.Release == r.Id {
			return b, errors.New("cant delete build contained in active release")
		}

		wr := &dynamodb.WriteRequest{
			DeleteRequest: &dynamodb.DeleteRequest{
				Key: map[string]*dynamodb.AttributeValue{
					"id": &dynamodb.AttributeValue{
						S: aws.String(r.Id),
					},
				},
			},
		}

		wrs = append(wrs, wr)
	}

	// delete all release items
	// TODO: Move to ReleaseDelete and also clean up env, task definition, etc.
	p.dynamodb().BatchWriteItem(&dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			releasesTable(app): wrs,
		},
	})

	// delete build item
	p.dynamodb().DeleteItem(&dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"id": &dynamodb.AttributeValue{S: aws.String(id)},
		},
		TableName: aws.String(buildsTable(app)),
	})

	// delete ECR images
	err = p.deleteImages(a, b)
	if err != nil {
		return b, err
	}

	return b, nil
}

func (p *AWSProvider) BuildList(app string) (structs.Builds, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

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

	res, err := p.dynamodb().Query(req)
	if err != nil {
		return nil, err
	}

	builds := make(structs.Builds, len(res.Items))

	for i, item := range res.Items {
		builds[i] = *p.buildFromItem(item, a.Outputs["Settings"])
	}

	return builds, nil
}

func (p *AWSProvider) BuildRelease(b *structs.Build) (*structs.Release, error) {
	releases, err := p.ReleaseList(b.App)
	if err != nil {
		return nil, err
	}

	r := structs.NewRelease(b.App)
	newId := r.Id

	if len(releases) > 0 {
		r = &releases[0]
	}

	r.Id = newId
	r.Created = time.Time{}
	r.Build = b.Id
	r.Manifest = b.Manifest

	a, err := p.AppGet(b.App)
	if err != nil {
		return r, err
	}

	err = p.ReleaseSave(r, a.Outputs["Settings"], a.Parameters["Key"])
	if err != nil {
		return r, err
	}

	b.Release = r.Id
	err = p.BuildSave(b, "")
	return r, err
}

// BuildSave creates or updates a build item in DynamoDB. It takes an optional
// bucket argument, which if set indicates to PUT Log data into S3
func (p *AWSProvider) BuildSave(b *structs.Build, bucket string) error {
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

	if bucket != "" {
		_, err := p.s3().PutObject(&s3.PutObjectInput{
			Body:          bytes.NewReader([]byte(b.Logs)),
			Bucket:        aws.String(bucket),
			ContentLength: aws.Int64(int64(len(b.Logs))),
			Key:           aws.String(fmt.Sprintf("builds/%s.log", b.Id)),
		})
		if err != nil {
			return err
		}
	}

	_, err := p.dynamodb().PutItem(req)

	return err
}

func (p *AWSProvider) buildArgs(a *structs.App, b *structs.Build, source string) []string {
	return []string{
		"run",
		"-i",
		"--name", fmt.Sprintf("build-%s", b.Id),
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		"-e", "APP",
		"-e", "BUILD",
		"-e", "DOCKER_AUTH",
		"-e", "RACK_HOST",
		"-e", "RACK_PASSWORD",
		"-e", "REGISTRY_EMAIL",
		"-e", "REGISTRY_USERNAME",
		"-e", "REGISTRY_PASSWORD",
		"-e", "REGISTRY_ADDRESS",
		"-e", "MANIFEST_PATH",
		"-e", "REPOSITORY",
		"-e", "NO_CACHE",
		os.Getenv("DOCKER_IMAGE_API"),
		"build2",
		source,
	}
}

func (p *AWSProvider) buildEnv(a *structs.App, b *structs.Build, manifest_path string, cache bool) ([]string, error) {
	// self-hosted registry auth
	email := "user@convox.com"
	username := "convox"
	password := os.Getenv("PASSWORD")
	address := os.Getenv("REGISTRY_HOST")

	// ECR auth
	if registryId := a.Outputs["RegistryId"]; registryId != "" {
		res, err := p.ecr().GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{
			RegistryIds: []*string{aws.String(registryId)},
		})

		if err != nil {
			return nil, err
		}

		if len(res.AuthorizationData) < 1 {
			return nil, fmt.Errorf("no authorization data")
		}

		endpoint := *res.AuthorizationData[0].ProxyEndpoint

		data, err := base64.StdEncoding.DecodeString(*res.AuthorizationData[0].AuthorizationToken)

		if err != nil {
			return nil, err
		}

		parts := strings.SplitN(string(data), ":", 2)

		password = parts[1]
		address = endpoint[8:]
		username = parts[0]
	}

	// TODO: The controller logged into private registries and app registry
	// Seems like this method should be able to generate docker auth config on its own
	dockercfg, err := ioutil.ReadFile("/root/.docker/config.json")
	if err != nil {
		return nil, err
	}

	env := []string{
		fmt.Sprintf("APP=%s", a.Name),
		fmt.Sprintf("BUILD=%s", b.Id),
		fmt.Sprintf("MANIFEST_PATH=%s", manifest_path),
		fmt.Sprintf("DOCKER_AUTH=%s", dockercfg),
		fmt.Sprintf("RACK_HOST=%s", os.Getenv("NOTIFICATION_HOST")),
		fmt.Sprintf("RACK_PASSWORD=%s", os.Getenv("PASSWORD")),
		fmt.Sprintf("REGISTRY_EMAIL=%s", email),
		fmt.Sprintf("REGISTRY_USERNAME=%s", username),
		fmt.Sprintf("REGISTRY_PASSWORD=%s", password),
		fmt.Sprintf("REGISTRY_ADDRESS=%s", address),
		fmt.Sprintf("REPOSITORY=%s", a.Outputs["RegistryRepository"]),
	}

	if cache == false {
		env = append(env, "NO_CACHE=true")
	}

	return env, nil
}

// deleteImages generates a list of fully qualified URLs for images for every process type
// in the build manifest then deletes them.
// Image URLs that point to ECR, e.g. 826133048.dkr.ecr.us-east-1.amazonaws.com/myapp-zridvyqapp:web.BSUSBFCUCSA,
// are deleted with the ECR BatchDeleteImage API.
// Image URLs that point to the convox-hosted registry, e.g. convox-826133048.us-east-1.elb.amazonaws.com:5000/myapp-web:BSUSBFCUCSA,
// are not yet supported and return an error.
func (p *AWSProvider) deleteImages(a *structs.App, b *structs.Build) error {
	var entries ManifestEntries

	err := yaml.Unmarshal([]byte(b.Manifest), &entries)

	if err != nil {
		return err
	}

	// failed builds could have an empty manifest
	if len(entries) == 0 {
		return nil
	}

	urls := []string{}

	for name, _ := range entries {
		img := fmt.Sprintf("%s/%s-%s:%s", os.Getenv("REGISTRY_HOST"), a.Name, name, b.Id)

		if registryId := a.Outputs["RegistryId"]; registryId != "" {
			img = fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s.%s", registryId, os.Getenv("AWS_REGION"), a.Outputs["RegistryRepository"], name, b.Id)
		}

		urls = append(urls, img)
	}

	imageIds := []*ecr.ImageIdentifier{}
	registryId := ""
	repositoryName := ""

	for _, url := range urls {
		if match := regexpECR.FindStringSubmatch(url); match != nil {
			registryId = match[1]
			repositoryName = match[3]

			imageIds = append(imageIds, &ecr.ImageIdentifier{
				ImageTag: aws.String(match[4]),
			})
		} else {
			return errors.New("URL not valid ECR")
		}
	}

	_, err = p.ecr().BatchDeleteImage(&ecr.BatchDeleteImageInput{
		ImageIds:       imageIds,
		RegistryId:     aws.String(registryId),
		RepositoryName: aws.String(repositoryName),
	})

	return err
}

// buildFromItem populates a Build struct from a DynamoDB Item. It also populates build.Logs
// from an S3 object if a bucket is passed in and a builds/B1234.log object exists.
func (p *AWSProvider) buildFromItem(item map[string]*dynamodb.AttributeValue, bucket string) *structs.Build {
	id := coalesce(item["id"], "")
	started, _ := time.Parse(SortableTime, coalesce(item["created"], ""))
	ended, _ := time.Parse(SortableTime, coalesce(item["ended"], ""))

	// if an app bucket was passed in, try to get logs from S3
	logs := ""

	if bucket != "" {
		key := fmt.Sprintf("builds/%s.log", id)

		req := &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}

		res, err := p.s3().GetObject(req)

		if err != nil {
			fmt.Printf("aws buildFromItem s3.GetObject bucket=%s key=%s err=%s\n", bucket, key, err)
		} else {
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				fmt.Printf("aws buildFromItem ioutil.ReadAll err=%s\n", err)
			} else {
				logs = string(body)
			}
		}
	}

	return &structs.Build{
		Id:          id,
		App:         coalesce(item["app"], ""),
		Description: coalesce(item["description"], ""),
		Logs:        logs,
		Manifest:    coalesce(item["manifest"], ""),
		Release:     coalesce(item["release"], ""),
		Status:      coalesce(item["status"], ""),
		Started:     started,
		Ended:       ended,
	}
}
