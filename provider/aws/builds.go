package aws

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/s3"
	"gopkg.in/yaml.v2"

	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/manifest"
)

var regexpECR = regexp.MustCompile(`(\d+)\.dkr\.ecr\.([^.]+)\.amazonaws\.com\/([^:]+):([^ ]+)`)

func (p *AWSProvider) BuildCopy(srcApp, id, destApp string) (*structs.Build, error) {
	srcA, err := p.AppGet(srcApp)
	if err != nil {
		return nil, err
	}

	srcB, err := p.BuildGet(srcApp, id)
	if err != nil {
		return nil, err
	}

	destA, err := p.AppGet(destApp)
	if err != nil {
		return nil, err
	}

	// make a .tgz file that is the source build manifest
	// with build directives removed, and image directives pointing to
	// fully qualified URLs of source build images
	var m manifest.Manifest
	err = yaml.Unmarshal([]byte(srcB.Manifest), &m)
	if err != nil {
		return nil, err
	}

	for name, entry := range m.Services {
		entry.Build.Context = ""
		entry.Image = p.registryTag(srcA, name, srcB.Id)
		m.Services[name] = entry
	}

	data, err := m.Raw()
	if err != nil {
		return nil, err
	}

	dir, err := ioutil.TempDir("", "source")
	if err != nil {
		return nil, err
	}

	err = os.Chmod(dir, 0755)
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(filepath.Join(dir, "docker-compose.yml"), data, 0644)
	if err != nil {
		return nil, err
	}

	tgz, err := createTarball(dir)
	if err != nil {
		return nil, err
	}

	// Build .tgz in context of destApp
	return p.BuildCreateTar(destA.Name, bytes.NewReader(tgz), "docker-compose.yml", fmt.Sprintf("Copy of %s %s", srcA.Name, srcB.Id), false)
}

func (p *AWSProvider) BuildCreateIndex(app string, index structs.Index, manifest, description string, cache bool) (*structs.Build, error) {
	dir, err := ioutil.TempDir("", "source")
	if err != nil {
		return nil, err
	}

	err = os.Chmod(dir, 0755)
	if err != nil {
		return nil, err
	}

	err = p.IndexDownload(&index, dir)
	if err != nil {
		return nil, err
	}

	tgz, err := createTarball(dir)
	if err != nil {
		return nil, err
	}

	return p.BuildCreateTar(app, bytes.NewReader(tgz), manifest, description, cache)
}

func (p *AWSProvider) BuildCreateRepo(app, url, manifest, description string, cache bool) (*structs.Build, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	b := structs.NewBuild(app)
	b.Description = description

	err = p.BuildSave(b)
	if err != nil {
		return nil, err
	}

	args := p.buildArgs(a, b, url)

	env, err := p.buildEnv(a, b, manifest, cache)
	if err != nil {
		return b, err
	}

	err = p.buildRun(a, b, args, env, nil)

	// build create is now complete or failed
	p.EventSend(&structs.Event{
		Action: "build:create",
		Data: map[string]string{
			"app": b.App,
			"id":  b.Id,
		},
	}, err)

	return b, err
}

func (p *AWSProvider) BuildCreateTar(app string, src io.Reader, manifest, description string, cache bool) (*structs.Build, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return nil, err
	}

	b := structs.NewBuild(app)
	b.Description = description

	err = p.BuildSave(b)
	if err != nil {
		return nil, err
	}

	// TODO: save the tarball in s3?

	args := p.buildArgs(a, b, "-")

	env, err := p.buildEnv(a, b, manifest, cache)
	if err != nil {
		return b, err
	}

	err = p.buildRun(a, b, args, env, src)

	p.EventSend(&structs.Event{
		Action: "build:create",
		Data: map[string]string{
			"app": b.App,
			"id":  b.Id,
		},
	}, err)

	return b, err
}

