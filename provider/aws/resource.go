package aws

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) ResourceGet(app, name string) (*structs.Resource, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	m, _, err := helpers.AppManifest(p, a.Name)
	if err != nil {
		return nil, err
	}

	rts := map[string]string{}

	for _, r := range m.Resources {
		rts[r.Name] = r.Type
	}

	ar, err := p.describeStackResource(&cloudformation.DescribeStackResourceInput{
		StackName:         aws.String(p.rackStack(a.Name)),
		LogicalResourceId: aws.String(fmt.Sprintf("Resource%s", upperName(name))),
	})
	if err != nil {
		return nil, err
	}

	sr, err := p.describeStack(cs(ar.StackResourceDetail.PhysicalResourceId, ""))
	if err != nil {
		return nil, err
	}

	r := &structs.Resource{
		Name: name,
		Type: rts[name],
		Url:  stackOutputs(sr)["Url"],
	}

	return r, nil
}

func (p *Provider) ResourceList(app string) (structs.Resources, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	m, _, err := helpers.AppManifest(p, a.Name)
	if err != nil {
		return nil, err
	}

	ars, err := p.describeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(p.rackStack(a.Name)),
	})
	if err != nil {
		return nil, err
	}

	arsns := map[string]string{}

	for _, ar := range ars.StackResources {
		arsns[cs(ar.LogicalResourceId, "")] = cs(ar.PhysicalResourceId, "")
	}

	rs := structs.Resources{}

	for _, r := range m.Resources {
		if sr, err := p.describeStack(arsns[fmt.Sprintf("Resource%s", upperName(r.Name))]); err == nil {
			rs = append(rs, structs.Resource{
				Name: r.Name,
				Type: r.Type,
				Url:  stackOutputs(sr)["Url"],
			})
		}
	}

	return rs, nil
}

/** system resources ***************************************************************************/

var resourceSystemParameters = map[string]bool{
	"CustomTopic":       true,
	"NotificationTopic": true,
	"Private":           true,
	"Release":           true,
	"SecurityGroups":    true,
	"Subnets":           true,
	"SubnetsPrivate":    true,
	"Version":           true,
	"Vpc":               true,
	"VpcCidr":           true,
}

