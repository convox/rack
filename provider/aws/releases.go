package aws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"math/rand"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/eventbridge"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/convox/rack/pkg/crypt"
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/manifest1"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) ReleaseCreate(app string, opts structs.ReleaseCreateOptions) (*structs.Release, error) {
	r := structs.NewRelease(app)

	cr, err := helpers.ReleaseLatest(p, app)
	if err != nil {
		return nil, err
	}

	if cr != nil {
		r.Build = cr.Build
		r.Env = cr.Env
	}

	if opts.Build != nil {
		r.Build = *opts.Build
	}

	if r.Build != "" {
		b, err := p.BuildGet(app, r.Build)
		if err != nil {
			return nil, err
		}

		r.Description = b.Description
		r.Manifest = b.Manifest
	}

	if opts.Env != nil {
		desc, err := helpers.EnvDiff(r.Env, *opts.Env)
		if err != nil {
			return nil, err
		}

		r.Description = fmt.Sprintf("env %s", desc)
		r.Env = *opts.Env
	}

	if opts.Description != nil {
		r.Description = *opts.Description
	}

	if err := p.releaseSave(r); err != nil {
		return nil, err
	}

	p.EventSend("release:create", structs.EventSendOptions{Data: map[string]string{"app": r.App, "id": r.Id}})

	return r, nil
}

// ReleaseGet returns a release
func (p *Provider) ReleaseGet(app, id string) (*structs.Release, error) {
	if id == "" {
		return nil, fmt.Errorf("release id must not be empty")
	}

	item, err := p.fetchRelease(app, id)
	if err != nil {
		return nil, err
	}

	r, err := releaseFromItem(item)
	if err != nil {
		return nil, err
	}

	settings, err := p.appResource(app, "Settings")
	if err != nil {
		return nil, err
	}

	data, err := p.s3Get(settings, fmt.Sprintf("releases/%s/env", r.Id))
	if err != nil {
		return nil, err
	}

	key, err := p.rackResource("EncryptionKey")
	if err != nil {
		return nil, err
	}

	if key != "" {
		if d, err := crypt.New().Decrypt(key, data); err == nil {
			data = d
		}
	}

	env := structs.Environment{}

	if err := env.Load(data); err != nil {
		return nil, err
	}

	r.Env = env.String()

	return r, nil
}

// ReleaseList returns a list of the latest releases, with the length specified in limit
func (p *Provider) ReleaseList(app string, opts structs.ReleaseListOptions) (structs.Releases, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	if opts.Limit == nil {
		opts.Limit = options.Int(10)
	}

	req := &dynamodb.QueryInput{
		KeyConditions: map[string]*dynamodb.Condition{
			"app": {
				AttributeValueList: []*dynamodb.AttributeValue{
					{S: aws.String(a.Name)},
				},
				ComparisonOperator: aws.String("EQ"),
			},
		},
		IndexName:        aws.String("app.created"),
		Limit:            aws.Int64(int64(*opts.Limit)),
		ScanIndexForward: aws.Bool(false),
		TableName:        aws.String(p.DynamoReleases),
	}

	res, err := p.dynamodb().Query(req)
	if err != nil {
		return nil, err
	}

	releases := make(structs.Releases, len(res.Items))

	for i, item := range res.Items {
		r, err := releaseFromItem(item)
		if err != nil {
			return nil, err
		}

		releases[i] = *r
	}

	return releases, nil
}

