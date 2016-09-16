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
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	docker "github.com/fsouza/go-dockerclient"

	"github.com/convox/rack/api/crypt"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/manifest"
)

var regexpECR = regexp.MustCompile(`(\d+)\.dkr\.ecr\.([^.]+)\.amazonaws\.com\/([^:]+):([^ ]+)`)

func (p *AWSProvider) BuildCreate(app, method, url string, opts structs.BuildOptions) (*structs.Build, error) {
	log := Logger.At("BuildCreate").Namespace("app=%q method=%q url=%q", app, method, url).Start()

	_, err := p.AppGet(app)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	b := structs.NewBuild(app)
	b.Description = opts.Description
	b.Started = time.Now()

	if err := p.BuildSave(b); err != nil {
		log.Error(err)
		return nil, err
	}

	if err := p.runBuild(b, method, url, opts); err != nil {
		log.Error(err)
		return nil, err
	}

	p.EventSend(&structs.Event{
		Action: "build:create",
		Data: map[string]string{
			"app": b.App,
			"id":  b.Id,
		},
	}, nil)

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
			"id": &dynamodb.AttributeValue{S: aws.String(id)},
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

	m, err := manifest.Load([]byte(build.Manifest))
	if err != nil {
		log.Error(err)
		return fmt.Errorf("manifest error: %s", err)
	}

	if len(m.Services) < 1 {
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

	if err := p.dockerLogin(repo); err != nil {
		log.Error(err)
		return err
	}

	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		log.Error(err)
		return err
	}

	defer os.Remove(tmp)

	for service := range m.Services {
		image := fmt.Sprintf("%s:%s.%s", repo.URI, service, build.Id)
		file := filepath.Join(tmp, fmt.Sprintf("%s.%s.tar", service, build.Id))

		log.Step("pull").Logf("image=%q", image)
		out, err := exec.Command("docker", "pull", image).CombinedOutput()
		if err != nil {
			log.Error(fmt.Errorf(lastline(out)))
			return err
		}

		log.Step("save").Logf("image=%q file=%q", image, file)
		out, err = exec.Command("docker", "save", "-o", file, image).CombinedOutput()
		if err != nil {
			log.Error(fmt.Errorf(lastline(out)))
			return err
		}

		stat, err := os.Stat(file)
		if err != nil {
			log.Error(err)
			return err
		}

		header := &tar.Header{
			Typeflag: tar.TypeReg,
			Name:     fmt.Sprintf("%s.%s.tar", service, build.Id),
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

		if err := os.Remove(file); err != nil {
			log.Error(err)
			return err
		}
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
			"id": &dynamodb.AttributeValue{S: aws.String(id)},
		},
		TableName: aws.String(p.DynamoBuilds),
	}

	res, err := p.dynamodb().GetItem(req)
	if err != nil {
		return nil, err
	}

	if res.Item == nil {
		return nil, fmt.Errorf("no such build: %s", id)
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
	targetBuild.Description = fmt.Sprintf("imported")
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

	if err := p.dockerLogin(repo); err != nil {
		log.Error(err)
		return nil, err
	}

	gz, err := gzip.NewReader(r)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	tr := tar.NewReader(gz)

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
			manifest, err := extractImageManifest(tee)
			if err != nil {
				log.Error(err)
				return nil, err
			}

			if err := pw.Close(); err != nil {
				log.Error(err)
				return nil, err
			}

			if err := cmd.Wait(); err != nil {
				log.Errorf(lastline(outb.Bytes()))
				return nil, err
			}

			if len(manifest) != 1 || len(manifest[0].RepoTags) != 1 {
				log.Errorf("invalid image manifest")
				return nil, fmt.Errorf("invalid image manifest")
			}

			image := manifest[0].RepoTags[0]
			ps := strings.Split(header.Name, ".")[0]
			target := fmt.Sprintf("%s:%s.%s", repo.URI, ps, targetBuild.Id)

			log.Step("tag").Logf("from=%q to=%q", image, target)
			if out, err := exec.Command("docker", "tag", image, target).CombinedOutput(); err != nil {
				log.Errorf(lastline(out))
				return nil, err
			}

			log.Step("push").Logf("to=%q", target)
			if out, err := exec.Command("docker", "push", target).CombinedOutput(); err != nil {
				log.Errorf(lastline(out))
				return nil, err
			}
		}
	}

	env, err := p.EnvironmentGet(app)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	release := structs.NewRelease(app)

	if p.IsTest() {
		release.Id = "R23456"
	}

	targetBuild.Ended = time.Now()
	targetBuild.Logs = sourceBuild.Logs
	targetBuild.Manifest = sourceBuild.Manifest
	targetBuild.Release = release.Id

	if err := p.BuildSave(targetBuild); err != nil {
		log.Error(err)
		return nil, err
	}

	release.Env = env.Raw()
	release.Build = targetBuild.Id
	release.Manifest = targetBuild.Manifest

	if err := p.ReleaseSave(release); err != nil {
		log.Error(err)
		return nil, err
	}

	log.Successf("build=%q release=%q", targetBuild.Id, release.Id)

	return targetBuild, nil
}

