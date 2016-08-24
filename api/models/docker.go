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
	"github.com/convox/rack/api/structs"
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
	cres, err := DescribeContainerInstances()

	if len(cres.ContainerInstances) == 0 {
		return "", fmt.Errorf("no container instances")
	}

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
		MaxResults: aws.Int64(1000),
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
	log := Logger.At("DockerLogin").Start()

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

	log = log.Namespace("host=%q user=%q", ac.ServerAddress, ac.Username)

	args := []string{"login", "-e", ac.Email, "-u", ac.Username, "-p", ac.Password, ac.ServerAddress}

	if _, err := exec.Command("docker", args...).CombinedOutput(); err != nil {
		log.Error(err)
		return "", err
	}

	log.Success()
	return ac.ServerAddress, nil
}

func DockerLogout(ac docker.AuthConfiguration) error {
	log := Logger.At("DockerLogout").Namespace("host=%q user=%q", ac.ServerAddress, ac.Username).Start()

	args := []string{"logout", ac.ServerAddress}

	if _, err := exec.Command("docker", args...).CombinedOutput(); err != nil {
		log.Error(err)
		return err
	}

	log.Success()
	return nil
}

// Log into the appropriate registry for the given app
// This could be the self-hosted v1 registry or an ECR registry
func AppDockerLogin(app structs.App) (string, error) {
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
	log := Logger.At("PullAppImages").Start()

	if os.Getenv("DEVELOPMENT") == "true" {
		return
	}

	maxRetries := 5

	apps, err := ListApps()

	if err != nil {
		log.Step("ListApps").Error(err)
		return
	}

	for _, app := range apps {
		a, err := Provider().AppGet(app.Name)
		if err != nil {
			log.Step("GetApp").Error(err)
			continue
		}

		// retry login a few times in case v1 registry is not yet available
		for i := 0; i < maxRetries; i++ {
			_, err = AppDockerLogin(*a)

			if err == nil {
				break
			}

			log.Step("AppDockerLogin").Error(err)
			time.Sleep(30 * time.Second)
		}

		resources, err := ListResources(a.Name)
		if err != nil {
			log.Step("Resources").Error(err)
		}

		for key, r := range resources {
			if strings.HasSuffix(key, "TaskDefinition") {
				td, err := ECS().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
					TaskDefinition: aws.String(r.Id),
				})

				if err != nil {
					log.Step("DescribeTaskDefinition").Error(err)
					continue
				}

				for _, cd := range td.TaskDefinition.ContainerDefinitions {
					log = log.Namespace("image=%q", *cd.Image).Step("Pull")
					_, err := exec.Command("docker", "pull", *cd.Image).CombinedOutput()

					if err != nil {
						log.Error(err)
						fmt.Printf("ns=kernel cn=docker fn=PullAppImages at=exec.Command cmd=%q err=%q\n", fmt.Sprintf("docker pull %s", *cd.Image), err.Error())
						continue
					}

					log.Success()
				}
			}
		}
	}
}

func GetPrivateRegistriesAuth() (Environment, docker.AuthConfigurations119, error) {
	log := Logger.At("GetPrivateRegistriesAuth").Start()

	acs := docker.AuthConfigurations119{}

	env, err := GetRackSettings()

	if err != nil {
		return env, acs, err
	}

	data := []byte(env["DOCKER_AUTH_DATA"])

	if len(data) > 0 {
		if err := json.Unmarshal(data, &acs); err != nil {
			log.Step("json.Unmarshal").Error(err)
			return nil, nil, err
		}
	}

	log.Success()
	return env, acs, nil
}

func LoginPrivateRegistries() error {
	log := Logger.At("LoginPrivateRegistries").Start()

	_, acs, err := GetPrivateRegistriesAuth()

	if err != nil {
		log.Step("GetPrivateRegistriesAuth").Error(err)
		return err
	}

	for _, ac := range acs {
		DockerLogin(ac)
	}

	log.Success()
	return nil
}
