package aws

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/structs"
)

// ResourceCreate creates a new resource.
// Note: see also createResource() below.
func (p *AWSProvider) ResourceCreate(name, kind string, opts structs.ResourceCreateOptions) (*structs.Resource, error) {
	_, err := p.ResourceGet(name)
	if awsError(err) != "ValidationError" {
		return nil, fmt.Errorf("resource named %s already exists", name)
	}

	s := &structs.Resource{
		Name:       name,
		Parameters: cfParams(opts.Parameters),
		Type:       kind,
	}
	s.Parameters["CustomTopic"] = customTopic
	s.Parameters["NotificationTopic"] = notificationTopic

	var req *cloudformation.CreateStackInput

	switch s.Type {
	case "memcached", "mysql", "postgres", "redis", "sqs":
		req, err = p.createResource(s)
	case "fluentd":
		req, err = p.createResourceURL(s, "tcp")
	case "s3":
		if s.Parameters["Topic"] != "" {
			s.Parameters["Topic"] = fmt.Sprintf("%s-%s", p.Rack, s.Parameters["Topic"])
		}
		req, err = p.createResource(s)
	case "sns":
		if s.Parameters["Queue"] != "" {
			s.Parameters["Queue"] = fmt.Sprintf("%s-%s", p.Rack, s.Parameters["Queue"])
		}
		req, err = p.createResource(s)
	case "syslog":
		s.Parameters["Private"] = fmt.Sprintf("%t", p.SubnetsPrivate != "")
		req, err = p.createResourceURL(s, "tcp", "tcp+tls", "udp")
	case "webhook":
		s.Parameters["Url"] = fmt.Sprintf("http://%s/sns?endpoint=%s", p.NotificationHost, url.QueryEscape(s.Parameters["Url"]))
		req, err = p.createResourceURL(s, "http", "https")
	default:
		err = fmt.Errorf("Invalid resource type: %s", s.Type)
	}
	if err != nil {
		return s, err
	}

	keys := []string{}

	for key := range s.Parameters {
		keys = append(keys, key)
	}

	// sort keys for easier testing
	sort.Strings(keys)

	// pass through resource parameters as Cloudformation Parameters
	for _, key := range keys {
		req.Parameters = append(req.Parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String(key),
			ParameterValue: aws.String(s.Parameters[key]),
		})
	}

	// tag the resource
	tags := map[string]string{
		"Name":     s.Name,
		"Rack":     p.Rack,
		"Resource": s.Type,
		"System":   "convox",
		"Type":     "resource",
	}
	tagKeys := []string{}

	for key := range tags {
		tagKeys = append(tagKeys, key)
	}

	// sort keys for easier testing
	sort.Strings(tagKeys)
	for _, key := range tagKeys {
		req.Tags = append(req.Tags, &cloudformation.Tag{Key: aws.String(key), Value: aws.String(tags[key])})
	}

	_, err = p.cloudformation().CreateStack(req)

	p.EventSend(&structs.Event{
		Action: "resource:create",
		Data: map[string]string{
			"name": s.Name,
			"type": s.Type,
		},
	}, err)

	return s, err
}

// ResourceDelete deletes a resource.
func (p *AWSProvider) ResourceDelete(name string) (*structs.Resource, error) {
	s, err := p.ResourceGet(name)
	if err != nil {
		return nil, err
	}

	apps, err := p.resourceApps(*s)
	if err != nil {
		return nil, err
	}

	if len(apps) > 0 {
		return nil, fmt.Errorf("resource is linked to %s", apps[0].Name)
	}

	_, err = p.cloudformation().DeleteStack(&cloudformation.DeleteStackInput{
		StackName: aws.String(p.rackStack(s.Name)),
	})

	p.EventSend(&structs.Event{
		Action: "resource:delete",
		Data: map[string]string{
			"name": s.Name,
			"type": s.Type,
		},
	}, err)

	return s, err
}

