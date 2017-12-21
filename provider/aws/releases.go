package aws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/convox/rack/crypt"
	"github.com/convox/rack/helpers"
	"github.com/convox/rack/manifest"
	"github.com/convox/rack/manifest1"
	"github.com/convox/rack/structs"
)

func (p *AWSProvider) ReleaseCreate(app string, opts structs.ReleaseCreateOptions) (*structs.Release, error) {
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

	if opts.Env != nil {
		r.Env = *opts.Env
	}

	if r.Build != "" {
		b, err := p.BuildGet(app, r.Build)
		if err != nil {
			return nil, err
		}

		r.Manifest = b.Manifest
	}

	fmt.Printf("r = %+v\n", r)

	if err := p.releaseSave(r); err != nil {
		return nil, err
	}

	return r, nil
}

// ReleaseGet returns a release
func (p *AWSProvider) ReleaseGet(app, id string) (*structs.Release, error) {
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
func (p *AWSProvider) ReleaseList(app string, opts structs.ReleaseListOptions) (structs.Releases, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	if opts.Count == 0 {
		opts.Count = 10
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
		Limit:            aws.Int64(int64(opts.Count)),
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
func (p *AWSProvider) ReleasePromote(app, id string) error {
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

	m, err := manifest.Load([]byte(r.Manifest), manifest.Environment(env))
	if err != nil {
		return err
	}

	for _, s := range m.Services {
		if s.Internal && !p.Internal {
			return fmt.Errorf("rack does not support internal services")
		}
	}

	tp := map[string]interface{}{
		"App":      r.App,
		"Env":      env,
		"Manifest": m,
		"Release":  r,
		"Version":  p.Release,
	}

	data, err := formationTemplate("app", tp)
	if err != nil {
		return err
	}

	// fmt.Printf("string(data) = %+v\n", string(data))

	ou, err := p.ObjectStore(app, "", bytes.NewReader(data), structs.ObjectStoreOptions{Public: true})
	if err != nil {
		return err
	}

	updates := map[string]string{
		"LogBucket": p.LogBucket,
	}

	if err := p.updateStack(p.rackStack(r.App), ou.Url, updates); err != nil {
		return err
	}

	go p.waitForPromotion(r)

	return nil
}

func (p *AWSProvider) releasePromoteGeneration1(a *structs.App, r *structs.Release) error {
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
		"Environment": fmt.Sprintf("https://%s.s3.amazonaws.com/releases/%s/env", settings, r.Id),
		"Manifest":    m,
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
	params["LogBucket"] = p.LogBucket
	params["Rack"] = p.Rack
	params["Release"] = r.Id
	params["Subnets"] = p.Subnets
	params["SubnetsPrivate"] = coalesces(p.SubnetsPrivate, p.Subnets)
	params["Version"] = p.Release
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
						if strings.Contains(*cert.Arn, fmt.Sprintf("server-certificate/cert-%s-", os.Getenv("RACK"))) {
							listener[1] = *cert.Arn
							break
						}
					}

					// if not, generate and upload a self-signed cert
					if listener[1] == "" {
						name := fmt.Sprintf("cert-%s-%d-%05d", os.Getenv("RACK"), time.Now().Unix(), rand.Intn(100000))

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

	ou, err := p.ObjectStore(a.Name, "", bytes.NewReader(data), structs.ObjectStoreOptions{Public: true})
	if err != nil {
		return err
	}

	if err := p.updateStack(p.rackStack(a.Name), ou.Url, params); err != nil {
		return err
	}

	go p.waitForPromotion(r)

	return err
}

// ReleaseSave saves a Release
func (p *AWSProvider) releaseSave(r *structs.Release) error {
	if r.Id == "" {
		return fmt.Errorf("Id can not be blank")
	}

	if r.Created.IsZero() {
		r.Created = time.Now()
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

func (p *AWSProvider) fetchRelease(app, id string) (map[string]*dynamodb.AttributeValue, error) {
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
		return nil, errorNotFound(fmt.Sprintf("no such release: %s", id))
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
		Id:       coalesce(item["id"], ""),
		App:      coalesce(item["app"], ""),
		Build:    coalesce(item["build"], ""),
		Manifest: coalesce(item["manifest"], ""),
		Created:  created,
	}

	return release, nil
}

// releasesDeleteAll will delete all releases associate with app
// This includes the active release which implies this should only be called when deleting an app.
func (p *AWSProvider) releaseDeleteAll(app string) error {

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
func (p *AWSProvider) deleteReleaseItems(qi *dynamodb.QueryInput, tableName string) error {
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

func (p *AWSProvider) resolveLinks(a *structs.App, m *manifest1.Manifest, r *structs.Release) (*manifest1.Manifest, error) {
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
		for k, v := range entry.Environment {
			val := fmt.Sprintf("%s=%s", k, v)
			linkableEnvs = append(linkableEnvs, val)
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

func (p *AWSProvider) waitForPromotion(r *structs.Release) {
	event := &structs.Event{
		Action: "release:promote",
		Data: map[string]string{
			"app": r.App,
			"id":  r.Id,
		},
	}
	stackName := fmt.Sprintf("%s-%s", os.Getenv("RACK"), r.App)

	waitch := make(chan error)
	go func() {
		var err error
		//we have observed stack stabalization failures take up to 3 hours
		for i := 0; i < 3; i++ {
			err = p.cloudformation().WaitUntilStackUpdateComplete(&cloudformation.DescribeStacksInput{
				StackName: aws.String(stackName),
			})
			if err != nil {
				if err.Error() == "exceeded 120 wait attempts" {
					continue
				}
			}
			break
		}
		waitch <- err
	}()

	for {
		select {
		case err := <-waitch:
			if err == nil {
				event.Status = "success"
				p.EventSend(event, nil)
				return
			}

			if err != nil && err.Error() == "exceeded 120 wait attempts" {
				p.EventSend(event, fmt.Errorf("couldn't determine promotion status, timed out"))
				fmt.Println(fmt.Errorf("couldn't determine promotion status, timed out"))
				return
			}

			resp, err := p.cloudformation().DescribeStacks(&cloudformation.DescribeStacksInput{
				StackName: aws.String(stackName),
			})
			if err != nil {
				p.EventSend(event, fmt.Errorf("unable to check stack status: %s", err))
				fmt.Println(fmt.Errorf("unable to check stack status: %s", err))
				return
			}

			if len(resp.Stacks) < 1 {
				p.EventSend(event, fmt.Errorf("app stack was not found: %s", stackName))
				fmt.Println(fmt.Errorf("app stack was not found: %s", stackName))
				return
			}

			se, err := p.cloudformation().DescribeStackEvents(&cloudformation.DescribeStackEventsInput{
				StackName: aws.String(stackName),
			})
			if err != nil {
				p.EventSend(event, fmt.Errorf("unable to check stack events: %s", err))
				fmt.Println(fmt.Errorf("unable to check stack events: %s", err))
				return
			}

			var lastEvent *cloudformation.StackEvent

			for _, e := range se.StackEvents {
				switch *e.ResourceStatus {
				case "UPDATE_FAILED", "DELETE_FAILED", "CREATE_FAILED":
					lastEvent = e
					break
				}
			}

			ee := fmt.Errorf("unable to determine release error")
			if lastEvent != nil {
				ee = fmt.Errorf(
					"[%s:%s] [%s]: %s",
					*lastEvent.ResourceType,
					*lastEvent.LogicalResourceId,
					*lastEvent.ResourceStatus,
					*lastEvent.ResourceStatusReason,
				)
			}

			p.EventSend(event, fmt.Errorf("release %s failed - %s", r.Id, ee.Error()))
		}
	}
}