// BuildLogs streams the logs for a Build to an io.Writer
func (p *AWSProvider) BuildLogs(app, id string, w io.Writer) error {
	log := Logger.At("BuildLogs").Namespace("app=%q id=%q", app, id).Start()

	b, err := p.BuildGet(app, id)
	if err != nil {
		return err
	}

	switch b.Status {
	case "running":
		task, err := p.describeTask(b.Tags["task"])
		if err != nil {
			return err
		}

		ci, err := p.containerInstance(*task.ContainerInstanceArn)
		if err != nil {
			return err
		}

		dc, err := p.dockerInstance(*ci.Ec2InstanceId)
		if err != nil {
			return err
		}

		cs, err := dc.ListContainers(docker.ListContainersOptions{
			All: true,
			Filters: map[string][]string{
				"label": []string{fmt.Sprintf("com.amazonaws.ecs.task-arn=%s", *task.TaskArn)},
			},
		})
		if err != nil {
			return err
		}
		if len(cs) != 1 {
			return fmt.Errorf("could not find container for task: %s", *task.TaskArn)
		}

		err = dc.Logs(docker.LogsOptions{
			Container:         cs[0].ID,
			OutputStream:      w,
			ErrorStream:       w,
			InactivityTimeout: 20 * time.Minute,
			Follow:            true,
			Stdout:            true,
			Stderr:            true,
		})
		if err != nil {
			return err
		}
	default:
		u, err := url.Parse(b.Logs)
		if err != nil {
			return err
		}

		switch u.Scheme {
		case "object":
			r, err := p.ObjectFetch(u.Path)
			if err != nil {
				return err
			}

			if _, err := io.Copy(w, r); err != nil {
				return err
			}
		default:
			if _, err := w.Write([]byte(b.Logs)); err != nil {
				return err
			}
		}
	}

	log.Success()
	return nil
}

