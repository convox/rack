package models

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"time"

	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/client"
	"github.com/convox/rack/manifest1"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

var (
	customTopic         = os.Getenv("CUSTOM_TOPIC")
	cloudformationTopic = os.Getenv("CLOUDFORMATION_TOPIC")
	statusCodePrefix    = client.StatusCodePrefix
)

type App struct {
	Generation string `json:"generation,omitempty"`
	Name       string `json:"name"`
	Release    string `json:"release"`
	Status     string `json:"status"`

	Outputs    map[string]string `json:"-"`
	Parameters map[string]string `json:"-"`
	Tags       map[string]string `json:"-"`
}

type Apps []App

func ListApps() (Apps, error) {
	res, err := DescribeStacks()
	if err != nil {
		return nil, err
	}

	apps := make(Apps, 0)

	for _, stack := range res.Stacks {
		tags := stackTags(stack)

		if tags["System"] == "convox" && tags["Type"] == "app" {
			if tags["Rack"] == "" || tags["Rack"] == os.Getenv("RACK") {
				apps = append(apps, *appFromStack(stack))
			}
		}
	}

	return apps, nil
}

// Deprecated: Provider.AppGet() should be used instead
func GetApp(name string) (*App, error) {
	stackName := shortNameToStackName(name)
	app, err := getAppByStackName(stackName)

	if name != stackName && awsError(err) == "ValidationError" {
		// Only lookup an unbound app if the name/stackName differ and the
		// bound lookup fails.
		app, err = getAppByStackName(name)
	}

	if app != nil {
		if app.Tags["Rack"] != "" && app.Tags["Rack"] != os.Getenv("RACK") {
			return nil, fmt.Errorf("no such app: %s", name)

		} else if len(app.Tags) == 0 && name != os.Getenv("RACK") {
			// This checks for a rack. An app with zero tags is a rack (this assumption should be addressed).
			// Makes sure the name equals current rack name; otherwise error out.
			return nil, fmt.Errorf("invalid rack: %s", name)
		}
	}

	return app, err
}

func GetAppBound(name string) (*App, error) {
	return getAppByStackName(shortNameToStackName(name))
}

func GetAppUnbound(name string) (*App, error) {
	return getAppByStackName(name)
}

func getAppByStackName(stackName string) (*App, error) {
	res, err := DescribeStack(stackName)

	if err != nil {
		return nil, err
	}

	app := appFromStack(res.Stacks[0])

	return app, nil
}

var regexValidAppName = regexp.MustCompile(`\A[a-zA-Z][-a-zA-Z0-9]{3,29}\z`)

func (a *App) IsBound() bool {
	if a.Tags == nil {
		// Default to bound.
		return true
	}

	if _, ok := a.Tags["Name"]; ok {
		// Bound apps MUST have a "Name" tag.
		return true
	}

	// Tags are present but "Name" tag is not, so we have an unbound app.
	return false
}

// StackName returns the app's stack if the app is bound. Otherwise returns the short name.
func (a *App) StackName() string {
	if a.IsBound() {
		return shortNameToStackName(a.Name)
	}

	return a.Name
}

