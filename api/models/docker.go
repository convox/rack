package models

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
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
