package aws

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) AppCancel(name string) error {
	_, err := p.cloudformation().CancelUpdateStack(&cloudformation.CancelUpdateStackInput{
		StackName: aws.String(p.rackStack(name)),
	})
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) AppCreate(name string, opts structs.AppCreateOptions) (*structs.App, error) {
	switch generation(opts.Generation) {
	case "1":
		return p.appCreateGeneration1(name)
	case "2":
	default:
		return nil, fmt.Errorf("unknown generation")
	}

	data, err := formationTemplate("app", nil)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"LogBucket": p.LogBucket,
		"Rack":      p.Rack,
	}

	tags := map[string]string{
		"Generation": "2",
		"System":     "convox",
		"Rack":       p.Rack,
		"Version":    p.Version,
		"Type":       "app",
		"Name":       name,
	}

	if err := p.createStack(p.rackStack(name), data, params, tags); err != nil {
		if awsError(err) == "AlreadyExistsException" {
			return nil, fmt.Errorf("app already exists: %s", name)
		}
		return nil, err
	}

	p.EventSend("app:create", structs.EventSendOptions{Data: map[string]string{"name": name}})

	return p.AppGet(name)
}

func (p *Provider) appCreateGeneration1(name string) (*structs.App, error) {
	data, err := formationTemplate("g1/app", map[string]interface{}{"Version": p.Version})
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"LogBucket":      p.LogBucket,
		"Private":        "No",
		"Rack":           p.Rack,
		"Subnets":        p.Subnets,
		"SubnetsPrivate": coalesces(p.SubnetsPrivate, p.Subnets),
	}

	if p.Private {
		params["Private"] = "Yes"
	}

	tags := map[string]string{
		"Generation": "1",
		"System":     "convox",
		"Rack":       p.Rack,
		"Version":    p.Version,
		"Type":       "app",
		"Name":       name,
	}

	if err := p.createStack(p.rackStack(name), data, params, tags); err != nil {
		if awsError(err) == "AlreadyExistsException" {
			return nil, fmt.Errorf("app already exists: %s", name)
		}
		return nil, err
	}

	p.EventSend("app:create", structs.EventSendOptions{Data: map[string]string{"name": name}})

	return p.AppGet(name)
}

// AppGet gets an app
func (p *Provider) AppGet(name string) (*structs.App, error) {
	stacks, err := p.describeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(p.Rack + "-" + name),
	})
	if ae, ok := err.(awserr.Error); ok && ae.Code() == "ValidationError" {
		return nil, errorNotFound(fmt.Sprintf("app not found: %s", name))
	}
	if err != nil {
		return nil, err
	}
	if len(stacks) != 1 {
		return nil, fmt.Errorf("could not load stack for app: %s", name)
	}

	app, err := p.appFromStack(stacks[0])
	if ae, ok := err.(awserr.Error); ok && ae.Code() == "ValidationError" {
		return nil, errorNotFound(fmt.Sprintf("app not found: %s", name))
	}
	if err != nil {
		return nil, err
	}

	if app.Tags["Rack"] != "" && app.Tags["Rack"] != p.Rack {
		return nil, errorNotFound(fmt.Sprintf("%s not found", name))
	}

	return app, nil
}

// AppDelete deletes an app
func (p *Provider) AppDelete(name string) error {
	app, err := p.AppGet(name)
	if err != nil {
		return err
	}

	if app.Tags["Type"] != "app" || app.Tags["System"] != "convox" || app.Tags["Rack"] != p.Rack {
		return fmt.Errorf("invalid app: %s", name)
	}

	resources, err := p.ResourceList()
	if err != nil {
		return err
	}

	for _, s := range resources {
		apps, err := p.resourceApps(s)
		if err != nil {
			return err
		}

		for _, a := range apps {
			if a.Name == name {
				return fmt.Errorf("app is linked to %s resource", s.Name)
			}
		}
	}

	_, err = p.cloudformation().DeleteStack(&cloudformation.DeleteStackInput{StackName: aws.String(p.rackStack(app.Name))})
	if err != nil {
		helpers.TrackEvent("kernel-app-delete-error", nil)
		return err
	}

	go p.cleanup(app)

	return nil
}

func (p *Provider) AppList() (structs.Apps, error) {
	log := p.logger("AppList")

	stacks, err := p.describeStacks(&cloudformation.DescribeStacksInput{})
	if err != nil {
		return nil, log.Error(err)
	}

	apps := make(structs.Apps, 0)

	for _, stack := range stacks {
		tags := stackTags(stack)

		if tags["System"] == "convox" && tags["Type"] == "app" && tags["Rack"] == p.Rack {
			a, err := p.appFromStack(stack)
			if err != nil {
				return nil, err
			}

			apps = append(apps, *a)
		}
	}

	return apps, log.Success()
}

func (p *Provider) AppLogs(app string, opts structs.LogsOptions) (io.ReadCloser, error) {
	group, err := p.appResource(app, "LogGroup")
	if err != nil {
		return nil, err
	}

	return p.subscribeLogs(group, opts)
}

