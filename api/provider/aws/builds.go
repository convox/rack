package aws

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
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
