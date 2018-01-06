package aws

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	docker "github.com/fsouza/go-dockerclient"

	"github.com/convox/rack/manifest"
	"github.com/convox/rack/manifest1"
	"github.com/convox/rack/options"
	"github.com/convox/rack/structs"
)

// ECR host is formatted like 123456789012.dkr.ecr.us-east-1.amazonaws.com
var regexpECRHost = regexp.MustCompile(`(\d+)\.dkr\.ecr\.([^.]+)\.amazonaws\.com`)
var regexpECRImage = regexp.MustCompile(`(\d+)\.dkr\.ecr\.([^.]+)\.amazonaws\.com\/([^:]+):([^ ]+)`)

// StackID is formatted like arn:aws:cloudformation:us-east-1:332653055745:stack/dev/d164aa20-ba89-11e6-b65c-50d5ca632656
var regexpStackID = regexp.MustCompile(`arn:aws:cloudformation:([^.]+):(\d+):stack/([^.]+)/([^.]+)`)

func (p *AWSProvider) BuildCreate(app, method, url string, opts structs.BuildCreateOptions) (*structs.Build, error) {
	log := Logger.At("BuildCreate").Namespace("app=%q method=%q url=%q", app, method, url).Start()

	_, err := p.AppGet(app)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	b := structs.NewBuild(app)

	if opts.Description != nil {
		b.Description = *opts.Description
	}

	b.Started = time.Now()

	if p.IsTest() {
		b.Id = "B123"
		b.Started = time.Unix(1473028693, 0).UTC()
		b.Ended = time.Unix(1473028892, 0).UTC()
	}

	if err := p.buildSave(b); err != nil {
		log.Error(err)
		return nil, err
	}

	if err := p.runBuild(b, method, url, opts); err != nil {
		log.Error(err)
		return nil, err
	}

	// AWS currently has a limit of 1000 images in ECR
	// This is a "hopefully temporary" and brute force means
	// to prevent hitting limits during deployment
	bs, err := p.BuildList(app, structs.BuildListOptions{Count: aws.Int(150)})
	if err != nil {
		fmt.Printf("Error listing builds for cleanup: %s\n", err.Error())
	} else {
		if len(bs) >= 50 {
			go func() {
				for _, b := range bs[50:] {
					_, err := p.BuildDelete(app, b.Id)
					if err != nil {
						fmt.Printf("Error cleaning up build %s: %s\n", b.Id, err.Error())
					}
					time.Sleep(1 * time.Second)
				}
			}()
		}
	}

	log.Success()
	return b, nil
}

// BuildDelete deletes the build specified by id belonging to app
// Care should be taken as this could delete the build used by the active release
func (p *AWSProvider) BuildDelete(app, id string) (*structs.Build, error) {
	b, err := p.BuildGet(app, id)
	if err != nil {
		return nil, err
	}

	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	r, err := p.ReleaseGet(app, a.Release)
	if err != nil {
		return nil, err
	}

	if r.Build == id {
		return nil, fmt.Errorf("build is currently active")
	}

	// delete build item
	_, err = p.dynamodb().DeleteItem(&dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"id": {S: aws.String(id)},
		},
		TableName: aws.String(p.DynamoBuilds),
	})
	if err != nil {
		return b, err
	}

	// delete ECR images
	err = p.deleteImages(a, b)
	return b, err
}