func (p *Provider) AppMetrics(name string, opts structs.MetricsOptions) (structs.Metrics, error) {
	metrics := map[string]bool{}

	if opts.Metrics != nil {
		for _, m := range opts.Metrics {
			metrics[m] = true
		}
	}

	mds, err := p.appMetricDefinitions(name)
	if err != nil {
		return nil, err
	}

	mms := structs.Metrics{}

	for _, md := range mds {
		if len(metrics) > 0 && !metrics[md.Name] {
			continue
		}

		m, err := p.cloudwatchMetric(md, opts)
		if err != nil {
			return nil, err
		}

		existing := false

		for i, mm := range mms {
			if mm.Name == m.Name {
				existing = true

				for j := range mm.Values {
					mms[i].Values[j].Average += mm.Values[j].Average
					mms[i].Values[j].Count += mm.Values[j].Count
					mms[i].Values[j].Maximum += mm.Values[j].Maximum
					mms[i].Values[j].Minimum += mm.Values[j].Minimum
					mms[i].Values[j].Sum += mm.Values[j].Sum
				}

				break
			}
		}

		if !existing {
			mms = append(mms, *m)
		}
	}

	return mms, nil
}

func (p *Provider) AppUpdate(app string, opts structs.AppUpdateOptions) error {
	params := opts.Parameters

	if params == nil {
		params = map[string]string{}
	}

	if opts.Lock != nil {
		_, err := p.cloudformation().UpdateTerminationProtection(&cloudformation.UpdateTerminationProtectionInput{
			EnableTerminationProtection: opts.Lock,
			StackName:                   aws.String(p.rackStack(app)),
		})
		if err != nil {
			return err
		}
	}

	if opts.Sleep != nil {
		return fmt.Errorf("sleeping not yet supported on aws racks")
	}

	return p.updateStack(p.rackStack(app), nil, opts.Parameters, map[string]string{})
}

func (p *Provider) appFromStack(stack *cloudformation.Stack) (*structs.App, error) {
	name := *stack.StackName
	tags := stackTags(stack)
	if value, ok := tags["Name"]; ok {
		// StackName probably includes the Rack prefix, prefer Name tag.
		name = value
	}

	a := &structs.App{
		Name:       name,
		Generation: coalesces(stackTags(stack)["Generation"], "1"),
		Locked:     cb(stack.EnableTerminationProtection, false),
		Release:    coalesces(stackOutputs(stack)["Release"], stackParameters(stack)["Release"]),
		Status:     humanStatus(*stack.StackStatus),
		Outputs:    stackOutputs(stack),
		Parameters: stackParameters(stack),
		Tags:       stackTags(stack),
	}

	return a, nil
}

// appRepository defines an image repository for an App
type appRepository struct {
	ID   string
	Name string
	URI  string
}

// appRepository gets an app's repository data
func (p *Provider) appRepository(name string) (*appRepository, error) {
	app, err := p.AppGet(name)
	if err != nil {
		return nil, err
	}

	if app.Tags["Generation"] == "2" {
		return p.appRepository2(name)
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
			ID:   *resp.Repositories[0].RegistryId,
			Name: repoName,
			URI:  *resp.Repositories[0].RepositoryUri,
		}, nil
	}

	return nil, fmt.Errorf("no repo found")
}

func (p *Provider) appRepository2(app string) (*appRepository, error) {
	reg, err := p.appResource(app, "Registry")
	if err != nil {
		return nil, err
	}

	aid, err := p.accountId()
	if err != nil {
		return nil, err
	}

	repo := &appRepository{
		ID:   aid,
		Name: reg,
		URI:  fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s", aid, p.Region, reg),
	}

	return repo, nil
}

// cleanup deletes AWS resources that aren't handled by the CloudFormation during stack deletion.
func (p *Provider) cleanup(app *structs.App) error {
	settings, err := p.appResource(app.Name, "Settings")
	if err != nil {
		return err
	}

	if err := p.deleteBucket(settings); err != nil {
		return err
	}

	err = p.buildsDeleteAll(app)
	if err != nil {
		fmt.Printf("fn=cleanup level=error msg=\"%s\"", err)
		return err
	}

	reg, err := p.appResource(app.Name, "Registry")
	if err != nil {
		// handle generation 1
		if strings.HasPrefix(err.Error(), "resource not found") {
			app, err := p.AppGet(app.Name)
			if err != nil {
				return err
			}

			reg = app.Outputs["RegistryRepository"]
		} else {
			return err
		}
	}

	_, err = p.ecr().DeleteRepository(&ecr.DeleteRepositoryInput{
		RepositoryName: aws.String(reg),
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
			StackName: aws.String(p.rackStack(app.Name)),
		})

		// return when stack is not found indicating successful delete
		if ae, ok := err.(awserr.Error); ok {
			if ae.Code() == "ValidationError" { // Error indicates stack wasn't found, hence deleted.
				helpers.TrackEvent("kernel-app-delete-success", nil)
				// Last ditch effort to remove the empty bucket CF leaves behind.
				_, err := p.s3().DeleteBucket(&s3.DeleteBucketInput{Bucket: aws.String(settings)})
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

				_, err := p.cloudformation().DeleteStack(&cloudformation.DeleteStackInput{StackName: aws.String(p.rackStack(app.Name))})
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
func (p *Provider) deleteBucket(bucket string) error {
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

func (p *Provider) cleanupBucketObject(bucket, key, version string) {
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