// BuildDelete deletes the build specified by id belonging to app
// Care should be taken as this could delete the build used by the active release
func (p *AWSProvider) BuildDelete(app, id string) (*structs.Build, error) {
	b, err := p.BuildGet(app, id)
	if err != nil {
		return b, err
	}

	a, err := p.AppGet(app)
	if err != nil {
		return b, err
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
func (p *AWSProvider) BuildImport(appName string, r io.Reader) (*structs.Build, error) {

	build, images, err := readImportArtifact(r)
	if err != nil {
		return nil, err
	}

	app, err := p.AppGet(appName)
	if err != nil {
		return nil, err
	}

	// load the images to repo
	for _, img := range images {
		cmd := exec.Command("docker", "load")
		cmd.Stdin = bytes.NewReader(img)

		out, err := cmd.Output()
		output := string(out)
		if err != nil {
			return nil, fmt.Errorf("docker load failed: %s", err)
		}

		fmt.Printf("fn=BuildImport at=DockerLoad level=info msg=\"%s\"\n", output)

		loadPrefix := "Loaded image: "
		if !strings.HasPrefix(output, loadPrefix) {
			return nil, fmt.Errorf("unexpected docker load output: %s", output)
		}

		imageSplit := strings.Split(output, loadPrefix)
		if len(imageSplit) < 2 {
			return nil, fmt.Errorf("docker load output split failed: %s", output)
		}

		tag := strings.Split(imageSplit[1], ":")[1]

		repo, err := p.appRepository(app.Name)
		if err != nil {
			return nil, err
		}

		newName := fmt.Sprintf("%s:%s", repo.URI, strings.TrimSpace(tag))
		cmd = exec.Command("docker", "tag", strings.TrimSpace(imageSplit[1]), newName)

		out, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("docker tag failed: %s", err)
		}

		//TODO: Remove the orignal import tag (from imageSplit) if it didn't originally exist

		fmt.Printf("fn=BuildImport at=DockerTag level=info msg=\"new tag %s\"\n", newName)

		cmd = exec.Command("docker", "push", newName)
		out, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("docker push failed: %s", err)
		}
	}

	oldEnv, err := p.EnvironmentGet(app.Name)
	if err != nil {
		return nil, err
	}

	release := structs.NewRelease(app.Name)
	release.Env = oldEnv.Raw()
	release.Build = build.Id
	release.Manifest = build.Manifest

	err = p.ReleaseSave(release, app.Outputs["Settings"], app.Parameters["Key"])
	if err != nil {
		return nil, err
	}

	build.Release = release.Id
	build.App = app.Name
	err = p.BuildSave(build)
	if err != nil {
		return nil, err
	}

	return build, nil
}