// ReleasePromote promotes a release
func (p *Provider) ReleasePromote(app, id string, opts structs.ReleasePromoteOptions) error {
	a, err := p.AppGet(app)
	if err != nil {
		return err
	}

	r, err := p.ReleaseGet(app, id)
	if err != nil {
		return err
	}

	switch a.Tags["Generation"] {
	case "", "1":
		return p.releasePromoteGeneration1(a, r)
	case "2":
	default:
		return fmt.Errorf("unknown generation for app: %s", a.Name)
	}

	env := structs.Environment{}

	if err := env.Load([]byte(r.Env)); err != nil {
		return err
	}

	m, err := manifest.Load([]byte(r.Manifest), env)
	if err != nil {
		return err
	}

	isRdsIngoreStep := false
	for _, r := range m.Resources {
		if opts.Xignore != nil && *opts.Xignore == r.Name {
			isRdsIngoreStep = true
		}
	}

	if isRdsIngoreStep {
		var temp manifest.Resources
		for _, r := range m.Resources {
			if *opts.Xignore != r.Name {
				temp = append(temp, r)
			}
		}
		m.Resources = temp
		m.Services = manifest.Services{}
		m.Timers = manifest.Timers{}
	}

	for _, s := range m.Services {
		if s.Internal && !p.Internal {
			return fmt.Errorf("rack does not support internal services")
		}

		if !s.Internal && p.InternalOnly {
			return fmt.Errorf("rack only supports internal services")
		}
	}

	cs, err := p.CertificateList()
	if err != nil {
		return err
	}

	ccs := structs.Certificates{}

	for _, c := range cs {
		if c.Expiration.After(time.Now()) {
			ccs = append(ccs, c)
		}
	}

	tp := map[string]interface{}{
		"App":          r.App,
		"Certificates": ccs,
		"Manifest":     m,
		"Password":     p.Password,
		"Release":      r,
		"Topic":        p.CloudformationTopic,
		"Version":      p.Version,
	}

	if r.Build != "" {
		b, err := p.BuildGet(app, r.Build)
		if err != nil {
			return err
		}

		tp["Build"] = b
		tp["WildcardDomain"] = b.WildcardDomain
	}

	hasThirdAZ := len(strings.Split(p.AvailabilityZones, ",")) == 3

	for _, r := range m.Resources {

		if opts.Xrds != nil && *opts.Xrds == r.Name && opts.Xsnapshot != nil {
			r.Options["snapshot"] = *opts.Xsnapshot
		}

		rtp := map[string]interface{}{
			"Name":                  r.Name,
			"ThirdAvailabilityZone": hasThirdAZ,
			"Tags":                  r.Tags,
		}

		var data []byte
		var params map[string]string

		if r.Options["noUpdate"] == "xtruex" {
			data, params, err = p.getResourceTemplateAndParams(app, r.Name)
			if err != nil {
				return err
			}

			// these params are set from the app itself, so remove from params
			removeParamKeys := []string{
				"Rack",
				"Password",
				"AutoMinorVersionUpgrade",
			}
			for _, k := range removeParamKeys {
				delete(params, k)
			}
		} else {
			data, err = formationTemplate(fmt.Sprintf("resource/%s", r.Type), rtp)
			if err != nil {
				return err
			}

			params, err = p.ResourceDefaults(app, r.Name)
			if err != nil {
				return err
			}

			for k, v := range r.Options {
				params[upperName(k)] = v
			}
		}

		if sourceDB := params[upperName("readSourceDB")]; sourceDB != "" {
			if rName := strings.TrimPrefix(sourceDB, "#convox.resources."); rName != sourceDB {
				exist := false
				if rName != r.Name {
					for i := range m.Resources {
						if m.Resources[i].Name == rName {
							exist = true
							if m.Resources[i].Type != r.Type {
								return fmt.Errorf("resource type mismatch for readSourceDB")
							}
							params[upperName("readSourceDB")], err = p.getResourceDBIdentifier(app, rName)
							if err != nil {
								return err
							}
						}
					}
				}
				if !exist {
					return fmt.Errorf("invalid resource name for readSourceDB")
				}
			}
		}

		ou, err := p.ObjectStore(app, "", bytes.NewReader(data), structs.ObjectStoreOptions{Presign: options.Bool(true)})
		if err != nil {
			return err
		}

		tp[fmt.Sprintf("ResourceParams%s", upperName(r.Name))] = params
		tp[fmt.Sprintf("ResourceTemplate%s", upperName(r.Name))] = ou.Url
	}

	for _, s := range m.Services {
		min := s.Deployment.Minimum
		max := s.Deployment.Maximum

		if opts.Min != nil {
			min = *opts.Min
		}

		if opts.Max != nil {
			max = *opts.Max
		}

		stackAutoscale, err := p.stackParameter(p.Rack, "Autoscale")
		if err != nil {
			return err
		}

		autoscale := false
		if stackAutoscale == "Yes" {
			autoscale = true
		}

		stp := map[string]interface{}{
			"App":            r.App,
			"Autoscale":      autoscale,
			"Build":          tp["Build"],
			"DeploymentMin":  min,
			"DeploymentMax":  max,
			"Manifest":       tp["Manifest"],
			"Password":       p.Password,
			"Release":        tp["Release"],
			"Service":        s,
			"Tags":           s.Tags,
			"WildcardDomain": tp["WildcardDomain"],
		}

		data, err := formationTemplate("service", stp)
		if err != nil {
			return err
		}

		ou, err := p.ObjectStore(app, "", bytes.NewReader(data), structs.ObjectStoreOptions{Presign: options.Bool(true)})
		if err != nil {
			return err
		}

		tp[fmt.Sprintf("ServiceTemplate%s", upperName(s.Name))] = ou.Url
	}

	for _, t := range m.Timers {
		ttp := map[string]interface{}{
			"App":       r.App,
			"Build":     tp["Build"],
			"Manifest":  tp["Manifest"],
			"Password":  p.Password,
			"Release":   tp["Release"],
			"Timer":     t,
			"TimeState": "",
		}

		if p.MaintainTimerState {
			ttp["TimeState"], err = p.getTimerState(app, t.Name)
			if err != nil {
				return err
			}
		}

		data, err := formationTemplate("timer", ttp)
		if err != nil {
			return err
		}

		ou, err := p.ObjectStore(app, "", bytes.NewReader(data), structs.ObjectStoreOptions{Presign: options.Bool(true)})
		if err != nil {
			return err
		}

		tp[fmt.Sprintf("TimerTemplate%s", upperName(t.Name))] = ou.Url
	}

	data, err := formationTemplate("app", tp)
	if err != nil {
		return err
	}

	private, err := p.stackParameter(p.Rack, "Private")
	if err != nil {
		return err
	}

	lambdaInVpc, err := p.stackParameter(p.Rack, "PlaceLambdaInVpc")
	if err != nil {
		return err
	}

	readonlyRootFilesystem, err := p.stackParameter(p.Rack, "EnableContainerReadonlyRootFilesystem")
	if err != nil {
		return err
	}

	updates := map[string]string{
		"LogBucket":                             p.LogBucket,
		"LogDriver":                             p.LogDriver,
		"PlaceLambdaInVpc":                      lambdaInVpc,
		"Private":                               private,
		"SyslogDestination":                     p.SyslogDestination,
		"SyslogFormat":                          p.SyslogFormat,
		"EnableContainerReadonlyRootFilesystem": readonlyRootFilesystem,
	}

	if m.Params != nil {
		for k, v := range m.Params {
			updates[k] = v
		}
	}

	tags := map[string]string{
		"Version": p.Version,
	}

	customRackTags, err := p.getCustomTags(p.Rack)
	if err != nil {
		return err
	}
	customAppTags, err := p.getCustomTags(p.rackStack(r.App))
	if err != nil {
		return err
	}
	for k, v := range customRackTags {
		tags[k] = v
	}
	for k, v := range customAppTags {
		tags[k] = v
	}

	cfid := fmt.Sprintf("%s-%s", time.Now().UTC().Format(helpers.CompactSortableTime), r.Id)

	if err := p.updateStack(p.rackStack(r.App), data, updates, tags, cfid); err != nil {
		return err
	}

	p.EventSend("release:promote", structs.EventSendOptions{Data: map[string]string{"app": r.App, "id": r.Id}, Status: options.String("start")})

	return nil
}

