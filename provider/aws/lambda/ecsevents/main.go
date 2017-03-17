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

	format := fmt.Sprintf("[ECS] service=\"%s\" task=\"%s\" status=\"%s\"", service, task, e.Detail.LastStatus)
	messages := []string{}

	for _, c := range e.Detail.Containers {
		msg := fmt.Sprintf("%s container=\"%s\"", format, c.Name)
		messages = append(messages, msg)
	}

	if e.Detail.LastStatus == "PENDING" {
		split := strings.SplitN(e.Detail.ClusterArn, ":cluster/", 2)
		if len(split) != 2 {
			return fmt.Errorf("unkown cluster: %s", service)
		}

		el, err := getServiceEvents(service, split[1])
		if err != nil {
			fmt.Printf("unable to get service events: %s\n", err)
		} else {

			for i, ee := range el {
				if i >= 5 {
					break // only get the last 5 events
				}
				m := strings.Replace(*ee.Message, "(", "", -1)
				m = strings.Replace(m, ")", "", -1)
				m = fmt.Sprintf("[ECS] event=\"%s\"", m)
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

	fmt.Printf("e.ID = %+v\n", e.ID)
	_, err = CWL.CreateLogStream(&cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(logGroup),
		LogStreamName: aws.String(e.ID),
	})
	if err != nil {
		return err
	}

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