// BuildLogs gets a Build's logs from S3. If there is no log file in S3, that is not an error.
func (p *AWSProvider) BuildLogs(app, id string) (string, error) {
	a, err := p.AppGet(app)
	if err != nil {
		return "", err
	}

	key := fmt.Sprintf("builds/%s.log", id)

	req := &s3.GetObjectInput{
		Bucket: aws.String(a.Outputs["Settings"]),
		Key:    aws.String(key),
	}

	res, err := p.s3().GetObject(req)
	if err != nil {
		if awsError(err) == "NoSuchKey" {
			return "", nil
		}
		return "", err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
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

	a, err := p.AppGet(b.App)
	if err != nil {
		return r, err
	}

	err = p.ReleaseSave(r, a.Outputs["Settings"], a.Parameters["Key"])
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
	a, err := p.AppGet(b.App)
	if err != nil {
		return err
	}

	if b.Id == "" {
		return fmt.Errorf("Id can not be blank")
	}

	if b.Started.IsZero() {
		b.Started = time.Now()
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

	if b.Release != "" {
		req.Item["release"] = &dynamodb.AttributeValue{S: aws.String(b.Release)}
	}

	if !b.Ended.IsZero() {
		req.Item["ended"] = &dynamodb.AttributeValue{S: aws.String(b.Ended.Format(sortableTime))}
	}

	if b.Logs != "" {
		_, err := p.s3().PutObject(&s3.PutObjectInput{
			Body:          bytes.NewReader([]byte(b.Logs)),
			Bucket:        aws.String(a.Outputs["Settings"]),
			ContentLength: aws.Int64(int64(len(b.Logs))),
			Key:           aws.String(fmt.Sprintf("builds/%s.log", b.Id)),
		})
		if err != nil {
			return err
		}
	}

	_, err = p.dynamodb().PutItem(req)

	return err
}

func (p *AWSProvider) buildArgs(a *structs.App, b *structs.Build, source string) []string {
	return []string{
		"run",
		"-i",
		"--name", fmt.Sprintf("build-%s", b.Id),
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		"-e", "APP",
		"-e", "BUILD",
		"-e", "DOCKER_AUTH",
		"-e", "RACK_HOST",
		"-e", "RACK_PASSWORD",
		"-e", "REGISTRY_EMAIL",
		"-e", "REGISTRY_USERNAME",
		"-e", "REGISTRY_PASSWORD",
		"-e", "REGISTRY_ADDRESS",
		"-e", "MANIFEST_PATH",
		"-e", "REPOSITORY",
		"-e", "NO_CACHE",
		p.DockerImageAPI,
		"build",
		source,
	}
}

func (p *AWSProvider) buildEnv(a *structs.App, b *structs.Build, manifest_path string, cache bool) ([]string, error) {
	// self-hosted registry auth
	email := "user@convox.com"
	username := "convox"
	password := p.Password
	address := p.RegistryHost

	// ECR auth
	if registryId := a.Outputs["RegistryId"]; registryId != "" {
		res, err := p.ecr().GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{
			RegistryIds: []*string{aws.String(registryId)},
		})

		if err != nil {
			return nil, err
		}

		if len(res.AuthorizationData) < 1 {
			return nil, fmt.Errorf("no authorization data")
		}

		endpoint := *res.AuthorizationData[0].ProxyEndpoint

		data, err := base64.StdEncoding.DecodeString(*res.AuthorizationData[0].AuthorizationToken)

		if err != nil {
			return nil, err
		}

		parts := strings.SplitN(string(data), ":", 2)

		password = parts[1]
		address = endpoint[8:]
		username = parts[0]
	}

	// TODO: The controller logged into private registries and app registry
	// Seems like this method should be able to generate docker auth config on its own
	dockercfg, err := ioutil.ReadFile("/root/.docker/config.json")
	if err != nil {
		return nil, err
	}

	// Determin callback host. Local Rack should use a variant of localhost
	host := p.NotificationHost

	if p.Development {
		out, err := exec.Command("docker", "run", "convox/docker-gateway").Output()
		if err != nil {
			return nil, err
		}

		host = strings.TrimSpace(string(out))
	}

	env := []string{
		fmt.Sprintf("APP=%s", a.Name),
		fmt.Sprintf("BUILD=%s", b.Id),
		fmt.Sprintf("MANIFEST_PATH=%s", manifest_path),
		fmt.Sprintf("DOCKER_AUTH=%s", dockercfg),
		fmt.Sprintf("RACK_HOST=%s", host),
		fmt.Sprintf("RACK_PASSWORD=%s", p.Password),
		fmt.Sprintf("REGISTRY_EMAIL=%s", email),
		fmt.Sprintf("REGISTRY_USERNAME=%s", username),
		fmt.Sprintf("REGISTRY_PASSWORD=%s", password),
		fmt.Sprintf("REGISTRY_ADDRESS=%s", address),
		fmt.Sprintf("REPOSITORY=%s", a.Outputs["RegistryRepository"]),
	}

	if !cache {
		env = append(env, "NO_CACHE=true")
	}

	return env, nil
}

// buildFromItem populates a Build struct from a DynamoDB Item
func (p *AWSProvider) buildFromItem(item map[string]*dynamodb.AttributeValue) *structs.Build {
	id := coalesce(item["id"], "")
	started, _ := time.Parse(sortableTime, coalesce(item["created"], ""))
	ended, _ := time.Parse(sortableTime, coalesce(item["ended"], ""))

	return &structs.Build{
		Id:          id,
		App:         coalesce(item["app"], ""),
		Description: coalesce(item["description"], ""),
		Manifest:    coalesce(item["manifest"], ""),
		Release:     coalesce(item["release"], ""),
		Status:      coalesce(item["status"], ""),
		Started:     started,
		Ended:       ended,
	}
}

func (p *AWSProvider) buildRun(a *structs.App, b *structs.Build, args []string, env []string, stdin io.Reader) error {
	cmd := exec.Command("docker", args...)
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stderr = cmd.Stdout // redirect cmd stderr to stdout

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		helpers.Error(nil, err) // send internal error to rollbar
		return err
	}

	// start build command
	err = cmd.Start()
	if err != nil {
		helpers.Error(nil, err) // send internal error to rollbar
		return err
	}

	go p.buildWait(a, b, cmd, stdout)

	return nil
}