func (p *Provider) getCustomTags(rackName string) (map[string]string, error) {
	stack, err := p.describeStack(rackName)
	if err != nil {
		return nil, err
	}

	tags := stackTags(stack)

	reservedTagNames := []string{"App", "System", "Type", "Version", "Generation", "Name", "Rack"}

	for _, r := range reservedTagNames {
		delete(tags, r)
	}

	return tags, nil
}

func (p *Provider) releasePromoteGeneration1(a *structs.App, r *structs.Release) error {
	m, err := manifest1.Load([]byte(r.Manifest))
	if err != nil {
		return err
	}

	// set the image
	for i, entry := range m.Services {
		s := m.Services[i]
		s.Image = entry.RegistryImage(a.Name, r.Build, a.Outputs)
		m.Services[i] = s
	}

	m, err = p.resolveLinks(a, m, r)
	if err != nil {
		return err
	}

	settings, err := p.appResource(r.App, "Settings")
	if err != nil {
		return err
	}

	tp := map[string]interface{}{
		"App":         a,
		"Cluster":     p.Cluster,
		"Environment": fmt.Sprintf("https://%s.s3.amazonaws.com/releases/%s/env", settings, r.Id),
		"Manifest":    m,
		"Region":      p.Region,
		"Version":     p.Version,
	}

	if r.Build != "" {
		b, err := p.BuildGet(a.Name, r.Build)
		if err != nil {
			return err
		}

		tp["Build"] = b
	}

	data, err := formationTemplate("g1/app", tp)
	if err != nil {
		return err
	}

	// If release formation was saved in S3, get that instead
	f, err := p.s3Get(settings, fmt.Sprintf("templates/%s", r.Id))
	if err != nil && awsError(err) != "NoSuchKey" {
		return err
	}
	if err == nil {
		data = f
	}

	fmt.Printf("ns=kernel at=release.promote at=s3Get found=%t\n", err == nil)

	params := map[string]string{}

	params["Cluster"] = p.Cluster
	params["Key"] = p.EncryptionKey
	params["ELBLogBucket"] = p.ELBLogBucket
	params["LogBucket"] = p.LogBucket
	params["Rack"] = p.Rack
	params["Release"] = r.Id
	params["Subnets"] = p.Subnets
	params["SubnetsPrivate"] = coalesces(p.SubnetsPrivate, p.Subnets)
	params["VPC"] = p.Vpc
	params["VPCCIDR"] = p.VpcCidr

	for _, entry := range m.Services {
		for _, mapping := range entry.Ports {
			listenerParam := fmt.Sprintf("%sPort%dListener", upperName(entry.Name), mapping.Balancer)

			randomPort := entry.Randoms()[strconv.Itoa(mapping.Balancer)]
			listener := []string{strconv.Itoa(randomPort), ""}

			// copy values from existing parameters
			if v, ok := a.Parameters[listenerParam]; ok {
				listener = strings.Split(v, ",")
				if len(listener) != 2 {
					return fmt.Errorf("%s not in port,cert format", listenerParam)
				}
			}

			// validate protocol labels
			proto := entry.Labels[fmt.Sprintf("convox.port.%d.protocol", mapping.Balancer)]

			// set a default cert if not defined in existing parameter
			switch proto {
			case "https", "tls":
				if listener[1] == "" {
					// if rack already has a self-signed cert, reuse it
					certs, err := p.iam().ListServerCertificates(&iam.ListServerCertificatesInput{})
					if err != nil {
						return err
					}

					for _, cert := range certs.ServerCertificateMetadataList {
						if strings.Contains(*cert.Arn, fmt.Sprintf("server-certificate/cert-%s-", p.Rack)) {
							listener[1] = *cert.Arn
							break
						}
					}

					// if not, generate and upload a self-signed cert
					if listener[1] == "" {
						name := fmt.Sprintf("cert-%s-%d-%05d", p.Rack, time.Now().Unix(), rand.Intn(100000))

						body, key, err := generateSelfSignedCertificate("*.*.elb.amazonaws.com")
						if err != nil {
							return err
						}

						input := &iam.UploadServerCertificateInput{
							CertificateBody:       aws.String(string(body)),
							PrivateKey:            aws.String(string(key)),
							ServerCertificateName: aws.String(name),
						}

						res, err := p.iam().UploadServerCertificate(input)
						if err != nil {
							return err
						}

						listener[1] = *res.ServerCertificateMetadata.Arn

						if err := p.waitForServerCertificate(name); err != nil {
							return err
						}
					}

					params[listenerParam] = strings.Join(listener, ",")
				}
			}
		}
	}

	// cache the template
	if err := p.s3Put(settings, fmt.Sprintf("templates/%s", r.Id), data, false); err != nil {
		return err
	}

	if err := p.updateStack(p.rackStack(a.Name), data, params, map[string]string{}, r.Id); err != nil {
		return err
	}

	p.EventSend("release:promote", structs.EventSendOptions{Data: map[string]string{"app": r.App, "id": r.Id}, Status: options.String("start")})

	return nil
}

