package models

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecr"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
	"github.com/convox/rack/api/structs"
)

func Docker(host string) (*docker.Client, error) {
	if host == "" {
		h, err := DockerHost()

		if err != nil {
			return nil, err
		}

		host = h
	}

	if h := os.Getenv("TEST_DOCKER_HOST"); h != "" {
		host = h
	}

	return docker.NewClient(host)
}

func DockerHost() (string, error) {
	ares, err := ECS().ListContainerInstances(&ecs.ListContainerInstancesInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
	})

	if len(ares.ContainerInstanceArns) == 0 {
		return "", fmt.Errorf("no container instances")
	}

	cres, err := ECS().DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(os.Getenv("CLUSTER")),
		ContainerInstances: ares.ContainerInstanceArns,
	})

	if err != nil {
		return "", err
	}

	if len(cres.ContainerInstances) == 0 {
		return "", fmt.Errorf("no container instances")
	}

	id := *cres.ContainerInstances[rand.Intn(len(cres.ContainerInstances))].Ec2InstanceId

	ires, err := EC2().DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{Name: aws.String("instance-id"), Values: []*string{&id}},
		},
	})

	if len(ires.Reservations) != 1 || len(ires.Reservations[0].Instances) != 1 {
		return "", fmt.Errorf("could not describe container instance")
	}

	ip := *ires.Reservations[0].Instances[0].PrivateIpAddress

	if os.Getenv("DEVELOPMENT") == "true" {
		ip = *ires.Reservations[0].Instances[0].PublicIpAddress
	}

	return fmt.Sprintf("http://%s:2376", ip), nil
}

func DockerLogin(ac docker.AuthConfiguration) error {
	if ac.Email == "" {
		ac.Email = "user@convox.com"
	}

	args := []string{"login", "-e", ac.Email, "-u", ac.Username, "-p", ac.Password, ac.ServerAddress}

	out, err := exec.Command("docker", args...).CombinedOutput()

	// log args with password masked
	args[6] = "*****"
	cmd := fmt.Sprintf("docker %s", strings.Trim(fmt.Sprint(args), "[]"))

	if err != nil {
		fmt.Printf("ns=kernel cn=docker at=DockerLogin state=error step=exec.Command cmd=%q out=%q err=%q\n", cmd, out, err)
	} else {
		fmt.Printf("ns=kernel cn=docker at=DockerLogin state=success step=exec.Command cmd=%q\n", cmd)
	}

	return err
}

func DockerLogout(ac docker.AuthConfiguration) error {
	args := []string{"logout", ac.ServerAddress}

	out, err := exec.Command("docker", args...).CombinedOutput()

	cmd := fmt.Sprintf("docker %s", strings.Trim(fmt.Sprint(args), "[]"))

	if err != nil {
		fmt.Printf("ns=kernel cn=docker at=DockerLogout state=error step=exec.Command cmd=%q out=%q err=%q\n", cmd, out, err)
	} else {
		fmt.Printf("ns=kernel cn=docker at=DockerLogout state=success step=exec.Command cmd=%q\n", cmd)
	}

	return err
}

// Log into the appropriate registry for the given app
// This could be the self-hosted v1 registry or an ECR registry
func AppDockerLogin(app structs.App) (string, error) {
	if registryId := app.Outputs["RegistryId"]; registryId != "" {
		res, err := ECR().GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{
			RegistryIds: []*string{aws.String(app.Outputs["RegistryId"])},
		})

		if err != nil {
			return "", err
		}

		if len(res.AuthorizationData) < 1 {
			return "", fmt.Errorf("no authorization data")
		}

		endpoint := *res.AuthorizationData[0].ProxyEndpoint

		data, err := base64.StdEncoding.DecodeString(*res.AuthorizationData[0].AuthorizationToken)

		if err != nil {
			return "", err
		}

		parts := strings.SplitN(string(data), ":", 2)

		err = DockerLogin(docker.AuthConfiguration{
			Email:         "user@convox.com",
			Password:      parts[1],
			ServerAddress: endpoint,
			Username:      parts[0],
		})

		return endpoint[8:], err
	}

	// fall back to v1 registry login
	err := DockerLogin(docker.AuthConfiguration{
		Email:         "user@convox.com",
		Password:      os.Getenv("PASSWORD"),
		ServerAddress: os.Getenv("REGISTRY_HOST"),
		Username:      "convox",
	})

	return os.Getenv("REGISTRY_HOST"), err
}