func (a *App) Create() error {
	helpers.TrackEvent("kernel-app-create-start", nil)

	if !regexValidAppName.MatchString(a.Name) {
		return fmt.Errorf("app name can contain only alphanumeric characters, dashes and must be between 4 and 30 characters")
	}

	m := manifest1.Manifest{
		Services: make(map[string]manifest1.Service),
	}

	formation, err := a.Formation(m)
	if err != nil {
		helpers.TrackEvent("kernel-app-create-error", nil)
		return err
	}

	// SubnetsPrivate is a List<AWS::EC2::Subnet::Id> and can not be empty
	// So reuse SUBNETS if SUBNETS_PRIVATE is not set
	subnetsPrivate := os.Getenv("SUBNETS_PRIVATE")
	if subnetsPrivate == "" {
		subnetsPrivate = os.Getenv("SUBNETS")
	}

	params := map[string]string{
		"Cluster":        os.Getenv("CLUSTER"),
		"Internal":       os.Getenv("INTERNAL"),
		"LogBucket":      os.Getenv("LOG_BUCKET"),
		"Private":        os.Getenv("PRIVATE"),
		"Subnets":        os.Getenv("SUBNETS"),
		"SubnetsPrivate": subnetsPrivate,
		"Version":        os.Getenv("RELEASE"),
		"VPC":            os.Getenv("VPC"),
		"VPCCIDR":        os.Getenv("VPCCIDR"),
	}

	if os.Getenv("ENCRYPTION_KEY") != "" {
		params["Key"] = os.Getenv("ENCRYPTION_KEY")
	}

	tags := map[string]string{
		"Rack":   os.Getenv("RACK"),
		"System": "convox",
		"Type":   "app",
		"Name":   a.Name,
	}

	req := &cloudformation.CreateStackInput{
		Capabilities:     []*string{aws.String("CAPABILITY_IAM")},
		StackName:        aws.String(a.StackName()),
		TemplateBody:     aws.String(formation),
		NotificationARNs: []*string{aws.String(cloudformationTopic)},
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

	_, err = CloudFormation().CreateStack(req)

	if err != nil {
		helpers.TrackEvent("kernel-app-create-error", nil)
		return err
	}

	helpers.TrackEvent("kernel-app-create-success", nil)

	NotifySuccess("app:create", map[string]string{"name": a.Name})

	return nil
}

func (a *App) Delete() error {
	helpers.TrackEvent("kernel-app-delete-start", nil)

	err := Provider().AppDelete(a.Name)
	if err != nil {
		return err
	}

	NotifySuccess("app:delete", map[string]string{"name": a.Name})

	return nil
}

// Shortcut for updating current parameters
// If template changed, more care about new or removed parameters must be taken (see Release.Promote or System.Save)
func (a *App) UpdateParams(changes map[string]string) error {
	req := &cloudformation.UpdateStackInput{
		StackName:           aws.String(a.StackName()),
		Capabilities:        []*string{aws.String("CAPABILITY_IAM")},
		UsePreviousTemplate: aws.Bool(true),
		NotificationARNs:    []*string{aws.String(cloudformationTopic)},
	}

	// sort parameters by key name to make test requests stable
	var keys []string
	for key := range a.Parameters {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		if updatedValue, present := changes[key]; present {
			req.Parameters = append(req.Parameters, &cloudformation.Parameter{
				ParameterKey:   aws.String(key),
				ParameterValue: aws.String(updatedValue),
			})
		} else {
			req.Parameters = append(req.Parameters, &cloudformation.Parameter{
				ParameterKey:     aws.String(key),
				UsePreviousValue: aws.Bool(true),
			})
		}
	}

	_, err := UpdateStack(req)

	return err
}

func (a *App) Formation(m manifest1.Manifest) (string, error) {
	tmplData := map[string]interface{}{
		"App":      a,
		"Manifest": m,
	}
	data, err := assetTemplate("app", "app", tmplData)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (a *App) ForkRelease() (*Release, error) {
	release := structs.NewRelease(a.Name)

	if a.Release != "" {
		r, err := Provider().ReleaseGet(a.Name, a.Release)
		if err != nil {
			return nil, err
		}
		release = r
	}

	release.Id = generateId("R", 10)
	release.Created = time.Now()

	env, err := Provider().EnvironmentGet(a.Name)
	if err != nil {
		fmt.Printf("fn=ForkRelease level=error msg=\"error getting environment: %s\"", err)
	}

	release.Env = env.Raw()

	return &Release{
		Id:       release.Id,
		App:      release.App,
		Build:    release.Build,
		Env:      release.Env,
		Manifest: release.Manifest,
		Created:  release.Created,
	}, nil
}

func (a *App) Resources() (Resources, error) {
	resources, err := ListResources(a.Name)
	if err != nil {
		return nil, err
	}

	return resources, nil
}

func appFromStack(stack *cloudformation.Stack) *App {
	name := *stack.StackName
	tags := stackTags(stack)
	if value, ok := tags["Name"]; ok {
		// StackName probably includes the Rack prefix, prefer Name tag.
		name = value
	}
	return &App{
		Name:       name,
		Generation: first(stackTags(stack)["Generation"], "1"),
		Release:    first(stackOutputs(stack)["Release"], stackParameters(stack)["Release"]),
		Status:     humanStatus(*stack.StackStatus),
		Outputs:    stackOutputs(stack),
		Parameters: stackParameters(stack),
		Tags:       tags,
	}
}

func (s Apps) Len() int {
	return len(s)
}

func (s Apps) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

func (s Apps) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (a App) CronJobs(m manifest1.Manifest) []CronJob {
	cronjobs := []CronJob{}

	for _, entry := range m.Services {
		labels := entry.LabelsByPrefix("convox.cron")
		for key, value := range labels {
			cronjob := NewCronJobFromLabel(key, value)
			e := entry
			cronjob.Service = &e
			cronjob.App = &a
			cronjobs = append(cronjobs, cronjob)
		}
	}
	return cronjobs
}
