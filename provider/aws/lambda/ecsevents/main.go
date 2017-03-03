package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ecs"
)

var msg = `{
	"account": "990037048036",
	"detail": {
		"clusterArn": "arn:aws:ecs:us-east-1:990037048036:cluster/devrack-Cluster-1K78FI9LDOTEO",
		"containerInstanceArn": "arn:aws:ecs:us-east-1:990037048036:container-instance/ff6b3f46-fa63-41e7-ab62-ba92df8cccdd",
		"containers": [{
			"containerArn": "arn:aws:ecs:us-east-1:990037048036:container/001c4cef-c701-4d17-90bc-0e5c5bc17621",
			"lastStatus": "RUNNING",
			"name": "web",
			"networkBindings": [{
				"bindIP": "0.0.0.0",
				"containerPort": 8080,
				"hostPort": 37132,
				"protocol": "tcp"
			}, {
				"bindIP": "0.0.0.0",
				"containerPort": 8080,
				"hostPort": 51496,
				"protocol": "tcp"
			}],
			"taskArn": "arn:aws:ecs:us-east-1:990037048036:task/b2292b44-5a13-435a-b2fa-458f8a2363f1"
		}],
		"createdAt": "2017-03-02T16:35:48.514Z",
		"desiredStatus": "RUNNING",
		"group": "service:devrack-node-workers-ServiceWeb-J67T2KMVUAF8",
		"lastStatus": "PENDING",
		"overrides": {
			"containerOverrides": [{
				"name": "web"
			}]
		},
		"startedBy": "ecs-svc/9223370548382827191",
		"taskArn": "arn:aws:ecs:us-east-1:990037048036:task/b2292b44-5a13-435a-b2fa-458f8a2363f1",
		"taskDefinitionArn": "arn:aws:ecs:us-east-1:990037048036:task-definition/devrack-node-workers-web:42",
		"updatedAt": "2017-03-02T16:35:49.678Z",
		"version": 2
	},
	"detail-type": "ECS Task State Change",
	"id": "ee00af90-ce43-413e-8cb6-c02d88687fd3",
	"region": "us-east-1",
	"resources": [
		"arn:aws:ecs:us-east-1:990037048036:task/b2292b44-5a13-435a-b2fa-458f8a2363f1"
	],
	"source": "aws.ecs",
	"time": "2017-03-02T16:35:49Z",
	"version": "0"
}`

var (
	// ECS gives access to the ecs API
	ECS = ecs.New(session.New(), nil)
	// CWL gives access to the cloudwatch logs API
	CWL = cloudwatchlogs.New(session.New(), nil)
	// CF gives access to the cloudformation API
	CF = cloudformation.New(session.New(), nil)
)

// Event represents a cloudwatch event
type Event struct {
	Detail     Detail `json:"detail"`
	DetailType string `json:"detail-type"`
	ID         string `json:"id"`
	Source     string `json:"source"`
}

// Detail represents the details of a given event
type Detail struct {
	ClusterArn string      `json:"clusterArn"`
	Containers []Container `json:"containers"`
	Group      string      `json:"group"`
	LastStatus string      `json:"lastStatus"`
	TaskArn    string      `json:"taskArn"`
}

// Container represents a container that belong to a task
type Container struct {
	LastStatus string `json:"lastStatus"`
	Name       string `json:"name"`
}

// Message is a map of data sent in a Cloudformation event
type Message map[string]string

func main() {
	if len(os.Args) < 2 {
		die(fmt.Errorf("must specify event as argument"))
	}

	data := []byte(os.Args[1])
	//data := []byte(msg)

	var e Event

	if err := json.Unmarshal(data, &e); err != nil {
		die(err)
	}

	if e.Source != "aws.ecs" && e.DetailType != "ECS Task State Change" {
		die(fmt.Errorf("Ignoring: %s - %s", e.Source, e.DetailType))
	}

	if err := handle(e); err != nil {
		die(err)
	}
}