// ResourceGet retrieves a resource.
func (p *AWSProvider) ResourceGet(name string) (*structs.Resource, error) {
	stacks, err := p.describeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(p.rackStack(name)),
	})
	if err != nil {
		return nil, err
	}
	if len(stacks) != 1 {
		return nil, fmt.Errorf("could not load stack for resource: %s", name)
	}

	outputs := stackOutputs(stacks[0])
	tags := stackTags(stacks[0])

	s := resourceFromStack(stacks[0])

	if tags["Rack"] != "" && tags["Rack"] != p.Rack {
		return nil, fmt.Errorf("no such stack on this rack: %s", name)
	}

	switch tags["Resource"] {
	case "memcached":
		s.Url = fmt.Sprintf("%s:%s", outputs["Port11211TcpAddr"], outputs["Port11211TcpPort"])
	case "mysql":
		s.Url = fmt.Sprintf("mysql://%s:%s@%s:%s/%s", outputs["EnvMysqlUsername"], outputs["EnvMysqlPassword"], outputs["Port3306TcpAddr"], outputs["Port3306TcpPort"], outputs["EnvMysqlDatabase"])
	case "papertrail":
		s.Url = s.Parameters["Url"]
	case "postgres":
		s.Url = fmt.Sprintf("postgres://%s:%s@%s:%s/%s", outputs["EnvPostgresUsername"], outputs["EnvPostgresPassword"], outputs["Port5432TcpAddr"], outputs["Port5432TcpPort"], outputs["EnvPostgresDatabase"])
	case "redis":
		s.Url = fmt.Sprintf("redis://%s:%s/%s", outputs["Port6379TcpAddr"], outputs["Port6379TcpPort"], outputs["EnvRedisDatabase"])
	case "s3":
		s.Url = fmt.Sprintf("s3://%s:%s@%s", outputs["AccessKey"], outputs["SecretAccessKey"], outputs["Bucket"])
	case "sns":
		s.Url = fmt.Sprintf("sns://%s:%s@%s", outputs["AccessKey"], outputs["SecretAccessKey"], outputs["Topic"])
	case "sqs":
		if u, err := url.Parse(outputs["Queue"]); err == nil {
			u.Scheme = "sqs"
			u.User = url.UserPassword(outputs["AccessKey"], outputs["SecretAccessKey"])
			s.Url = u.String()
		}
	case "webhook":
		if parsedURL, err := url.Parse(s.Parameters["Url"]); err == nil {
			s.Url = parsedURL.Query().Get("endpoint")
		}
	}

	// Populate linked apps
	apps, err := p.resourceApps(s)
	if err != nil {
		return nil, err
	}

	s.Apps = apps

	return &s, nil
}

//resourceApps returns the apps that have been linked with a resource (ignoring apps that have been delete out of band)
func (p *AWSProvider) resourceApps(s structs.Resource) (structs.Apps, error) {
	stacks, err := p.describeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(p.rackStack(s.Name)),
	})
	if err != nil {
		return nil, err
	}
	if len(stacks) != 1 {
		return nil, fmt.Errorf("could not load stack for resource: %s", s.Name)
	}

	outputs := stackOutputs(stacks[0])

	apps := structs.Apps(make([]structs.App, 0))

	for key, value := range outputs {
		if strings.HasSuffix(key, "Link") {
			// Extract app name from log group
			index := strings.Index(value, "-LogGroup")
			// avoid runtime panic
			if index == -1 {
				continue
			}
			r := strings.NewReplacer(fmt.Sprintf("%s-", p.Rack), "", value[index:], "")
			app := r.Replace(value)

			a, err := p.AppGet(app)
			if err != nil {
				if err.Error() == fmt.Sprintf("%s not found", app) {
					continue
				}
				return nil, err
			}

			apps = append(apps, *a)
		}
	}
	return apps, nil
}

// ResourceList lists the resources.
func (p *AWSProvider) ResourceList() (structs.Resources, error) {
	stacks, err := p.describeStacks(&cloudformation.DescribeStacksInput{})
	if err != nil {
		return nil, err
	}

	resources := structs.Resources{}

	for _, stack := range stacks {
		tags := stackTags(stack)

		// if it's a resource and the Rack tag is either the current rack or blank
		if tags["System"] == "convox" && (tags["Type"] == "resource" || tags["Type"] == "service") && tags["App"] == "" {
			if tags["Rack"] == p.Rack || tags["Rack"] == "" {
				resources = append(resources, resourceFromStack(stack))
			}
		}
	}

	for _, s := range resources {
		apps, err := p.resourceApps(s)
		if err != nil {
			return nil, err
		}
		s.Apps = apps
	}

	return resources, nil
}

// ResourceLink creates a link between the provided app and resource.
func (p *AWSProvider) ResourceLink(name, app, process string) (*structs.Resource, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	s, err := p.ResourceGet(name)
	if err != nil {
		return nil, err
	}

	// already linked
	apps, err := p.resourceApps(*s)
	if err != nil {
		return nil, err
	}

	for _, linkedApp := range apps {
		if a.Name == linkedApp.Name {
			return nil, fmt.Errorf("resource %s is already linked to app %s", s.Name, a.Name)
		}
	}

	// Update Resource and/or App stacks
	switch s.Type {
	case "fluentd", "syslog":
		err = p.linkResource(a, s) // Update resource to know about App
	default:
		err = fmt.Errorf("resource type %s does not have a link strategy", s.Type)
	}

	return s, err
}

// ResourceUnlink removes a link between the provided app and resource.
func (p *AWSProvider) ResourceUnlink(name, app, process string) (*structs.Resource, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	s, err := p.ResourceGet(name)
	if err != nil {
		return nil, err
	}

	apps, err := p.resourceApps(*s)
	if err != nil {
		return nil, err
	}

	// already linked
	linked := false
	for _, linkedApp := range apps {
		if a.Name == linkedApp.Name {
			linked = true
			break
		}
	}

	if !linked {
		return nil, fmt.Errorf("resource %s is not linked to app %s", s.Name, a.Name)
	}

	// Update Resource and/or App stacks
	switch s.Type {
	case "fluentd", "syslog":
		err = p.unlinkResource(a, s) // Update resource to forget about App
	default:
		err = fmt.Errorf("resource type %s does not have an unlink strategy", s.Type)
	}

	return s, err
}