// BuildExport exports a build artifact
func (p *AWSProvider) BuildExport(app, id string, w io.Writer) error {
	log := Logger.At("BuildExport").Start()

	build, err := p.BuildGet(app, id)
	if err != nil {
		log.Error(err)
		return err
	}

	a, err := p.AppGet(app)
	if err != nil {
		return err
	}

	services := []string{}

	switch a.Tags["Generation"] {
	case "2":
		r, err := p.ReleaseGet(app, build.Release)
		if err != nil {
			log.Error(err)
			return err
		}

		env := structs.Environment{}

		if err := env.Load([]byte(r.Env)); err != nil {
			log.Error(err)
			return err
		}

		m, err := manifest.Load([]byte(build.Manifest), manifest.Environment(env))
		if err != nil {
			log.Error(err)
			return err
		}

		for _, s := range m.Services {
			services = append(services, s.Name)
		}
	default:
		m, err := manifest1.Load([]byte(build.Manifest))
		if err != nil {
			log.Error(err)
			return fmt.Errorf("manifest error: %s", err)
		}

		for name := range m.Services {
			services = append(services, name)
		}
	}

	if len(services) < 1 {
		log.Errorf("no services found to export")
		return fmt.Errorf("no services found to export")
	}

	bjson, err := json.MarshalIndent(build, "", "  ")
	if err != nil {
		return err
	}

	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)

	dataHeader := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "build.json",
		Mode:     0600,
		Size:     int64(len(bjson)),
	}

	if err := tw.WriteHeader(dataHeader); err != nil {
		log.Error(err)
		return err
	}

	if _, err := tw.Write(bjson); err != nil {
		log.Error(err)
		return err
	}

	repo, err := p.appRepository(build.App)
	if err != nil {
		log.Error(err)
		return err
	}

	if err := p.dockerLogin(); err != nil {
		log.Error(err)
		return err
	}

	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		log.Error(err)
		return err
	}

	defer os.Remove(tmp)

	images := []string{}

	for _, service := range services {
		images = append(images, fmt.Sprintf("%s:%s.%s", repo.URI, service, build.Id))
	}

	errch := make(chan error, len(images))

	for _, image := range images {
		go func(img string) {
			log.Step("pull").Logf("image=%q", img)
			out, err := exec.Command("docker", "pull", img).CombinedOutput()
			if err != nil {
				errch <- fmt.Errorf("%s: %s\n", lastline(out), err.Error())
				return
			}

			errch <- nil
		}(image)
	}

	for i := 0; i < len(images); i++ {
		if err := <-errch; err != nil {
			return err
		}
	}

	name := fmt.Sprintf("%s.%s.tar", app, build.Id)
	file := filepath.Join(tmp, name)
	args := []string{"save", "-o", file}
	args = append(args, images...)

	log.Step("save").Logf("file=%q images=%d", file, len(images))
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s\n", strings.TrimSpace(string(out)), err.Error())
	}

	defer func() {
		if err := os.Remove(file); err != nil {
			log.Error(err)
		}
	}()

	stat, err := os.Stat(file)
	if err != nil {
		log.Error(err)
		return err
	}

	header := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     name,
		Mode:     0600,
		Size:     stat.Size(),
	}

	if err := tw.WriteHeader(header); err != nil {
		log.Error(err)
		return err
	}

	fd, err := os.Open(file)
	if err != nil {
		log.Error(err)
		return err
	}

	log.Step("copy").Logf("file=%q", file)
	if _, err := io.Copy(tw, fd); err != nil {
		log.Error(err)
		return err
	}

	if err := tw.Close(); err != nil {
		log.Error(err)
		return err
	}

	if err := gz.Close(); err != nil {
		log.Error(err)
		return err
	}

	log.Success()
	return nil
}

func (p *AWSProvider) BuildGet(app, id string) (*structs.Build, error) {
	req := &dynamodb.GetItemInput{
		ConsistentRead: aws.Bool(true),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {S: aws.String(id)},
		},
		TableName: aws.String(p.DynamoBuilds),
	}

	res, err := p.dynamodb().GetItem(req)
	if err != nil {
		return nil, err
	}

	if res.Item == nil {
		return nil, NoSuchBuild(id)
	}

	build := p.buildFromItem(res.Item)

	return build, nil
}