// ReleaseSave saves a Release
func (p *Provider) releaseSave(r *structs.Release) error {
	if r.Id == "" {
		return fmt.Errorf("Id can not be blank")
	}

	if r.Created.IsZero() {
		r.Created = time.Now().UTC()
	}

	if p.IsTest() {
		r.Created = time.Unix(1473028693, 0).UTC()
	}

	req := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"id":      {S: aws.String(r.Id)},
			"app":     {S: aws.String(r.App)},
			"created": {S: aws.String(r.Created.Format(sortableTime))},
		},
		TableName: aws.String(p.DynamoReleases),
	}

	if r.Build != "" {
		req.Item["build"] = &dynamodb.AttributeValue{S: aws.String(r.Build)}
	}

	if r.Description != "" {
		req.Item["description"] = &dynamodb.AttributeValue{S: aws.String(r.Description)}
	}

	if r.Manifest != "" {
		req.Item["manifest"] = &dynamodb.AttributeValue{S: aws.String(r.Manifest)}
	}

	env := []byte(r.Env)

	key, err := p.rackResource("EncryptionKey")
	if err != nil {
		return err
	}

	if key != "" {
		env, err = crypt.New().Encrypt(key, []byte(env))
		if err != nil {
			return err
		}
	}

	settings, err := p.appResource(r.App, "Settings")
	if err != nil {
		return err
	}

	a, err := p.AppGet(r.App)
	if err != nil {
		return err
	}

	sreq := &s3.PutObjectInput{
		Body:          bytes.NewReader(env),
		Bucket:        aws.String(settings),
		ContentLength: aws.Int64(int64(len(env))),
		Key:           aws.String(fmt.Sprintf("releases/%s/env", r.Id)),
	}

	switch a.Tags["Generation"] {
	case "2":
	default:
		sreq.ACL = aws.String("public-read")
	}

	_, err = p.s3().PutObject(sreq)
	if err != nil {
		return err
	}

	_, err = p.dynamodb().PutItem(req)
	return err
}

