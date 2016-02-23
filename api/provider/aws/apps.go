package aws

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/api/structs"
)

var (
	ValidAppName = regexp.MustCompile(`\A[a-zA-Z][-a-zA-Z0-9]{3,29}\z`)
)

func (p *AWSProvider) AppList() (structs.Apps, error) {
	res, err := p.CachedDescribeStacks(nil)

	if err != nil {
		return nil, err
	}

	apps := make(structs.Apps, 0)

	for _, stack := range res.Stacks {
		tags := stackTags(stack)

		if tags["System"] == "convox" && tags["Type"] == "app" {
			if tags["Rack"] == "" || tags["Rack"] == os.Getenv("RACK") {
				apps = append(apps, appFromStack(stack))
			}
		}
	}

	return apps, nil
}

func (p *AWSProvider) AppGet(name string) (*structs.App, error) {
	res, err := p.CachedDescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(name),
	})

	if awsError(err) == "ValidationError" {
		return nil, fmt.Errorf("no such app: %s", name)
	}

	if err != nil {
		return nil, err
	}

	if len(res.Stacks) != 1 {
		return nil, fmt.Errorf("could not load stack for app: %s", name)
	}

	tags := stackTags(res.Stacks[0])

	if name != os.Getenv("RACK") && (tags["System"] != "convox" || tags["Type"] != "app" || tags["Rack"] != os.Getenv("RACK")) {
		return nil, fmt.Errorf("no such app: %s", name)
	}

	app := appFromStack(res.Stacks[0])

	return &app, nil
}

func (p *AWSProvider) AppCreate(name string) error {
	if !ValidAppName.MatchString(name) {
		return fmt.Errorf("app name can contain only alphanumeric characters and dashes and must be between 4 and 30 characters")
	}

	app := structs.App{
		Name: name,
	}

	formation, err := appFormation(app)

	if err != nil {
		return err
	}

	params := map[string]string{
		"Cluster":        os.Getenv("CLUSTER"),
		"Private":        os.Getenv("PRIVATE"),
		"Subnets":        os.Getenv("SUBNETS"),
		"SubnetsPrivate": os.Getenv("SUBNETS"),
		"Version":        os.Getenv("RELEASE"),
		"VPC":            os.Getenv("VPC"),
	}

	if v := os.Getenv("ENCRYPTION_KEY"); v != "" {
		params["Key"] = v
	}

	if v := os.Getenv("SUBNETS_PRIVATE"); v != "" {
		params["SubnetsPrivate"] = v
	}

	tags := map[string]string{
		"Rack":   os.Getenv("RACK"),
		"System": "convox",
		"Type":   "app",
	}

	// FIXME upload formation to s3 and use TemplateUrl

	req := &cloudformation.CreateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(app.Name),
		TemplateBody: aws.String(formation),
	}

	for key, value := range params {
		req.Parameters = append(req.Parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String(key),
			ParameterValue: aws.String(value),
		})
	}

	for key, value := range tags {
		req.Tags = append(req.Tags, &cloudformation.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	_, err = p.cloudformation().CreateStack(req)

	if err != nil {
		return err
	}

	p.NotifySuccess("app:create", map[string]string{
		"name": app.Name,
	})

	return nil
}

func (p *AWSProvider) AppDelete(app *structs.App) error {
	_, err := p.cloudformation().DeleteStack(&cloudformation.DeleteStackInput{
		StackName: aws.String(app.Name),
	})

	if err != nil {
		return err
	}

	go p.appCleanup(app)

	p.NotifySuccess("app:delete", map[string]string{
		"name": app.Name,
	})

	return nil
}

/** helpers ****************************************************************************************/

func (p *AWSProvider) appCleanup(app *structs.App) error {
	err := p.cleanupBucket(app.Outputs["Settings"])

	if err != nil {
		return err
	}

	// FIXME: once builds are moved over

	// builds, err := ListBuilds(a.Name)

	// if err != nil {
	//   return err
	// }

	// for _, build := range builds {
	//   go cleanupBuild(build)
	// }

	releases, err := p.ReleaseList(app.Name)

	if err != nil {
		return err
	}

	for _, release := range releases {
		go p.releaseCleanup(release)
	}

	// monitor and stack deletion state for up to 10 minutes
	// retry once if DELETE_FAILED to automate around transient errors
	// send delete success event only when stack is gone
	for i := 0; i < 60; i++ {
		time.Sleep(10 * time.Second)

		res, err := p.cloudformation().DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: aws.String(app.Name),
		})

		// return when stack is not found indicating successful delete
		if awsError(err) == "ValidationError" {
			return nil
		}

		if err != nil {
			continue
		}

		if len(res.Stacks) < 1 {
			return nil
		}

		if *res.Stacks[0].StackStatus == "DELETE_FAILED" {
			_, err := p.cloudformation().DeleteStack(&cloudformation.DeleteStackInput{
				StackName: aws.String(app.Name),
			})

			if err == nil {
				break
			}
		}
	}

	return nil
}