// ResourceCreate creates a new resource.
// Note: see also createResource() below.
func (p *Provider) SystemResourceCreate(kind string, opts structs.ResourceCreateOptions) (*structs.Resource, error) {
	name := fmt.Sprintf("%s-%d", kind, (rand.Intn(8999) + 1000))

	if opts.Name != nil {
		name = *opts.Name
	}

	_, err := p.SystemResourceGet(name)
	if awsError(err) != "ValidationError" {
		return nil, fmt.Errorf("resource named %s already exists", name)
	}

	s := &structs.Resource{
		Name:       name,
		Parameters: cfParams(opts.Parameters),
		Type:       kind,
	}

	var req *cloudformation.CreateStackInput

	switch s.Type {
	case "memcached", "mysql", "postgres", "redis", "sqs":
		req, err = p.createResource(s)
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
		req, err = p.createResourceURL(s, "tcp", "tcp+tls", "udp")
	case "webhook":
		req, err = p.createResourceURL(s, "http", "https")
	default:
		err = fmt.Errorf("invalid resource type: %s", s.Type)
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
	if err != nil {
		p.EventSend("resource:create", structs.EventSendOptions{Data: map[string]string{"name": s.Name, "type": s.Type}, Error: options.String(err.Error())})
		return nil, err
	}

	p.EventSend("resource:create", structs.EventSendOptions{Data: map[string]string{"name": s.Name, "type": s.Type}})

	return s, err
}

// ResourceDelete deletes a resource.
func (p *Provider) SystemResourceDelete(name string) error {
	r, err := p.SystemResourceGet(name)
	if err != nil {
		return err
	}

	apps, err := p.resourceApps(*r)
	if err != nil {
		return err
	}

	if len(apps) > 0 {
		return fmt.Errorf("resource is linked to %s", apps[0].Name)
	}

	switch r.Type {
	case "syslog":
		if err := p.deleteSyslogInterfaces(r); err != nil {
			return err
		}
	}

	_, err = p.cloudformation().DeleteStack(&cloudformation.DeleteStackInput{
		StackName: aws.String(p.rackStack(r.Name)),
	})
	if err != nil {
		p.EventSend("resource:delete", structs.EventSendOptions{Data: map[string]string{"name": r.Name, "type": r.Type}, Error: options.String(err.Error())})
		return err
	}

	p.EventSend("resource:delete", structs.EventSendOptions{Data: map[string]string{"name": r.Name, "type": r.Type}})

	return nil
}

// ResourceGet retrieves a resource.
func (p *Provider) SystemResourceGet(name string) (*structs.Resource, error) {
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

	if tags["Type"] != "resource" && tags["Type"] != "service" {
		return nil, errorNotFound(fmt.Sprintf("resource not found: %s", name))
	}

	s := resourceFromStack(stacks[0])

	if tags["Rack"] != p.Rack {
		return nil, fmt.Errorf("rack mismatch for stack: %s", name)
	}

	switch coalesces(tags["Resource"], tags["Service"]) {
	case "memcached":
		s.Url = fmt.Sprintf("%s:%s", outputs["Port11211TcpAddr"], outputs["Port11211TcpPort"])
	case "mysql":
		s.Url = fmt.Sprintf("mysql://%s:%s@%s:%s/%s", outputs["EnvMysqlUsername"], outputs["EnvMysqlPassword"], outputs["Port3306TcpAddr"], outputs["Port3306TcpPort"], outputs["EnvMysqlDatabase"])
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
	case "syslog":
		s.Url = s.Parameters["Url"]
	case "webhook":
		u, err := webhookURL(s.Parameters["Url"])
		if err != nil {
			return nil, err
		}
		s.Url = u
	}

	for k := range s.Parameters {
		if resourceSystemParameters[k] {
			delete(s.Parameters, k)
		}

		if k == "Password" {
			s.Parameters[k] = "****"
		}
	}

	// Populate linked apps
	switch s.Type {
	case "syslog":
		apps, err := p.resourceApps(s)
		if err != nil {
			return nil, err
		}
		s.Apps = apps
	}

	return &s, nil
}

// ResourceList lists the resources.
func (p *Provider) SystemResourceList() (structs.Resources, error) {
	stacks, err := p.describeStacks(&cloudformation.DescribeStacksInput{})
	if err != nil {
		return nil, err
	}

	resources := structs.Resources{}

	for _, stack := range stacks {
		tags := stackTags(stack)

		// if it's a resource and the Rack tag is either the current rack or blank
		if tags["System"] == "convox" && (tags["Type"] == "resource" || tags["Type"] == "service") && tags["App"] == "" && tags["Rack"] == p.Rack {
			resources = append(resources, resourceFromStack(stack))
		}
	}

	for _, s := range resources {
		switch s.Type {
		case "syslog":
			apps, err := p.resourceApps(s)
			if err != nil {
				return nil, err
			}
			s.Apps = apps
		}
	}

	return resources, nil
}

// ResourceLink creates a link between the provided app and resource.
func (p *Provider) SystemResourceLink(name, app string) (*structs.Resource, error) {
	s, err := p.SystemResourceGet(name)
	if err != nil {
		return nil, err
	}

	a, err := p.AppGet(app)
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

	s.Apps = apps

	// Update Resource and/or App stacks
	switch s.Type {
	case "syslog":
		err = p.linkResource(a, s) // Update resource to know about App
	default:
		err = fmt.Errorf("resource type %s does not have a link strategy", s.Type)
	}

	return s, err
}

func (p *Provider) SystemResourceTypes() (structs.ResourceTypes, error) {
	files, err := ioutil.ReadDir("provider/aws/templates/resource/")
	if err != nil {
		return nil, err
	}

	rts := structs.ResourceTypes{}

	for _, f := range files {
		name := strings.Split(f.Name(), ".")[0]

		data, err := resourceFormation(name, nil)
		if err != nil {
			return nil, err
		}

		rt := structs.ResourceType{
			Name:       name,
			Parameters: structs.ResourceParameters{},
		}

		var r struct {
			Parameters map[string]struct {
				Default     string
				Description string
			}
		}

		if err := json.Unmarshal([]byte(data), &r); err != nil {
			return nil, err
		}

		for k, p := range r.Parameters {
			def := p.Default

			if k == "Password" {
				def = "(generated)"
			}

			if resourceSystemParameters[k] {
				continue
			}

			rt.Parameters = append(rt.Parameters, structs.ResourceParameter{
				Default:     def,
				Description: p.Description,
				Name:        k,
			})
		}

		rts = append(rts, rt)
	}

	return rts, nil
}

// ResourceUnlink removes a link between the provided app and resource.
func (p *Provider) SystemResourceUnlink(name, app string) (*structs.Resource, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	s, err := p.SystemResourceGet(name)
	if err != nil {
		return nil, err
	}

	fmt.Printf("s.Parameters = %+v\n", s.Parameters)

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

	s.Apps = apps

	// Update Resource and/or App stacks
	switch s.Type {
	case "syslog":
		err = p.unlinkResource(a, s) // Update resource to forget about App
	default:
		err = fmt.Errorf("resource type %s does not have an unlink strategy", s.Type)
	}

	return s, err
}

// ResourceUpdate updates a resource with new params.
func (p *Provider) SystemResourceUpdate(name string, opts structs.ResourceUpdateOptions) (*structs.Resource, error) {
	s, err := p.SystemResourceGet(name)
	if err != nil {
		return nil, err
	}

	err = p.updateResource(s, opts.Parameters)

	return s, err
}

func (p *Provider) createResource(s *structs.Resource) (*cloudformation.CreateStackInput, error) {
	params := map[string]string{}

	for k, v := range s.Parameters {
		params[k] = v
	}

	formation, err := resourceFormation(s.Type, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range p.resourceSystemParameters() {
		s.Parameters[k] = v
	}

	if s.Parameters["Password"] == "" {
		pw, err := generatePassword()
		if err != nil {
			return nil, err
		}

		s.Parameters["Password"] = pw
	}

	// reapply manually-specified parameters
	for k, v := range params {
		s.Parameters[k] = v
	}

	if err := filterFormationParameters(s, formation); err != nil {
		return nil, err
	}

	req := &cloudformation.CreateStackInput{
		Capabilities:     []*string{aws.String("CAPABILITY_IAM")},
		NotificationARNs: []*string{aws.String(p.CloudformationTopic)},
		StackName:        aws.String(fmt.Sprintf("%s-%s", p.Rack, s.Name)),
		TemplateBody:     aws.String(formation),
	}

	return req, nil
}

func (p *Provider) createResourceURL(s *structs.Resource, allowedProtocols ...string) (*cloudformation.CreateStackInput, error) {
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

// clean up any ENIs attached to the lambda function as they will block stack deletion
func (p *Provider) deleteSyslogInterfaces(r *structs.Resource) error {
	fmt.Printf("r = %+v\n", r)

	fn, err := p.stackResource(p.rackStack(r.Name), "Function")
	if err != nil {
		fmt.Printf("err = %+v\n", err)
		return nil
	}

	res, err := p.ec2().DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("requester-id"), Values: []*string{aws.String(fmt.Sprintf("*:%s", *fn.PhysicalResourceId))}},
		},
	})
	if err != nil {
		return err
	}

	for _, ni := range res.NetworkInterfaces {
		if ni.Attachment != nil {
			_, err := p.ec2().DetachNetworkInterface(&ec2.DetachNetworkInterfaceInput{
				AttachmentId: ni.Attachment.AttachmentId,
			})
			if err != nil {
				return err
			}
		}

		for {
			res, err := p.ec2().DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
				Filters: []*ec2.Filter{
					{Name: aws.String("network-interface-id"), Values: []*string{ni.NetworkInterfaceId}},
				},
			})
			if err != nil {
				return err
			}
			if len(res.NetworkInterfaces) < 1 {
				return nil
			}

			if res.NetworkInterfaces[0].Attachment == nil {
				break
			}

			time.Sleep(1 * time.Second)
		}

		_, err := p.ec2().DeleteNetworkInterface(&ec2.DeleteNetworkInterfaceInput{
			NetworkInterfaceId: ni.NetworkInterfaceId,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

//resourceApps returns the apps that have been linked with a resource (ignoring apps that have been delete out of band)
func (p *Provider) resourceApps(s structs.Resource) (structs.Apps, error) {
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
			if ix := strings.Index(value, "-LogGroup"); ix > -1 {
				value = value[:ix]
			}
			if prefix := fmt.Sprintf("%s-", p.Rack); strings.HasPrefix(value, prefix) {
				value = strings.Replace(value, prefix, "", 1)
			}
			app := value

			a, err := p.AppGet(app)
			if err != nil {
				return nil, err
			}

			apps = append(apps, *a)
		}
	}

	return apps, nil
}