// BuildList returns a list of the latest builds, with the length specified in limit
func (p *AWSProvider) BuildList(app string, limit int64) (structs.Builds, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	req := &dynamodb.QueryInput{
		KeyConditions: map[string]*dynamodb.Condition{
			"app": &dynamodb.Condition{
				AttributeValueList: []*dynamodb.AttributeValue{&dynamodb.AttributeValue{S: aws.String(a.Name)}},
				ComparisonOperator: aws.String("EQ"),
			},
		},
		IndexName:        aws.String("app.created"),
		Limit:            aws.Int64(limit),
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

func (p *AWSProvider) BuildRelease(b *structs.Build) (*structs.Release, error) {
	releases, err := p.ReleaseList(b.App, 20)
	if err != nil {
		return nil, err
	}

	r := structs.NewRelease(b.App)
	newId := r.Id

	if len(releases) > 0 {
		r = &releases[0]
	}

	r.Id = newId
	r.Created = time.Time{}
	r.Build = b.Id
	r.Manifest = b.Manifest

	err = p.ReleaseSave(r)
	if err != nil {
		return r, err
	}

	b.Release = r.Id
	err = p.BuildSave(b)

	if err == nil {
		p.EventSend(&structs.Event{
			Action: "release:create",
			Data: map[string]string{
				"app": r.App,
				"id":  r.Id,
			},
		}, nil)
	}

	return r, err
}

// BuildSave creates or updates a build item in DynamoDB. It takes an optional
// bucket argument, which if set indicates to PUT Log data into S3
func (p *AWSProvider) BuildSave(b *structs.Build) error {
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
			"id":      &dynamodb.AttributeValue{S: aws.String(b.Id)},
			"app":     &dynamodb.AttributeValue{S: aws.String(b.App)},
			"status":  &dynamodb.AttributeValue{S: aws.String(b.Status)},
			"created": &dynamodb.AttributeValue{S: aws.String(b.Started.Format(sortableTime))},
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
	r, err := p.ObjectFetch("env")
	if err != nil && !ErrorNotFound(err) {
		return "", err
	}

	data := []byte("{}")

	if r != nil {
		defer r.Close()

		d, err := ioutil.ReadAll(r)
		if err != nil {
			return "", err
		}
		data = d

		if p.EncryptionKey != "" {
			cr := crypt.New(p.Region, p.Access, p.Secret)

			if d, err := cr.Decrypt(p.EncryptionKey, data); err == nil {
				data = d
			}
		}
	}

	var env map[string]string
	err = json.Unmarshal(data, &env)
	if err != nil {
		return "", err
	}

	type authEntry struct {
		Username string
		Password string
	}

	auth := map[string]authEntry{}

	if ea, ok := env["DOCKER_AUTH_DATA"]; ok {
		if err := json.Unmarshal([]byte(ea), &auth); err != nil {
			return "", err
		}
	}

	a, err := p.AppGet(build.App)
	if err != nil {
		return "", err
	}

	res, err := p.ecr().GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{
		RegistryIds: []*string{aws.String(a.Outputs["RegistryId"])},
	})
	if err != nil {
		return "", err
	}
	if len(res.AuthorizationData) != 1 {
		return "", fmt.Errorf("no authorization data")
	}

	token, err := base64.StdEncoding.DecodeString(*res.AuthorizationData[0].AuthorizationToken)
	if err != nil {
		return "", err
	}

	parts := strings.SplitN(string(token), ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid auth data")
	}

	host := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", a.Outputs["RegistryId"], p.Region)

	auth[host] = authEntry{
		Username: parts[0],
		Password: parts[1],
	}

	data, err = json.Marshal(auth)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (p *AWSProvider) runBuild(build *structs.Build, method, url string, opts structs.BuildOptions) error {
	log := Logger.At("runBuild").Namespace("method=%q url=%q", method, url).Start()

	br, err := p.stackResource(p.Rack, "RackBuildTasks")
	if err != nil {
		log.Error(err)
		return err
	}

	a, err := p.AppGet(build.App)
	if err != nil {
		return err
	}

	td := *br.PhysicalResourceId

	auth, err := p.buildAuth(build)
	if err != nil {
		return err
	}

	push := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:{service}.{build}", a.Outputs["RegistryId"], p.Region, a.Outputs["RegistryRepository"])

	req := &ecs.RunTaskInput{
		Cluster:        aws.String(p.Cluster),
		Count:          aws.Int64(1),
		StartedBy:      aws.String(fmt.Sprintf("convox.%s", build.App)),
		TaskDefinition: aws.String(td),
		Overrides: &ecs.TaskOverride{
			ContainerOverrides: []*ecs.ContainerOverride{
				&ecs.ContainerOverride{
					Name: aws.String("build"),
					Command: []*string{
						aws.String("build"),
						aws.String("-method"), aws.String(method),
						aws.String("-cache"), aws.String(fmt.Sprintf("%t", opts.Cache)),
					},
					Environment: []*ecs.KeyValuePair{
						&ecs.KeyValuePair{
							Name:  aws.String("BUILD_APP"),
							Value: aws.String(build.App),
						},
						&ecs.KeyValuePair{
							Name:  aws.String("BUILD_AUTH"),
							Value: aws.String(auth),
						},
						&ecs.KeyValuePair{
							Name:  aws.String("BUILD_CONFIG"),
							Value: aws.String(opts.Config),
						},
						&ecs.KeyValuePair{
							Name:  aws.String("BUILD_ID"),
							Value: aws.String(build.Id),
						},
						&ecs.KeyValuePair{
							Name:  aws.String("BUILD_PUSH"),
							Value: aws.String(push),
						},
						&ecs.KeyValuePair{
							Name:  aws.String("BUILD_RELEASE"),
							Value: aws.String(a.Release),
						},
						&ecs.KeyValuePair{
							Name:  aws.String("BUILD_URL"),
							Value: aws.String(url),
						},
						&ecs.KeyValuePair{
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

	if err := p.BuildSave(b); err != nil {
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
					"label": []string{fmt.Sprintf("com.amazonaws.ecs.task-arn=%s", *task.TaskArn)},
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

	m, err := manifest.Load([]byte(b.Manifest))
	if err != nil {
		return err
	}

	// failed builds could have an empty manifest
	if len(m.Services) == 0 {
		return nil
	}

	urls := []string{}

	for name, _ := range m.Services {
		urls = append(urls, p.registryTag(a, name, b.Id))
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

func (p *AWSProvider) dockerLogin(repo *appRepository) error {
	log := Logger.At("dockerLogin").Namespace("repo=%q", repo.URI).Start()

	log.Step("token").Logf("id=%q", repo.ID)
	tres, err := p.ecr().GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{
		RegistryIds: []*string{aws.String(repo.ID)},
	})
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

	log.Step("login").Logf("host=%q user=%q", repo.URI, authParts[0])
	out, err := exec.Command("docker", "login", "-u", authParts[0], "-p", authParts[1], repo.URI).CombinedOutput()
	if err != nil {
		log.Errorf(lastline(out))
		return err
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
			":app": &dynamodb.AttributeValue{S: aws.String(app.Name)},
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
					"id": &dynamodb.AttributeValue{
						S: aws.String(b.Id),
					},
				},
			},
		}

		wrs = append(wrs, wr)
	}

	return p.dynamoBatchDeleteItems(wrs, p.DynamoBuilds)
}