// ResourceUpdate updates a resource with new params.
func (p *AWSProvider) ResourceUpdate(name string, params map[string]string) (*structs.Resource, error) {
	s, err := p.ResourceGet(name)
	if err != nil {
		return nil, err
	}

	for key, value := range cfParams(params) {
		s.Parameters[key] = value
	}

	err = p.updateResource(s)

	return s, err
}

// createResource creates a Resource.
// Note: see also ResourceCreate() above.
// This should probably be renamed to createResourceStack to be in conformity with createResourceURL below.
func (p *AWSProvider) createResource(s *structs.Resource) (*cloudformation.CreateStackInput, error) {
	formation, err := resourceFormation(s.Type, nil)
	if err != nil {
		return nil, err
	}

	if err := p.appendSystemParameters(s); err != nil {
		return nil, err
	}

	if err := filterFormationParameters(s, formation); err != nil {
		return nil, err
	}

	req := &cloudformation.CreateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(fmt.Sprintf("%s-%s", p.Rack, s.Name)),
		TemplateBody: aws.String(formation),
	}

	return req, nil
}

func (p *AWSProvider) createResourceURL(s *structs.Resource, allowedProtocols ...string) (*cloudformation.CreateStackInput, error) {
	if s.Parameters["Url"] == "" {
		return nil, fmt.Errorf("must specify a URL")
	}

	u, err := url.Parse(s.Parameters["Url"])
	if err != nil {
		return nil, err
	}

	valid := false

	for _, p := range allowedProtocols {
		if u.Scheme == p {
			valid = true
			break
		}
	}

	if !valid {
		return nil, fmt.Errorf("invalid URL scheme: %s. Allowed schemes are: %s", u.Scheme, strings.Join(allowedProtocols, ", "))
	}

	return p.createResource(s)
}

func (p *AWSProvider) updateResource(s *structs.Resource) error {
	formation, err := resourceFormation(s.Type, s)
	if err != nil {
		return err
	}

	if err := p.appendSystemParameters(s); err != nil {
		return err
	}

	if err := filterFormationParameters(s, formation); err != nil {
		return err
	}

	req := &cloudformation.UpdateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		StackName:    aws.String(p.rackStack(s.Name)),
		TemplateBody: aws.String(formation),
	}

	for key, value := range s.Parameters {
		req.Parameters = append(req.Parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String(key),
			ParameterValue: aws.String(value),
		})
	}

	_, err = p.cloudformation().UpdateStack(req)

	return err
}

// add to links
func (p *AWSProvider) linkResource(a *structs.App, s *structs.Resource) error {
	for _, app := range s.Apps {
		if a.Name == app.Name {
			return fmt.Errorf("app already linked")
		}
	}

	s.Apps = append(s.Apps, *a)

	return p.updateResource(s)
}

// delete from links
func (p *AWSProvider) unlinkResource(a *structs.App, s *structs.Resource) error {
	apps := structs.Apps{}

	for _, app := range s.Apps {
		if a.Name != app.Name {
			apps = append(apps, app)
		}
	}

	s.Apps = apps

	return p.updateResource(s)
}

func (p *AWSProvider) appendSystemParameters(s *structs.Resource) error {
	password, err := generatePassword()
	if err != nil {
		return err
	}

	if s.Parameters["Password"] == "" {
		s.Parameters["Password"] = password
	}

	s.Parameters["Release"] = p.Release
	s.Parameters["SecurityGroups"] = p.SecurityGroup
	s.Parameters["Subnets"] = p.Subnets
	s.Parameters["SubnetsPrivate"] = coalesceString(p.SubnetsPrivate, p.Subnets)
	s.Parameters["Vpc"] = p.Vpc
	s.Parameters["VpcCidr"] = p.VpcCidr

	return nil
}

func coalesceString(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

func filterFormationParameters(s *structs.Resource, formation string) error {
	var params struct {
		Parameters map[string]interface{}
	}

	if err := json.Unmarshal([]byte(formation), &params); err != nil {
		return err
	}

	for key := range s.Parameters {
		if _, ok := params.Parameters[key]; !ok {
			delete(s.Parameters, key)
		}
	}

	return nil
}

func generatePassword() (string, error) {
	data := make([]byte, 1024)

	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)

	return hex.EncodeToString(hash[:])[0:30], nil
}

func resourceFormation(kind string, data interface{}) (string, error) {
	d, err := buildTemplate(fmt.Sprintf("resource/%s", kind), "resource", data)
	if err != nil {
		return "", err
	}

	return d, nil
}

func resourceFromStack(stack *cloudformation.Stack) structs.Resource {
	params := stackParameters(stack)
	tags := stackTags(stack)
	name := coalesceString(tags["Name"], *stack.StackName)

	exports := map[string]string{}

	if url, ok := params["Url"]; ok {
		exports["URL"] = url
	}

	rtype := tags["Resource"]
	if rtype == "" {
		rtype = tags["Service"]
	}

	return structs.Resource{
		Name:       name,
		Parameters: params,
		Type:       rtype,
		Status:     humanStatus(*stack.StackStatus),
	}
}