func (p *Provider) fetchRelease(app, id string) (map[string]*dynamodb.AttributeValue, error) {
	res, err := p.dynamodb().GetItem(&dynamodb.GetItemInput{
		ConsistentRead: aws.Bool(true),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {S: aws.String(id)},
		},
		TableName: aws.String(p.DynamoReleases),
	})
	if err != nil {
		return nil, err
	}
	if res.Item == nil {
		return nil, errorNotFound(fmt.Sprintf("release not found: %s", id))
	}
	if res.Item["app"] == nil || *res.Item["app"].S != app {
		return nil, fmt.Errorf("mismatched app and release")
	}

	return res.Item, nil
}

func releaseFromItem(item map[string]*dynamodb.AttributeValue) (*structs.Release, error) {
	created, err := time.Parse(sortableTime, coalesce(item["created"], ""))
	if err != nil {
		return nil, err
	}

	release := &structs.Release{
		Id:          coalesce(item["id"], ""),
		App:         coalesce(item["app"], ""),
		Build:       coalesce(item["build"], ""),
		Manifest:    coalesce(item["manifest"], ""),
		Description: coalesce(item["description"], ""),
		Created:     created,
	}

	return release, nil
}

// releasesDeleteAll will delete all releases associate with app
// This includes the active release which implies this should only be called when deleting an app.
func (p *Provider) releaseDeleteAll(app string) error {

	// query dynamo for all releases for this app
	qi := &dynamodb.QueryInput{
		KeyConditionExpression: aws.String("app = :app"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":app": {S: aws.String(app)},
		},
		IndexName: aws.String("app.created"),
		TableName: aws.String(p.DynamoReleases),
	}

	return p.deleteReleaseItems(qi, p.DynamoReleases)
}

