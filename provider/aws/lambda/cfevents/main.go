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
	uuid "github.com/satori/go.uuid"
)

var (
	CF  = cloudformation.New(session.New(), nil)
	CWL = cloudwatchlogs.New(session.New(), nil)
)

type Event struct {
	Records []Record
}

type Message map[string]string

type Record struct {
	Sns struct {
		Message string
	}
}

type Metadata struct {
	Cluster string
	Rack    string
}

func main() {
	if len(os.Args) < 2 {
		die(fmt.Errorf("must specify event as argument"))
	}

	data := []byte(os.Args[1])

	var e Event

	if err := json.Unmarshal(data, &e); err != nil {
		die(err)
	}

	for _, r := range e.Records {
		if err := handle(r); err != nil {
			die(err)
		}
	}
}

func handle(r Record) error {
	m, err := parseMessage(r.Sns.Message)
	if err != nil {
		return err
	}
	fmt.Printf("m = %+v\n", m)

	resp, err := CF.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(m["StackName"]),
	})
	if err != nil {
		return err
	}

	if len(resp.Stacks) == 0 {
		return fmt.Errorf("stack not found")
	}

	stack := resp.Stacks[0]

	var logGroup string

	logStream := uuid.NewV4().String()
	for _, output := range stack.Outputs {
		if *output.OutputKey == "LogGroup" {
			logGroup = *output.OutputValue
			break
		}
	}

	if logGroup == "" {
		return fmt.Errorf("log group for %s not found", *stack.StackName)
	}
	fmt.Printf("logGroup = %+v\n", logGroup)

	_, err = CWL.CreateLogStream(&cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(logGroup),
		LogStreamName: aws.String(logStream),
	})
	if err != nil {
		return err
	}

	ts := time.Now().UTC()
	if t, ok := m["Timestamp"]; ok {
		tt, err := time.Parse(time.RFC3339, t)
		if err != nil {
			fmt.Printf("could not parse timestamp %s : %s\n", t, err)
		} else {
			ts = tt.UTC()
		}
	}

	fmt.Println("timestamp: ", m["Timestamp"])

	params := &cloudwatchlogs.PutLogEventsInput{
		LogEvents: []*cloudwatchlogs.InputLogEvent{
			{
				Message: aws.String(fmt.Sprintf(
					"CloudformationEvent resource=%s status=%s reason=\"%s\"",
					m["LogicalResourceId"],
					m["ResourceStatus"],
					m["ResourceStatusReason"],
				)),
				Timestamp: aws.Int64(ts.UnixNano() / 1000000),
			},
		},
		LogGroupName:  aws.String(logGroup),
		LogStreamName: aws.String(logStream),
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

func die(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	os.Exit(1)
}

func parseMessage(msg string) (Message, error) {
	m := Message{}

	lines := strings.Split(msg, "\n")

	for _, l := range lines {
		data := strings.SplitN(l, "=", 2)
		if len(data) == 2 {
			value := strings.Trim(data[1], "'")
			m[data[0]] = value
		}
	}

	return m, nil
}
