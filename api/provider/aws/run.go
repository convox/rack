package aws

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecr"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
	"github.com/convox/rack/api/structs"
)

func (p *AWSProvider) RunAttached(app, process, command string, rw io.ReadWriter) error {
	resources, err := p.stackResources(app)

	if err != nil {
		return err
	}

	task, err := p.ecs().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(resources[upperName(process)+"ECSTaskDefinition"].Id),
	})

	if err != nil {
		return err
	}

	var container *ecs.ContainerDefinition

	for _, container = range task.TaskDefinition.ContainerDefinitions {
		if *container.Name == process {
			break
		}
	}

	ea := make([]string, 0)

	for _, env := range container.Environment {
		ea = append(ea, fmt.Sprintf("%s=%s", *env.Name, *env.Value))
	}

	release, err := p.appLatestRelease(app)

	if err != nil {
		return err
	}

	if release == nil {
		return fmt.Errorf("no releases for app: %s", app)
	}

	manifest, err := structs.LoadManifest(release.Manifest)

	if err != nil {
		return err
	}

	me := manifest.Entry(process)

	if me == nil {
		return fmt.Errorf("no such process: %s", process)
	}

	binds := []string{}
	host := ""

	pss, err := p.ProcessList(app)

	if err != nil {
		return err
	}

	for _, ps := range pss {
		if ps.Name == process {
			binds = ps.Binds
			host = fmt.Sprintf("http://%s:2376", ps.Host)
			break
		}
	}

	a, err := p.AppGet(app)

	if err != nil {
		return err
	}

	var image, repository, tag, username, password, serverAddress string

	if registryId := a.Outputs["RegistryId"]; registryId != "" {
		image = fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s.%s", registryId, os.Getenv("AWS_REGION"), a.Outputs["RegistryRepository"], me.Name, release.Build)
		repository = fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s", registryId, os.Getenv("AWS_REGION"), a.Outputs["RegistryRepository"])
		tag = fmt.Sprintf("%s.%s", me.Name, release.Build)

		resp, err := p.ecr().GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{
			RegistryIds: []*string{aws.String(a.Outputs["RegistryId"])},
		})

		if err != nil {
			return err
		}

		if len(resp.AuthorizationData) < 1 {
			return fmt.Errorf("no authorization data")
		}

		endpoint := *resp.AuthorizationData[0].ProxyEndpoint
		serverAddress = endpoint[8:]

		data, err := base64.StdEncoding.DecodeString(*resp.AuthorizationData[0].AuthorizationToken)

		if err != nil {
			return err
		}

		parts := strings.SplitN(string(data), ":", 2)

		username = parts[0]
		password = parts[1]
	} else {
		image = fmt.Sprintf("%s/%s-%s:%s", os.Getenv("REGISTRY_HOST"), app, me.Name, release.Build)
		repository = fmt.Sprintf("%s/%s-%s", os.Getenv("REGISTRY_HOST"), app, me.Name)
		tag = release.Build
		serverAddress = os.Getenv("REGISTRY_HOST")
		username = "convox"
		password = os.Getenv("PASSWORD")
	}

	d, err := dockerClient(host)

	if err != nil {
		return err
	}

	err = d.PullImage(docker.PullImageOptions{
		Repository: repository,
		Tag:        tag,
	}, docker.AuthConfiguration{
		ServerAddress: serverAddress,
		Username:      username,
		Password:      password,
	})

	if err != nil {
		return err
	}

	res, err := d.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			Env:          ea,
			OpenStdin:    true,
			Tty:          true,
			Cmd:          []string{"sh", "-c", command},
			Image:        image,
			Labels: map[string]string{
				"com.convox.rack.type":    "oneoff",
				"com.convox.rack.app":     app,
				"com.convox.rack.process": process,
			},
		},
		HostConfig: &docker.HostConfig{
			Binds: binds,
		},
	})

	if err != nil {
		return err
	}

	ir, iw := io.Pipe()
	or, ow := io.Pipe()

	go d.AttachToContainer(docker.AttachToContainerOptions{
		Container:    res.ID,
		InputStream:  ir,
		OutputStream: ow,
		ErrorStream:  ow,
		Stream:       true,
		Stdin:        true,
		Stdout:       true,
		Stderr:       true,
		RawTerminal:  true,
	})

	go io.Copy(iw, rw)
	go io.Copy(rw, or)

	// hacky
	time.Sleep(100 * time.Millisecond)

	err = d.StartContainer(res.ID, nil)

	if err != nil {
		return err
	}

	code, err := d.WaitContainer(res.ID)

	if err != nil {
		return err
	}

	_, err = rw.Write([]byte(fmt.Sprintf("%s%d\n", StatusCodePrefix, code)))

	if err != nil {
		return err
	}

	return nil
}

func (p *AWSProvider) RunDetached(app, process, command string) error {
	resources, err := p.stackResources(app)

	if err != nil {
		return err
	}

	req := &ecs.RunTaskInput{
		Cluster:        aws.String(os.Getenv("CLUSTER")),
		Count:          aws.Int64(1),
		TaskDefinition: aws.String(resources[upperName(process)+"ECSTaskDefinition"].Id),
	}

	if command != "" {
		req.Overrides = &ecs.TaskOverride{
			ContainerOverrides: []*ecs.ContainerOverride{
				&ecs.ContainerOverride{
					Name: aws.String(process),
					Command: []*string{
						aws.String("sh"),
						aws.String("-c"),
						aws.String(command),
					},
				},
			},
		}
	}

	_, err = p.ecs().RunTask(req)

	if err != nil {
		return err
	}

	return nil
}