// BuildImport imports a build artifact
func (p *AWSProvider) BuildImport(app string, r io.Reader) (*structs.Build, error) {
	log := Logger.At("BuildImport").Namespace("app=%s", app).Start()

	var sourceBuild structs.Build

	// set up the new build
	targetBuild := structs.NewBuild(app)
	targetBuild.Started = time.Now()
	targetBuild.Status = "complete"

	if p.IsTest() {
		targetBuild.Id = "B12345"
	}

	repo, err := p.appRepository(app)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if err := p.dockerLogin(); err != nil {
		log.Error(err)
		return nil, err
	}

	gz, err := gzip.NewReader(r)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	tr := tar.NewReader(gz)

	var manifest imageManifest

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			return nil, err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}

		if header.Name == "build.json" {
			var buf bytes.Buffer
			io.Copy(&buf, tr)

			if err := json.Unmarshal(buf.Bytes(), &sourceBuild); err != nil {
				log.Error(err)
				return nil, err
			}

			targetBuild.Id = sourceBuild.Id

			_, err := p.BuildGet(app, sourceBuild.Id)
			if _, ok := err.(NoSuchBuild); err != nil && !ok {
				return nil, err
			}
			if err == nil {
				// build id already exists
				targetBuild.Id = generateId("B", 10)
			}
		}

		if strings.HasSuffix(header.Name, ".tar") {
			log.Step("load").Logf("tar=%q", header.Name)

			cmd := exec.Command("docker", "load")

			pr, pw := io.Pipe()
			tee := io.TeeReader(tr, pw)
			outb := &bytes.Buffer{}

			cmd.Stdin = pr
			cmd.Stdout = outb

			if err := cmd.Start(); err != nil {
				log.Error(err)
				return nil, err
			}

			log.Step("manifest").Logf("tar=%q", header.Name)
			manifest, err = extractImageManifest(tee)
			if err != nil {
				log.Error(err)
				return nil, err
			}

			if err := pw.Close(); err != nil {
				log.Error(err)
				return nil, err
			}

			if err := cmd.Wait(); err != nil {
				out := strings.TrimSpace(outb.String())
				return nil, log.Errorf("%s: %s\n", out, err.Error())
			}

			if len(manifest) == 0 {
				log.Errorf("invalid image manifest: no data")
				return nil, fmt.Errorf("invalid image manifest: no data")
			}
		}
	}

	for _, tags := range manifest {
		for _, image := range tags.RepoTags {
			fps := strings.Split(image, ":")[1]
			ps := strings.Split(fps, ".")[0]
			target := fmt.Sprintf("%s:%s.%s", repo.URI, ps, targetBuild.Id)

			log.Step("tag").Logf("from=%q to=%q", image, target)
			if out, err := exec.Command("docker", "tag", image, target).CombinedOutput(); err != nil {
				return nil, log.Error(fmt.Errorf("%s: %s\n", lastline(out), err.Error()))
			}

			log.Step("push").Logf("to=%q", target)
			if out, err := exec.Command("docker", "push", target).CombinedOutput(); err != nil {
				return nil, log.Error(fmt.Errorf("%s: %s\n", lastline(out), err.Error()))
			}
		}
	}

	targetBuild.Description = sourceBuild.Description
	targetBuild.Ended = time.Now()
	targetBuild.Logs = sourceBuild.Logs
	targetBuild.Manifest = sourceBuild.Manifest

	if err := p.buildSave(targetBuild); err != nil {
		log.Error(err)
		return nil, err
	}

	rr, err := p.ReleaseCreate(app, structs.ReleaseCreateOptions{Build: options.String(targetBuild.Id)})
	if err != nil {
		return nil, log.Error(err)
	}

	targetBuild.Release = rr.Id

	if err := p.buildSave(targetBuild); err != nil {
		log.Error(err)
		return nil, err
	}

	log.Successf("build=%q release=%q", targetBuild.Id, rr.Id)

	return targetBuild, nil
}