func (p *AWSProvider) buildWait(a *structs.App, b *structs.Build, cmd *exec.Cmd, stdout io.ReadCloser) {
	// scan all output
	scanner := bufio.NewScanner(stdout)
	out := ""
	for scanner.Scan() {
		text := scanner.Text()
		out += text + "\n"
	}
	if err := scanner.Err(); err != nil {
		helpers.Error(nil, err) // send internal error to rollbar
	}

	var cmdStatus string
	waitErr := make(chan error)
	timeout := time.After(1 * time.Hour)

	go func() {
		err := cmd.Wait()

		switch err.(type) {
		case *exec.ExitError:
			waitErr <- err
		default:
			waitErr <- nil
		}
	}()

	select {

	case werr := <-waitErr:
		// Wait / return code are errors, consider the build failed
		if werr != nil {
			cmdStatus = "failed"
		}

	case <-timeout:
		cmdStatus = "timeout"
		// Force kill the build container since its taking way to long
		killCmd := exec.Command("docker", "kill", fmt.Sprintf("build-%s", b.Id))
		killCmd.Start()
	}

	// reload build item to get data from BuildUpdate callback
	b, err := p.BuildGet(b.App, b.Id)
	if err != nil {
		helpers.Error(nil, err) // send internal error to rollbar
		return
	}

	if cmdStatus != "" { // Careful not to override the status set by BuildUpdate
		b.Status = cmdStatus
	}

	// save final build logs / status
	b.Logs = out
	err = p.BuildSave(b)
	if err != nil {
		helpers.Error(nil, err) // send internal error to rollbar
		return
	}
}

func createTarball(base string) ([]byte, error) {
	cwd, err := os.Getwd()

	if err != nil {
		return nil, err
	}

	err = os.Chdir(base)

	if err != nil {
		return nil, err
	}

	args := []string{"cz"}

	// If .dockerignore exists, use it to exclude files from the tarball
	if _, err = os.Stat(".dockerignore"); err == nil {
		args = append(args, "--exclude-from", ".dockerignore")
	}

	args = append(args, ".")

	cmd := exec.Command("tar", args...)

	out, err := cmd.StdoutPipe()

	if err != nil {
		return nil, err
	}

	cmd.Start()

	bytes, err := ioutil.ReadAll(out)

	if err != nil {
		return nil, err
	}

	err = cmd.Wait()

	if err != nil {
		return nil, err
	}

	err = os.Chdir(cwd)

	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// deleteImages generates a list of fully qualified URLs for images for every process type
// in the build manifest then deletes them.
// Image URLs that point to ECR, e.g. 826133048.dkr.ecr.us-east-1.amazonaws.com/myapp-zridvyqapp:web.BSUSBFCUCSA,
// are deleted with the ECR BatchDeleteImage API.
// Image URLs that point to the convox-hosted registry, e.g. convox-826133048.us-east-1.elb.amazonaws.com:5000/myapp-web:BSUSBFCUCSA,
// are not yet supported and return an error.
func (p *AWSProvider) deleteImages(a *structs.App, b *structs.Build) error {
	var m manifest.Manifest

	err := yaml.Unmarshal([]byte(b.Manifest), &m)
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
	fmt.Println()
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

func readImportArtifact(source io.Reader) (*structs.Build, [][]byte, error) {
	var build structs.Build
	var images [][]byte

	gzf, err := gzip.NewReader(source)
	if err != nil {
		return nil, nil, err
	}

	tarReader := tar.NewReader(gzf)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}

		switch header.Typeflag {
		case tar.TypeReg:
			raw := []byte{}

			if header.Name == "build.json" {
				jsonBuf := bytes.NewBuffer(raw)
				io.Copy(jsonBuf, tarReader)

				err = json.Unmarshal(jsonBuf.Bytes(), &build)
				if err != nil {
					return nil, nil, err
				}

			} else {
				if strings.HasSuffix(header.Name, ".tar") {

					buf := bytes.NewBuffer(raw)
					io.Copy(buf, tarReader)

					images = append(images, buf.Bytes())
				}
			}
		default:
			continue
		}
	}

	return &build, images, nil
}