func handle(e Event) error {

	split := strings.SplitN(e.Detail.Group, ":", 2)
	if len(split) != 2 {
		return fmt.Errorf("unkown group: %s", e.Detail.Group)
	}
	service := split[1]

	split = strings.SplitN(service, "-Service", 2)
	if len(split) != 2 {
		return fmt.Errorf("unkown service: %s", service)
	}

	logGroup, err := getLogGroup(split[0])
	if err != nil {
		return fmt.Errorf("log group error: %s", err)
	}

	split = strings.SplitN(e.Detail.TaskArn, "/", 2)
	if len(split) != 2 {
		return fmt.Errorf("unkown task: %s", e.Detail.TaskArn)
	}
	task := split[1]

	format := fmt.Sprintf("ECSEvent service=\"%s\" task=\"%s\" status=\"%s\"", service, task, e.Detail.LastStatus)
	messages := []string{}

	for _, c := range e.Detail.Containers {
		msg := fmt.Sprintf("%s container=\"%s\"", format, c.Name)
		messages = append(messages, msg)
	}

	if e.Detail.LastStatus == "PENDING" {
		split := strings.SplitN(e.Detail.ClusterArn, ":cluster/", 2)
		if len(split) != 2 {
			return fmt.Errorf("unkown service: %s", service)
		}

		el, err := getServiceEvents(service, split[1])
		if err != nil {
			fmt.Printf("unable to get service events: %s\n", err)
		} else {

			for i, ee := range el {
				if i > 5 {
					break // only get the last 5 events
				}
				m := strings.Replace(*ee.Message, "(", "", -1)
				m = strings.Replace(m, ")", "", -1)
				m = fmt.Sprintf("ECSEvent event=\"%s\"", m)
				messages = append(messages, m)
			}
		}
	}

	logEvents := []*cloudwatchlogs.InputLogEvent{}
	for _, m := range messages {
		logEvents = append(logEvents, &cloudwatchlogs.InputLogEvent{
			Message:   aws.String(m),
			Timestamp: aws.Int64(time.Now().UnixNano() / 1000000),
		})
	}

	_, err = CWL.CreateLogStream(&cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(logGroup),
		LogStreamName: aws.String(e.ID),
	})
	if err != nil {
		return err
	}

	fmt.Printf("e.ID = %+v\n", e.ID)
	params := &cloudwatchlogs.PutLogEventsInput{
		LogEvents:     logEvents,
		LogGroupName:  aws.String(logGroup),
		LogStreamName: aws.String(e.ID),
	}
	le, err := CWL.PutLogEvents(params)
	if err != nil {
		return err
	}

	if le.RejectedLogEventsInfo != nil {
		return fmt.Errorf("rejected log event: %s", le.RejectedLogEventsInfo.String())
	}

	fmt.Println("success")
	return nil
}

func getServiceEvents(service, cluster string) ([]*ecs.ServiceEvent, error) {
	params := &ecs.DescribeServicesInput{
		Services: []*string{
			aws.String(service),
		},
		Cluster: aws.String(cluster),
	}
	resp, err := ECS.DescribeServices(params)
	if err != nil {
		return nil, err
	}

	if len(resp.Services) == 0 {
		return nil, fmt.Errorf("service %s no found", service)
	}

	return resp.Services[0].Events, nil

}

func getLogGroup(stack string) (string, error) {
	resp, err := CF.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stack),
	})
	if err != nil {
		return "", err
	}

	if len(resp.Stacks) == 0 {
		return "", fmt.Errorf("stack not found")
	}

	s := resp.Stacks[0]

	var logGroup string
	for _, output := range s.Outputs {
		if *output.OutputKey == "LogGroup" {
			logGroup = *output.OutputValue
			break
		}
	}

	if logGroup == "" {
		return "", fmt.Errorf("log group for %s not found", *s.StackName)
	}
	return logGroup, nil
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	os.Exit(1)
}