// deleteReleaseItems deletes release items from Dynamodb based on query input and the tableName
func (p *Provider) deleteReleaseItems(qi *dynamodb.QueryInput, tableName string) error {
	res, err := p.dynamodb().Query(qi)
	if err != nil {
		return err
	}

	// collect release IDs to delete
	wrs := []*dynamodb.WriteRequest{}
	for _, item := range res.Items {
		r, err := releaseFromItem(item)
		if err != nil {
			return err
		}

		wr := &dynamodb.WriteRequest{
			DeleteRequest: &dynamodb.DeleteRequest{
				Key: map[string]*dynamodb.AttributeValue{
					"id": {
						S: aws.String(r.Id),
					},
				},
			},
		}

		wrs = append(wrs, wr)
	}

	return p.dynamoBatchDeleteItems(wrs, tableName)
}

func (p *Provider) resolveLinks(a *structs.App, m *manifest1.Manifest, r *structs.Release) (*manifest1.Manifest, error) {
	var registries map[string]struct {
		Username string
		Password string
	}

	data, err := p.buildAuth(nil)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(data), &registries); err != nil {
		return nil, err
	}

	for server, creds := range registries {
		if err := exec.Command("docker", "login", "-u", creds.Username, "-p", creds.Password, server).Run(); err != nil {
			return nil, err
		}
	}

	for i, entry := range m.Services {
		var inspect []struct {
			Config struct {
				Env []string
			}
		}

		imageName := entry.Image

		cmd := exec.Command("docker", "pull", imageName)
		out, err := cmd.CombinedOutput()
		fmt.Printf("ns=kernel at=release.formation at=entry.pull imageName=%q out=%q err=%q\n", imageName, string(out), err)
		if err != nil {
			return m, fmt.Errorf("could not pull %q", imageName)
		}

		cmd = exec.Command("docker", "inspect", imageName)
		out, err = cmd.CombinedOutput()
		// fmt.Printf("ns=kernel at=release.formation at=entry.inspect imageName=%q out=%q err=%q\n", imageName, string(out), err)
		if err != nil {
			return m, fmt.Errorf("could not inspect %q", imageName)
		}

		err = json.Unmarshal(out, &inspect)
		if err != nil {
			fmt.Printf("ns=kernel at=release.formation at=entry.unmarshal err=%q\n", err)
			return m, fmt.Errorf("could not inspect %q", imageName)
		}

		entry.Exports = make(map[string]string)
		linkableEnvs := make([]string, len(entry.Environment))
		for _, env := range entry.Environment {
			linkableEnvs = append(linkableEnvs, fmt.Sprintf("%s=%s", env.Name, env.Value))
		}

		if len(inspect) == 1 {
			linkableEnvs = append(linkableEnvs, inspect[0].Config.Env...)
		}

		for _, val := range linkableEnvs {
			if strings.HasPrefix(val, "LINK_") {
				parts := strings.SplitN(val, "=", 2)
				if len(parts) == 2 {
					entry.Exports[parts[0]] = parts[1]
					m.Services[i] = entry
				}
			}
		}
	}

	for i, entry := range m.Services {
		entry.LinkVars = make(map[string]template.HTML)
		for _, link := range entry.Links {
			other, ok := m.Services[link]
			if !ok {
				return m, fmt.Errorf("Cannot find link %q", link)
			}

			scheme := other.Exports["LINK_SCHEME"]
			if scheme == "" {
				scheme = "tcp"
			}

			mb := m.GetBalancer(link)
			if mb == nil {
				// commented out to be less strict, just don't create the link
				//return m, fmt.Errorf("Cannot discover balancer for link %q", link)
				continue
			}
			host := fmt.Sprintf(`{ "Fn::If" : [ "Enabled%s", { "Fn::GetAtt" : [ "%s", "DNSName" ] }, "DISABLED" ] }`, upperName(other.Name), mb.ResourceName())

			if len(other.Ports) == 0 {
				// commented out to be less strict, just don't create the link
				// return m, fmt.Errorf("Cannot link to %q because it does not expose ports in the manifest", link)
				continue
			}

			var port manifest1.Port
			linkPort := other.Exports["LINK_PORT"]
			if linkPort == "" {
				port = other.Ports[0]
			} else {
				i, err := strconv.Atoi(linkPort)
				if err != nil {
					return nil, err
				}

				var matchedPort = false
				for _, p := range other.Ports {
					if i == p.Container {
						port = p
						matchedPort = true
					}
				}

				if !matchedPort {
					return nil, fmt.Errorf("No Port matching %s found", linkPort)
				}
			}

			path := other.Exports["LINK_PATH"]

			var userInfo string
			if other.Exports["LINK_USERNAME"] != "" || other.Exports["LINK_PASSWORD"] != "" {
				userInfo = fmt.Sprintf("%s:%s@", other.Exports["LINK_USERNAME"], other.Exports["LINK_PASSWORD"])
			}

			html := fmt.Sprintf(`{ "Fn::Join": [ "", [ "%s", "://", "%s", %s, ":", "%d", "%s" ] ] }`,
				scheme, userInfo, host, port.Balancer, path)

			prefix := strings.ToUpper(link) + "_"
			prefix = strings.Replace(prefix, "-", "_", -1)
			entry.LinkVars[prefix+"HOST"] = template.HTML(host)
			entry.LinkVars[prefix+"SCHEME"] = template.HTML(fmt.Sprintf("%q", scheme))
			entry.LinkVars[prefix+"PORT"] = template.HTML(fmt.Sprintf("%d", port.Balancer))
			entry.LinkVars[prefix+"PASSWORD"] = template.HTML(fmt.Sprintf("%q", other.Exports["LINK_PASSWORD"]))
			entry.LinkVars[prefix+"USERNAME"] = template.HTML(fmt.Sprintf("%q", other.Exports["LINK_USERNAME"]))
			entry.LinkVars[prefix+"PATH"] = template.HTML(fmt.Sprintf("%q", path))
			entry.LinkVars[prefix+"URL"] = template.HTML(html)
			m.Services[i] = entry
		}
	}

	return m, nil
}