// BuildLogs streams the logs for a Build to an io.Writer
func (p *AWSProvider) BuildLogs(app, id string, opts structs.LogsOptions) (io.ReadCloser, error) {
	b, err := p.BuildGet(app, id)
	if err != nil {
		return nil, err
	}

	switch b.Status {
	case "running":
		task, err := p.describeTask(b.Tags["task"])
		if err != nil {
			return nil, err
		}

		ci, err := p.containerInstance(*task.ContainerInstanceArn)
		if err != nil {
			return nil, err
		}

		dc, err := p.dockerInstance(*ci.Ec2InstanceId)
		if err != nil {
			return nil, err
		}

		cs, err := dc.ListContainers(docker.ListContainersOptions{
			All: true,
			Filters: map[string][]string{
				"label": {fmt.Sprintf("com.amazonaws.ecs.task-arn=%s", *task.TaskArn)},
			},
		})
		if err != nil {
			return nil, err
		}
		if len(cs) != 1 {
			return nil, fmt.Errorf("could not find container for task: %s", *task.TaskArn)
		}

		r, w := io.Pipe()

		go func() {
			defer w.Close()
			dc.Logs(docker.LogsOptions{
				Container:         cs[0].ID,
				OutputStream:      w,
				ErrorStream:       w,
				InactivityTimeout: 20 * time.Minute,
				Follow:            true,
				Stdout:            true,
				Stderr:            true,
			})
		}()

		return r, nil
	default:
		u, err := url.Parse(b.Logs)
		if err != nil {
			return nil, err
		}

		switch u.Scheme {
		case "object":
			return p.ObjectFetch(app, u.Path)
		default:
			return ioutil.NopCloser(strings.NewReader(b.Logs)), nil
		}
	}

	return nil, fmt.Errorf("unreachable")
}

// BuildList returns a list of the latest builds, with the length specified in limit
func (p *AWSProvider) BuildList(app string, opts structs.BuildListOptions) (structs.Builds, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	if opts.Count == nil {
		opts.Count = aws.Int(10)
	}

	req := &dynamodb.QueryInput{
		KeyConditions: map[string]*dynamodb.Condition{
			"app": {
				AttributeValueList: []*dynamodb.AttributeValue{{S: aws.String(a.Name)}},
				ComparisonOperator: aws.String("EQ"),
			},
		},
		IndexName:        aws.String("app.created"),
		Limit:            aws.Int64(int64(*opts.Count)),
		ScanIndexForward: aws.Bool(false),
		TableName:        aws.String(p.DynamoBuilds),
	}

	res, err := p.dynamodb().Query(req)
	if err != nil {
		return nil, err
	}

	builds := make(structs.Builds, len(res.Items))

	for i, item := range res.Items {
		builds[i] = *p.buildFromItem(item)
	}

	return builds, nil
}

func (p *AWSProvider) BuildUpdate(app, id string, opts structs.BuildUpdateOptions) (*structs.Build, error) {
	b, err := p.BuildGet(app, id)
	if err != nil {
		return nil, err
	}

	if opts.Ended != nil {
		b.Ended = *opts.Ended
	}

	if opts.Logs != nil {
		b.Logs = *opts.Logs
	}

	if opts.Manifest != nil {
		b.Manifest = *opts.Manifest
	}

	if opts.Release != nil {
		b.Release = *opts.Release
	}

	if opts.Started != nil {
		b.Started = *opts.Started
	}

	if opts.Status != nil {
		b.Status = *opts.Status
	}

	if err := p.buildSave(b); err != nil {
		return nil, err
	}

	return b, nil
}

func (p *AWSProvider) buildSave(b *structs.Build) error {
	_, err := p.AppGet(b.App)
	if err != nil {
		return err
	}

	if b.Id == "" {
		return fmt.Errorf("Id can not be blank")
	}

	if b.Started.IsZero() {
		b.Started = time.Now()
	}

	if p.IsTest() {
		b.Started = time.Unix(1473028693, 0).UTC()
		b.Ended = time.Unix(1473028892, 0).UTC()
	}

	req := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"id":      {S: aws.String(b.Id)},
			"app":     {S: aws.String(b.App)},
			"status":  {S: aws.String(b.Status)},
			"created": {S: aws.String(b.Started.Format(sortableTime))},
		},
		TableName: aws.String(p.DynamoBuilds),
	}

	if b.Description != "" {
		req.Item["description"] = &dynamodb.AttributeValue{S: aws.String(b.Description)}
	}

	if b.Manifest != "" {
		req.Item["manifest"] = &dynamodb.AttributeValue{S: aws.String(b.Manifest)}
	}

	if b.Logs != "" {
		req.Item["logs"] = &dynamodb.AttributeValue{S: aws.String(b.Logs)}
	}

	if b.Reason != "" {
		req.Item["reason"] = &dynamodb.AttributeValue{S: aws.String(b.Reason)}
	}

	if b.Release != "" {
		req.Item["release"] = &dynamodb.AttributeValue{S: aws.String(b.Release)}
	}

	if !b.Ended.IsZero() {
		req.Item["ended"] = &dynamodb.AttributeValue{S: aws.String(b.Ended.Format(sortableTime))}
	}

	if len(b.Tags) > 0 {
		tags, err := json.Marshal(b.Tags)
		if err != nil {
			return err
		}

		req.Item["tags"] = &dynamodb.AttributeValue{B: tags}
	}

	_, err = p.dynamodb().PutItem(req)

	return err
}

