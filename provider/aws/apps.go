package aws

import (
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/api/structs"
)

// appRepository defines an image repository for an App
type appRepository struct {
	ID  string `json:"id"`
	URI string `json:"uri"`
}

// AppGet gets an app
func (p *AWSProvider) AppGet(name string) (*structs.App, error) {
	var res *cloudformation.DescribeStacksOutput
	var err error

	res, err = p.describeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(p.Rack + "-" + name),
	})
	if ae, ok := err.(awserr.Error); ok && ae.Code() == "ValidationError" {
		return nil, ErrorNotFound(fmt.Sprintf("%s not found", name))
	}
	if err != nil {
		return nil, err
	}
	if len(res.Stacks) != 1 {
		return nil, fmt.Errorf("could not load stack for app: %s", name)
	}

	app := appFromStack(res.Stacks[0])

	if app.Tags["Rack"] != "" && app.Tags["Rack"] != p.Rack {
		return nil, fmt.Errorf("no such app: %s", name)
	}

	return &app, nil
}

// AppDelete deletes an app
func (p *AWSProvider) AppDelete(name string) error {

	app, err := p.AppGet(name)
	if err != nil {
		return err
	}

	_, err = p.cloudformation().DeleteStack(&cloudformation.DeleteStackInput{StackName: aws.String(app.StackName())})
	if err != nil {
		helpers.TrackEvent("kernel-app-delete-error", nil)
		return err
	}

	go p.cleanup(app)

	return nil
}

// AppRepository gets an app's repository data
func (p *AWSProvider) appRepository(name string) (*appRepository, error) {
	app, err := p.AppGet(name)
	if err != nil {
		return nil, err
	}

	repoName := app.Outputs["RegistryRepository"]

	params := &ecr.DescribeRepositoriesInput{
		RepositoryNames: []*string{
			aws.String(repoName),
		},
	}

	resp, err := p.ecr().DescribeRepositories(params)
	if err != nil {
		return nil, err
	}

	if len(resp.Repositories) > 0 {
		return &appRepository{
			ID:  repoName,
			URI: *resp.Repositories[0].RepositoryUri,
		}, nil
	}

	return nil, fmt.Errorf("no repo found")
}

// cleanup deletes AWS resources that aren't handled by the CloudFormation during stack deletion.
func (p *AWSProvider) cleanup(app *structs.App) error {

	err := p.deleteBucket(app.Outputs["Settings"])
	if err != nil {
		fmt.Printf("fn=cleanup level=error msg=\"%s\"", err)
		return err
	}

	err = p.buildsDeleteAll(app)
	if err != nil {
		fmt.Printf("fn=cleanup level=error msg=\"%s\"", err)
		return err
	}

	_, err = p.ecr().DeleteRepository(&ecr.DeleteRepositoryInput{
		RepositoryName: aws.String(app.Outputs["RegistryRepository"]),
		Force:          aws.Bool(true),
	})
	if err != nil {
		fmt.Printf("fn=cleanup level=error msg=\"error deleting ecr repo: %s\"", err)
	}

	err = p.releaseDeleteAll(app.Name)
	if err != nil {
		fmt.Printf("fn=cleanup level=error msg=\"%s\"", err)
		return err
	}

	// monitor and stack deletion state for up to 10 minutes
	// retry once if DELETE_FAILED to automate around transient errors
	// send delete success event only when stack is gone
	shouldRetry := true

	for i := 0; i < 60; i++ {
		res, err := p.cloudformation().DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: aws.String(app.StackName()),
		})

		// return when stack is not found indicating successful delete
		if ae, ok := err.(awserr.Error); ok {
			if ae.Code() == "ValidationError" { // Error indicates stack wasn't found, hence deleted.
				helpers.TrackEvent("kernel-app-delete-success", nil)
				// Last ditch effort to remove the empty bucket CF leaves behind.
				_, err := p.s3().DeleteBucket(&s3.DeleteBucketInput{Bucket: aws.String(app.Outputs["Settings"])})
				if err != nil {
					fmt.Printf("last ditch effort bucket error: %s\n", err)
				}
				return nil
			}
		}

		if err == nil && len(res.Stacks) == 1 && shouldRetry {
			// if delete failed, issue one more delete stack and return
			s := res.Stacks[0]
			if *s.StackStatus == "DELETE_FAILED" {
				helpers.TrackEvent("kernel-app-delete-retry", nil)

				_, err := p.cloudformation().DeleteStack(&cloudformation.DeleteStackInput{StackName: aws.String(app.StackName())})

				if err != nil {
					helpers.TrackEvent("kernel-app-delete-retry-error", nil)
				} else {
					shouldRetry = false
				}
			}
		}

		time.Sleep(10 * time.Second)
	}

	return nil
}

// deleteBucket deletes all object versions and delete markers then deletes the bucket.
func (p *AWSProvider) deleteBucket(bucket string) error {
	req := &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
	}

	res, err := p.s3().ListObjectVersions(req)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)

	maxLen := 1000
	go func() {
		defer wg.Done()

		for i := 0; i < len(res.DeleteMarkers); i += maxLen {
			high := i + maxLen
			if high > len(res.DeleteMarkers) {
				high = len(res.DeleteMarkers)
			}

			objects := []*s3.ObjectIdentifier{}
			for _, obj := range res.DeleteMarkers[i:high] {
				objects = append(objects, &s3.ObjectIdentifier{Key: obj.Key, VersionId: obj.VersionId})
			}

			_, err := p.s3().DeleteObjects(&s3.DeleteObjectsInput{
				Bucket: aws.String(bucket),
				Delete: &s3.Delete{
					Objects: objects,
				},
			})
			if err != nil {
				fmt.Printf("failed to delete S3 markers: %s\n", err)
			}
		}
	}()

	go func() {
		defer wg.Done()

		for i := 0; i < len(res.Versions); i += maxLen {
			high := i + maxLen
			if high > len(res.Versions) {
				high = len(res.Versions)
			}

			objects := []*s3.ObjectIdentifier{}
			for _, obj := range res.Versions[i:high] {
				objects = append(objects, &s3.ObjectIdentifier{Key: obj.Key, VersionId: obj.VersionId})
			}

			_, err := p.s3().DeleteObjects(&s3.DeleteObjectsInput{
				Bucket: aws.String(bucket),
				Delete: &s3.Delete{
					Objects: objects,
				},
			})
			if err != nil {
				fmt.Printf("failed to delete S3 versions: %s\n", err)
			}
		}
	}()

	wg.Wait()

	_, err = p.s3().DeleteBucket(&s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return err
	}

	return nil
}

func (p *AWSProvider) cleanupBucketObject(bucket, key, version string) {
	req := &s3.DeleteObjectInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String(key),
		VersionId: aws.String(version),
	}

	_, err := p.s3().DeleteObject(req)
	if err != nil {
		fmt.Printf("error: %s\n", err)
	}
}

func appFromStack(stack *cloudformation.Stack) structs.App {
	name := *stack.StackName
	tags := stackTags(stack)
	if value, ok := tags["Name"]; ok {
		// StackName probably includes the Rack prefix, prefer Name tag.
		name = value
	}

	return structs.App{
		Name:       name,
		Release:    stackParameters(stack)["Release"],
		Status:     humanStatus(*stack.StackStatus),
		Outputs:    stackOutputs(stack),
		Parameters: stackParameters(stack),
		Tags:       stackTags(stack),
	}
}