func (p *Provider) getTimerState(app, timerName string) (string, error) {
	ars, err := p.describeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(p.rackStack(app)),
	})
	if err != nil {
		return "", err
	}

	arsns := map[string]string{}

	for _, ar := range ars.StackResources {
		arsns[cs(ar.LogicalResourceId, "")] = cs(ar.PhysicalResourceId, "")
	}

	timerOut, err := p.describeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(arsns[fmt.Sprintf("Timer%s", upperName(timerName))]),
	})
	if err != nil {
		return "", err
	}

	for _, to := range timerOut.StackResources {
		if cs(to.LogicalResourceId, "") == "Timer" {
			resp, err := p.eventbridge().DescribeRule(&eventbridge.DescribeRuleInput{
				Name: to.PhysicalResourceId,
			})
			if err != nil {
				return "", err
			}
			return *resp.State, nil
		}
	}
	return "", fmt.Errorf("Timer rule not found")
}

func (p *Provider) getResourceTemplateAndParams(app, resourceName string) ([]byte, map[string]string, error) {
	ars, err := p.describeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(p.rackStack(app)),
	})
	if err != nil {
		return nil, nil, err
	}

	arsns := map[string]string{}

	for _, ar := range ars.StackResources {
		arsns[cs(ar.LogicalResourceId, "")] = cs(ar.PhysicalResourceId, "")
	}

	stack := arsns[fmt.Sprintf("Resource%s", upperName(resourceName))]

	tmplBody, err := p.stackTemplate(stack)
	if err != nil {
		return nil, nil, err
	}

	res, err := p.describeStack(stack)
	if err != nil {
		return nil, nil, err
	}

	params := map[string]string{}
	for _, p := range res.Parameters {
		params[*p.ParameterKey] = *p.ParameterValue
	}

	return tmplBody, params, nil
}

func (p *Provider) getResourceDBIdentifier(app, resourceName string) (string, error) {
	ars, err := p.describeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(p.rackStack(app)),
	})
	if err != nil {
		return "", err
	}

	arsns := map[string]string{}

	for _, ar := range ars.StackResources {
		arsns[cs(ar.LogicalResourceId, "")] = cs(ar.PhysicalResourceId, "")
	}

	stack := arsns[fmt.Sprintf("Resource%s", upperName(resourceName))]
	res, err := p.describeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(stack),
	})
	if err != nil {
		return "", err
	}

	for _, r := range res.StackResources {
		if *r.ResourceType == "AWS::RDS::DBInstance" {
			return *r.PhysicalResourceId, nil
		}
	}

	return "", fmt.Errorf("db instance not found")
}