func (p *AWSProvider) buildAuth(build *structs.Build) (string, error) {
	type authEntry struct {
		Username string
		Password string
	}

	auth := map[string]authEntry{}

	registries, err := p.RegistryList()
	if err != nil {
		return "", err
	}

	for _, r := range registries {
		switch {
		case regexpECRHost.MatchString(r.Server):
			un, pw, err := p.authECR(r.Server, r.Username, r.Password)
			if err != nil {
				return "", err
			}

			auth[r.Server] = authEntry{
				Username: un,
				Password: pw,
			}
		default:
			auth[r.Server] = authEntry{
				Username: r.Username,
				Password: r.Password,
			}
		}
	}

	aid, err := p.accountId()
	if err != nil {
		return "", err
	}

	host := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", aid, p.Region)

	un, pw, err := p.authECR(host, "", "")
	if err != nil {
		return "", err
	}

	auth[host] = authEntry{
		Username: un,
		Password: pw,
	}

	data, err := json.Marshal(auth)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (p *AWSProvider) authECR(host, access, secret string) (string, string, error) {
	config := p.config()

	if access != "" {
		config.Credentials = credentials.NewStaticCredentials(access, secret, "")
	}

	if !regexpECRHost.MatchString(host) {
		return "", "", fmt.Errorf("invalid ECR hostname: %s", host)
	}

	registry := regexpECRHost.FindStringSubmatch(host)

	config.Region = &registry[2]

	e := ecr.New(session.New(), config)

	res, err := e.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{
		RegistryIds: []*string{aws.String(registry[1])},
	})

	if err != nil {
		return "", "", err
	}
	if len(res.AuthorizationData) != 1 {
		return "", "", fmt.Errorf("no authorization data")
	}

	token, err := base64.StdEncoding.DecodeString(*res.AuthorizationData[0].AuthorizationToken)
	if err != nil {
		return "", "", err
	}

	parts := strings.SplitN(string(token), ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid auth data")
	}

	return parts[0], parts[1], nil
}