func appFormation(app structs.App) (string, error) {
	data, err := templateLoad("app", "app", nil)

	if err != nil {
		return "", err
	}

	return string(data), nil
}

func appFromStack(stack *cloudformation.Stack) structs.App {
	return structs.App{
		Name:       *stack.StackName,
		Release:    stackParameters(stack)["Release"],
		Status:     humanStatus(*stack.StackStatus),
		Outputs:    stackOutputs(stack),
		Parameters: stackParameters(stack),
		Tags:       stackTags(stack),
	}
}

func (p *AWSProvider) appLatestRelease(app string) (*structs.Release, error) {
	releases, err := p.ReleaseList(app)

	if err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return nil, nil
	}

	return &releases[0], nil
}

func (p *AWSProvider) appServices(app string) ([]*ecs.Service, error) {
	services := []*ecs.Service{}

	resources, err := p.stackResources(app)

	if err != nil {
		return nil, err
	}

	arns := []*string{}

	i := 0
	for _, r := range resources {
		i = i + 1

		if r.Type == "Custom::ECSService" {
			arns = append(arns, aws.String(r.Id))
		}

		//have to make requests in batches of ten
		if len(arns) == 10 || (i == len(resources) && len(arns) > 0) {
			dres, err := p.ecs().DescribeServices(&ecs.DescribeServicesInput{
				Cluster:  aws.String(os.Getenv("CLUSTER")),
				Services: arns,
			})

			if err != nil {
				return nil, err
			}

			services = append(services, dres.Services...)
			arns = []*string{}
		}
	}

	return services, nil
}

func (p *AWSProvider) appTasks(app string) ([]*ecs.Task, error) {
	tasks := []*ecs.Task{}

	_, err := p.AppGet(app)

	if err != nil {
		return nil, err
	}

	resources, err := p.stackResources(app)

	if err != nil {
		return nil, err
	}

	services := []string{}

	for _, resource := range resources {
		if resource.Type == "Custom::ECSService" {
			parts := strings.Split(resource.Id, "/")
			services = append(services, parts[len(parts)-1])
		}
	}

	for _, service := range services {
		taskArns, err := p.ecs().ListTasks(&ecs.ListTasksInput{
			Cluster:     aws.String(os.Getenv("CLUSTER")),
			ServiceName: aws.String(service),
		})

		if err != nil {
			return nil, err
		}

		if len(taskArns.TaskArns) == 0 {
			continue
		}

		tres, err := p.ecs().DescribeTasks(&ecs.DescribeTasksInput{
			Cluster: aws.String(os.Getenv("CLUSTER")),
			Tasks:   taskArns.TaskArns,
		})

		if err != nil {
			return nil, err
		}

		tasks = append(tasks, tres.Tasks...)
	}

	return tasks, nil
}