func (p *Provider) updateResource(s *structs.Resource, params map[string]string) error {
	formation, err := resourceFormation(s.Type, s)
	if err != nil {
		return err
	}

	if params == nil {
		params = map[string]string{}
	}

	for k, v := range p.resourceSystemParameters() {
		params[k] = v
	}

	// inject webhook url for backwards-compatibility
	if s.Type == "webhook" {
		params["Url"] = s.Url
	}

	tags := map[string]string{
		"Name":     s.Name,
		"Rack":     p.Rack,
		"Resource": s.Type,
		"System":   "convox",
		"Type":     "resource",
	}

	if err := p.updateStack(p.rackStack(s.Name), []byte(formation), "json", params, tags); err != nil {
		return err
	}

	return nil
}

// add to links
func (p *Provider) linkResource(a *structs.App, s *structs.Resource) error {
	for _, app := range s.Apps {
		if a.Name == app.Name {
			return fmt.Errorf("app already linked")
		}
	}

	s.Apps = append(s.Apps, *a)

	return p.updateResource(s, nil)
}

// delete from links
func (p *Provider) unlinkResource(a *structs.App, s *structs.Resource) error {
	apps := structs.Apps{}

	for _, app := range s.Apps {
		if a.Name != app.Name {
			apps = append(apps, app)
		}
	}

	s.Apps = apps

	return p.updateResource(s, nil)
}