func (p *AWSProvider) runBuild(build *structs.Build, method, url string, opts structs.BuildCreateOptions) error {
	log := Logger.At("runBuild").Namespace("method=%q url=%q", method, url).Start()

	br, err := p.stackResource(p.Rack, "ApiBuildTasks")
	if err != nil {
		log.Error(err)
		return err
	}

	td := *br.PhysicalResourceId

	auth, err := p.buildAuth(build)
	if err != nil {
		return err
	}

	aid, err := p.accountId()
	if err != nil {
		return err
	}

	reg, err := p.appResource(build.App, "Registry")
	if err != nil {
		// handle generation 1
		if strings.HasPrefix(err.Error(), "resource not found") {
			app, err := p.AppGet(build.App)
			if err != nil {
				return err
			}

			reg = app.Outputs["RegistryRepository"]
		} else {
			return err
		}
	}

	a, err := p.AppGet(build.App)
	if err != nil {
		return err
	}

	push := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:{service}.{build}", aid, p.Region, reg)

	switch a.Tags["Generation"] {
	case "2":
		push = fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s", aid, p.Region, reg)
	}

	config := ""

	if opts.Config != nil {
		config = *opts.Config

	}

	req := &ecs.RunTaskInput{
		Cluster:        aws.String(p.BuildCluster),
		Count:          aws.Int64(1),
		StartedBy:      aws.String(fmt.Sprintf("convox.%s", build.App)),
		TaskDefinition: aws.String(td),
		Overrides: &ecs.TaskOverride{
			ContainerOverrides: []*ecs.ContainerOverride{
				{
					Name: aws.String("build"),
					Command: []*string{
						aws.String("build"),
						aws.String("-method"), aws.String(method),
						aws.String("-cache"), aws.String(fmt.Sprintf("%t", opts.Cache)),
					},
					Environment: []*ecs.KeyValuePair{
						{
							Name:  aws.String("BUILD_APP"),
							Value: aws.String(build.App),
						},
						{
							Name:  aws.String("BUILD_AUTH"),
							Value: aws.String(auth),
						},
						{
							Name:  aws.String("BUILD_CONFIG"),
							Value: aws.String(config),
						},
						{
							Name:  aws.String("BUILD_GENERATION"),
							Value: aws.String(a.Tags["Generation"]),
						},
						{
							Name:  aws.String("BUILD_ID"),
							Value: aws.String(build.Id),
						},
						{
							Name:  aws.String("BUILD_PUSH"),
							Value: aws.String(push),
						},
						{
							Name:  aws.String("BUILD_URL"),
							Value: aws.String(url),
						},
						{
							Name:  aws.String("HTTP_PROXY"),
							Value: aws.String(os.Getenv("HTTP_PROXY")),
						},
						{
							Name:  aws.String("RELEASE"),
							Value: aws.String(build.Id),
						},
					},
				},
			},
		},
	}

	task, err := p.runTask(req)
	if err != nil {
		log.Error(err)
		return err
	}

	b, err := p.BuildGet(build.App, build.Id)
	if err != nil {
		return err
	}

	b.Status = "running"

	b.Tags["task"] = *task.TaskArn

	if err := p.buildSave(b); err != nil {
		return err
	}

	if _, err := p.waitForTask(*task.TaskArn); err != nil {
		return err
	}

	if err := p.waitForContainer(task); err != nil {
		return err
	}

	return nil
}

func (p *AWSProvider) waitForContainer(task *ecs.Task) error {
	ci, err := p.containerInstance(*task.ContainerInstanceArn)
	if err != nil {
		return err
	}

	dc, err := p.dockerInstance(*ci.Ec2InstanceId)
	if err != nil {
		return err
	}

	tick := time.Tick(1 * time.Second)
	timeout := time.After(60 * time.Second)

	for {
		select {
		case <-tick:
			cs, err := dc.ListContainers(docker.ListContainersOptions{
				All: true,
				Filters: map[string][]string{
					"label": {fmt.Sprintf("com.amazonaws.ecs.task-arn=%s", *task.TaskArn)},
				},
			})
			if err != nil {
				return err
			}
			if len(cs) > 0 {
				return nil
			}
		case <-timeout:
			return fmt.Errorf("timeout waiting for container")
		}
	}

	return nil
}

// buildFromItem populates a Build struct from a DynamoDB Item
func (p *AWSProvider) buildFromItem(item map[string]*dynamodb.AttributeValue) *structs.Build {
	id := coalesce(item["id"], "")
	started, _ := time.Parse(sortableTime, coalesce(item["created"], ""))
	ended, _ := time.Parse(sortableTime, coalesce(item["ended"], ""))

	tags := map[string]string{}

	if item["tags"] != nil {
		json.Unmarshal(item["tags"].B, &tags)
	}

	return &structs.Build{
		Id:          id,
		App:         coalesce(item["app"], ""),
		Description: coalesce(item["description"], ""),
		Manifest:    coalesce(item["manifest"], ""),
		Logs:        coalesce(item["logs"], ""),
		Release:     coalesce(item["release"], ""),
		Reason:      coalesce(item["reason"], ""),
		Status:      coalesce(item["status"], ""),
		Started:     started,
		Ended:       ended,
		Tags:        tags,
	}
}

