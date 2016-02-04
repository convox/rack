package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/convox/rack/cmd/agent/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/cmd/agent/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/convox/rack/cmd/agent/Godeps/_workspace/src/github.com/stvp/rollbar"

	"github.com/convox/rack/cmd/agent/Godeps/_workspace/src/github.com/docker/docker/daemon/logger"
	docker "github.com/convox/rack/cmd/agent/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
)

type Monitor struct {
	client *docker.Client

	envs map[string]map[string]string

	agentId    string
	agentImage string

	amiId        string
	az           string
	instanceId   string
	instanceType string
	region       string

	dockerDriver        string
	dockerServerVersion string
	ecsAgentImage       string
	kernelVersion       string
	convoxVersion       string

	lock    sync.Mutex
	lines   map[string][][]byte
	loggers map[string]logger.Logger
}

func NewMonitor() *Monitor {
	fmt.Printf("monitor new region=%s kinesis=%s log_group=%s\n", os.Getenv("AWS_REGION"), os.Getenv("KINESIS"), os.Getenv("LOG_GROUP"))

	client, err := docker.NewClient(os.Getenv("DOCKER_HOST"))

	if err != nil {
		log.Fatal(err)
	}

	info, err := client.Info()

	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
	}

	img, err := GetECSAgentImage(client)

	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
	}

	m := &Monitor{
		client: client,

		envs: make(map[string]map[string]string),

		agentId:    "unknown",          // updated during handleRunning
		agentImage: "convox/agent:dev", // updated during handleRunning

		amiId:        "ami-dev",
		az:           "us-dev-1b",
		instanceId:   "i-dev",
		instanceType: "d1.dev",
		region:       "us-dev-1",

		dockerDriver:        info.Get("Driver"),
		dockerServerVersion: info.Get("ServerVersion"),
		ecsAgentImage:       img,
		kernelVersion:       info.Get("KernelVersion"),

		lines:   make(map[string][][]byte),
		loggers: make(map[string]logger.Logger),
	}

	cfg := ec2metadata.Config{}

	if os.Getenv("EC2_METADATA_ENDPOINT") != "" {
		cfg.Endpoint = aws.String(os.Getenv("EC2_METADATA_ENDPOINT"))
	}

	svc := ec2metadata.New(&cfg)

	if svc.Available() {
		m.amiId, _ = svc.GetMetadata("ami-id")
		m.az, _ = svc.GetMetadata("placement/availability-zone")
		m.instanceId, _ = svc.GetMetadata("instance-id")
		m.instanceType, _ = svc.GetMetadata("instance-type")
		m.region, _ = svc.Region()
	}

	message := fmt.Sprintf("az=%s instanceId=%s instanceType=%s region=%s agentImage=%s amiId=%s dockerServerVersion=%s ecsAgentImage=%s kernelVersion=%s",
		m.az, m.instanceId, m.instanceType, m.region,
		m.agentImage, m.amiId, m.dockerServerVersion, m.ecsAgentImage, m.kernelVersion,
	)

	m.logSystemMetric("monitor at=new", message, true)

	return m
}

// Write event to app CloudWatch Log Group and Kinesis stream
func (m *Monitor) logAppEvent(id, message string) {
	msg := []byte(fmt.Sprintf("%s %s %s : %s", time.Now().Format("2006-01-02 15:04:05"), m.instanceId, m.agentImage, message))

	if awslogger, ok := m.loggers[id]; ok {
		awslogger.Log(&logger.Message{
			ContainerID: id,
			Line:        msg,
			Timestamp:   time.Now(),
		})
	}

	if stream, ok := m.envs[id]["KINESIS"]; ok {
		m.addLine(stream, msg)
	}
}

// Write event to convox CloudWatch Log Group
func (m *Monitor) logSystemMetric(prefix, message string, kinesis bool) {
	message = fmt.Sprintf("%s instanceId=%s %s", prefix, m.instanceId, message)

	fmt.Println(message)

	id := m.agentId
	msg := []byte(fmt.Sprintf("%s %s %s : %s", time.Now().Format("2006-01-02 15:04:05"), m.instanceId, m.agentImage, message))

	if awslogger, ok := m.loggers[id]; ok {
		awslogger.Log(&logger.Message{
			ContainerID: id,
			Line:        msg,
			Timestamp:   time.Now(),
		})
	}

	if stream, ok := m.envs[id]["KINESIS"]; kinesis && ok {
		m.addLine(stream, msg)
	}
}

func GetECSAgentImage(client *docker.Client) (string, error) {
	containers, err := client.ListContainers(docker.ListContainersOptions{})

	if err != nil {
		return "error", err
	}

	for _, c := range containers {
		if strings.HasPrefix(c.Image, "amazon/amazon-ecs-agent") {
			ic, err := client.InspectContainer(c.ID)

			if err != nil {
				return "unknown", err
			}

			return ic.Image[0:12], nil
		}
	}

	return "notfound", nil
}

func (m *Monitor) ReportError(err error) {
	m.logSystemMetric("monitor at=error", fmt.Sprintf("err=%q", err), true)

	rollbar.Token = "f67f25b8a9024d5690f997bd86bf14b0"

	extraData := map[string]string{
		"agentId":    m.agentId,
		"agentImage": m.agentImage,

		"amiId":        m.amiId,
		"az":           m.az,
		"instanceId":   m.instanceId,
		"instanceType": m.instanceType,
		"region":       m.region,

		"dockerDriver":        m.dockerDriver,
		"dockerServerVersion": m.dockerServerVersion,
		"ecsAgentImage":       m.ecsAgentImage,
		"kernelVersion":       m.kernelVersion,
	}
	extraField := &rollbar.Field{"env", extraData}

	rollbar.Error(rollbar.CRIT, err, extraField)
}
