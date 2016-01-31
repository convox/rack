package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/convox/agent/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/ec2metadata"

	"github.com/convox/agent/Godeps/_workspace/src/github.com/docker/docker/daemon/logger"
	docker "github.com/convox/agent/Godeps/_workspace/src/github.com/fsouza/go-dockerclient"
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

	dockerVersion   string
	ecsAgentVersion string
	convoxVersion   string

	lock    sync.Mutex
	lines   map[string][][]byte
	loggers map[string]logger.Logger
}

func NewMonitor() *Monitor {
	client, err := docker.NewClient(os.Getenv("DOCKER_HOST"))

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("monitor new region=%s kinesis=%s log_group=%s\n", os.Getenv("AWS_REGION"), os.Getenv("KINESIS"), os.Getenv("LOG_GROUP"))

	m := &Monitor{
		client: client,

		envs: make(map[string]map[string]string),

		agentImage: "convox/agent", // also set during handleRunning

		amiId:        "ami-dev",
		az:           "us-dev-1b",
		instanceId:   "i-dev",
		instanceType: "d1.dev",
		region:       "us-dev-1",

		lines:   make(map[string][][]byte),
		loggers: make(map[string]logger.Logger),
	}

	svc := ec2metadata.New(&ec2metadata.Config{})

	if MetadataAvailable() {
		m.amiId, _ = svc.GetMetadata("ami-id")
		m.az, _ = svc.GetMetadata("placement/availability-zone")
		m.instanceId, _ = svc.GetMetadata("instance-id")
		m.instanceType, _ = svc.GetMetadata("instance-type")
		m.region, _ = svc.Region()
	}

	return m
}

func MetadataAvailable() bool {
	client := http.Client{
		Timeout: 500 * time.Millisecond,
	}

	_, err := client.Get("http://169.254.169.254/latest/meta-data/instance-id")

	if err != nil {
		fmt.Printf("error: %s\n", err)
		return false
	}

	return true
}

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

func (m *Monitor) logSystemEvent(prefix, message string) {
	msg := fmt.Sprintf("%s dim#amiId=%s dim#az=%s dim#instanceId=%s dim#instanceType=%s dim#region=%s %s",
		prefix,
		m.amiId, m.az, m.instanceId, m.instanceType, m.region,
		message,
	)

	fmt.Println(msg)

	m.logAppEvent(m.agentId, msg)
}