func (p *Provider) resourceSystemParameters() map[string]string {
	params := map[string]string{}

	params["NotificationTopic"] = p.NotificationTopic
	params["Private"] = fmt.Sprintf("%t", p.SubnetsPrivate != "")
	params["Release"] = p.Version
	params["SecurityGroups"] = p.SecurityGroup
	params["Subnets"] = p.Subnets
	params["SubnetsPrivate"] = coalesceString(p.SubnetsPrivate, p.Subnets)
	params["Version"] = p.Version
	params["Vpc"] = p.Vpc
	params["VpcCidr"] = p.VpcCidr

	return params
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

	exports := map[string]string{}

	if url, ok := params["Url"]; ok {
		exports["URL"] = url
	}

	rtype := tags["Resource"]
	if rtype == "" {
		rtype = tags["Service"]
	}

	delete(params, "Password")

	return structs.Resource{
		Name:       tags["Name"],
		Parameters: params,
		Type:       rtype,
		Status:     humanStatus(*stack.StackStatus),
	}
}

func webhookURL(webhook string) (string, error) {
	if !strings.Contains(webhook, "/sns?endpoint=") {
		return webhook, nil
	}

	u, err := url.Parse(webhook)
	if err != nil {
		return "", err
	}

	return u.Query().Get("endpoint"), nil
}