// deleteImages generates a list of fully qualified URLs for images for every process type
// in the build manifest then deletes them.
// Image URLs that point to ECR, e.g. 826133048.dkr.ecr.us-east-1.amazonaws.com/myapp-zridvyqapp:web.BSUSBFCUCSA,
// are deleted with the ECR BatchDeleteImage API.
// Image URLs that point to the convox-hosted registry, e.g. convox-826133048.us-east-1.elb.amazonaws.com:5000/myapp-web:BSUSBFCUCSA,
// are not yet supported and return an error.
func (p *AWSProvider) deleteImages(a *structs.App, b *structs.Build) error {

	m, err := manifest1.Load([]byte(b.Manifest))
	if err != nil {
		return err
	}

	// failed builds could have an empty manifest
	if len(m.Services) == 0 {
		return nil
	}

	urls := []string{}

	for name := range m.Services {
		urls = append(urls, p.registryTag(a, name, b.Id))
	}

	imageIds := []*ecr.ImageIdentifier{}
	registryId := ""
	repositoryName := ""

	for _, url := range urls {
		if match := regexpECRImage.FindStringSubmatch(url); match != nil {
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

func (p *AWSProvider) dockerLogin() error {
	log := Logger.At("dockerLogin").Start()

	tres, err := p.ecr().GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})
	if err != nil {
		log.Error(err)
		return err
	}
	if len(tres.AuthorizationData) != 1 {
		log.Errorf("no authorization data")
		return fmt.Errorf("no authorization data")
	}

	auth, err := base64.StdEncoding.DecodeString(*tres.AuthorizationData[0].AuthorizationToken)
	if err != nil {
		log.Error(err)
		return err
	}

	authParts := strings.SplitN(string(auth), ":", 2)
	if len(authParts) != 2 {
		log.Errorf("invalid auth data")
		return fmt.Errorf("invalid auth data")
	}

	registry, err := url.Parse(*tres.AuthorizationData[0].ProxyEndpoint)
	if err != nil {
		return log.Error(err)
	}

	log.Step("login").Logf("host=%q user=%q", registry.Host, authParts[0])
	out, err := exec.Command("docker", "login", "-u", authParts[0], "-p", authParts[1], registry.Host).CombinedOutput()
	if err != nil {
		return log.Error(fmt.Errorf("%s: %s\n", lastline(out), err.Error()))
	}

	log.Success()
	return nil
}

type imageManifest []struct {
	RepoTags []string
}

func extractImageManifest(r io.Reader) (imageManifest, error) {
	mtr := tar.NewReader(r)

	var manifest imageManifest

	for {
		mh, err := mtr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if mh.Name == "manifest.json" {
			var mdata bytes.Buffer

			if _, err := io.Copy(&mdata, mtr); err != nil {
				return nil, err
			}

			if err := json.Unmarshal(mdata.Bytes(), &manifest); err != nil {
				return nil, err
			}

			return manifest, nil
		}
	}

	return nil, fmt.Errorf("unable to locate manifest")
}

func (p *AWSProvider) registryTag(a *structs.App, serviceName, buildID string) string {
	tag := fmt.Sprintf("%s/%s-%s:%s", p.RegistryHost, a.Name, serviceName, buildID)

	if registryId := a.Outputs["RegistryId"]; registryId != "" {
		tag = fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s.%s", registryId, p.Region, a.Outputs["RegistryRepository"], serviceName, buildID)
	}

	return tag
}

func (p *AWSProvider) buildsDeleteAll(app *structs.App) error {
	// query dynamo for all builds belonging to app
	qi := &dynamodb.QueryInput{
		KeyConditionExpression: aws.String("app = :app"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":app": {S: aws.String(app.Name)},
		},
		IndexName: aws.String("app.created"),
		TableName: aws.String(p.DynamoBuilds),
	}

	res, err := p.dynamodb().Query(qi)
	if err != nil {
		return err
	}

	// collect builds IDs to delete
	wrs := []*dynamodb.WriteRequest{}
	for _, item := range res.Items {
		b := p.buildFromItem(item)

		wr := &dynamodb.WriteRequest{
			DeleteRequest: &dynamodb.DeleteRequest{
				Key: map[string]*dynamodb.AttributeValue{
					"id": {
						S: aws.String(b.Id),
					},
				},
			},
		}

		wrs = append(wrs, wr)
	}

	return p.dynamoBatchDeleteItems(wrs, p.DynamoBuilds)
}
