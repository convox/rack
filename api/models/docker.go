package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/fsouza/go-dockerclient"
)

var regexpECR = regexp.MustCompile(`(\d+)\.dkr\.ecr\.([^.]+)\.amazonaws\.com.*`)

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

func DockerLogin(ac docker.AuthConfiguration) (string, error) {
	if ac.Email == "" {
		ac.Email = "user@convox.com"
	}

	// if ECR URL, try Username and Password as IAM keys to get auth token
	if match := regexpECR.FindStringSubmatch(ac.ServerAddress); len(match) > 1 {
		ECR := ecr.New(session.New(), &aws.Config{
			Credentials: credentials.NewStaticCredentials(ac.Username, ac.Password, ""),
			Region:      aws.String(match[2]),
		})

		res, err := ECR.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{
			RegistryIds: []*string{aws.String(match[1])},
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

		ac.Password = parts[1]
		ac.ServerAddress = endpoint[8:]
		ac.Username = parts[0]
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

	return ac.ServerAddress, err
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
func AppDockerLogin(app App) (string, error) {
	if registryId := app.Outputs["RegistryId"]; registryId != "" {
		return DockerLogin(docker.AuthConfiguration{
			Email:         "user@convox.com",
			Password:      os.Getenv("AWS_SECRET"),
			ServerAddress: fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", registryId, os.Getenv("AWS_REGION")),
			Username:      os.Getenv("AWS_ACCESS"),
		})
	}

	// fall back to v1 registry login
	return DockerLogin(docker.AuthConfiguration{
		Email:         "user@convox.com",
		Password:      os.Getenv("PASSWORD"),
		ServerAddress: os.Getenv("REGISTRY_HOST"),
		Username:      "convox",
	})
}

func PullAppImages() {
	fmt.Printf("ns=kernel cn=docker fn=PullAppImages\n")

	if os.Getenv("DEVELOPMENT") == "true" {
		return
	}

	maxRetries := 5

	apps, err := ListApps()

	if err != nil {
		fmt.Printf("ns=kernel cn=docker fn=PullAppImages at=ListApps err=%q\n", err)
		return
	}

	for _, app := range apps {
		a, err := GetApp(app.Name)

		if err != nil {
			fmt.Printf("ns=kernel cn=docker fn=PullAppImages at=GetApp err=%q\n", err.Error())
			continue
		}

		// retry login a few times in case v1 registry is not yet available
		for i := 0; i < maxRetries; i++ {
			_, err = AppDockerLogin(app)

			if err == nil {
				break
			}

			fmt.Printf("ns=kernel cn=docker fn=PullAppImages at=AppDockerLogin err=%q\n", err.Error())
			time.Sleep(30 * time.Second)
		}

		resources, err := a.Resources()

		if err != nil {
			fmt.Printf("ns=kernel cn=docker fn=PullAppImages at=Resources err=%q\n", err)
		}

		for key, r := range resources {
			if strings.HasSuffix(key, "TaskDefinition") {
				td, err := ECS().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
					TaskDefinition: aws.String(r.Id),
				})

				if err != nil {
					fmt.Printf("ns=kernel cn=docker fn=PullAppImages at=DescribeTaskDefinition err=%q\n", err.Error())
					continue
				}

				for _, cd := range td.TaskDefinition.ContainerDefinitions {
					fmt.Printf("IMAGE: %s", *cd.Image)

					fmt.Printf("ns=kernel cn=docker fn=PullAppImages at=exec.Command cmd=%q\n", fmt.Sprintf("docker pull %s", *cd.Image))

					_, err := exec.Command("docker", "pull", *cd.Image).CombinedOutput()

					if err != nil {
						fmt.Printf("ns=kernel cn=docker fn=PullAppImages at=exec.Command cmd=%q err=%q\n", fmt.Sprintf("docker pull %s", *cd.Image), err.Error())
						continue
					}
				}
			}
		}
	}
}

func GetPrivateRegistriesAuth() (Environment, docker.AuthConfigurations119, error) {
	fmt.Printf("ns=kernel cn=docker fn=GetPrivateRegistriesAuth\n")

	acs := docker.AuthConfigurations119{}

	env, err := GetRackSettings()

	if err != nil {
		return env, acs, err
	}

	data := []byte(env["DOCKER_AUTH_DATA"])

	if len(data) > 0 {
		if err := json.Unmarshal(data, &acs); err != nil {
			return env, acs, err
		}
	}

	return env, acs, nil
}

func LoginPrivateRegistries() error {
	fmt.Printf("ns=kernel cn=docker fn=LoginPrivateRegistries\n")

	_, acs, err := GetPrivateRegistriesAuth()

	if err != nil {
		return err
	}

	for _, ac := range acs {
		DockerLogin(ac)
	}

	return nil
}
