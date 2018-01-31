package aws

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sqs"
)

var (
	appLogStreams = map[string]appLogStream{}
)

type appLogStream struct {
	LogGroup      string
	LogStream     string
	SequenceToken string
}

type queueHandler func(body string) error

type ecsEvent struct {
	Account    string
	DetailType string `json:"detail-type"`
	Detail     interface{}
	ID         string
	Region     string
	Resources  []string
	Source     string
	Time       time.Time
	Version    string
}

type detailTaskStateChange struct {
	ClusterArn string
	Containers []struct {
		ContainerArn string
		LastStatus   string
		Name         string
		TaskArn      string
	}
	CreatedAt     time.Time
	DesiredStatus string
	Group         string
	LastStatus    string
	StartedAt     time.Time
	StartedBy     string
	StoppedReason string
	TaskArn       string
	UpdatedAt     time.Time
}

// StartEventQueue starts the event queue workers
func (p *AWSProvider) workerEvents() {
	go p.handleAccountEvents()
	go p.handleCloudformationEvents()
}

func (p *AWSProvider) handleCloudformationEvents() {
	err := p.processQueue("CloudformationEvents", func(body string) error {
		var raw struct {
			Message string
			Subject string
		}

		if err := json.Unmarshal([]byte(body), &raw); err != nil {
			return err
		}

		message := map[string]string{}

		for _, line := range strings.Split(raw.Message, "\n") {
			parts := strings.SplitN(line, "='", 2)

			if len(parts) == 2 {
				message[strings.TrimSpace(parts[0])] = strings.TrimSuffix(parts[1], "'")
			}
		}

		app := strings.TrimPrefix(message["StackName"], fmt.Sprintf("%s-", os.Getenv("RACK")))

		stream, err := p.getAppLogStream(app)
		if err != nil {
			return err
		}

		req := &cloudwatchlogs.PutLogEventsInput{
			LogGroupName:  aws.String(stream.LogGroup),
			LogStreamName: aws.String(stream.LogStream),
		}

		if stream.SequenceToken != "" {
			req.SequenceToken = aws.String(stream.SequenceToken)
		}

		log := fmt.Sprintf("aws/cloudformation %s %s %s", message["ResourceStatus"], message["LogicalResourceId"], message["ResourceStatusReason"])

		req.LogEvents = []*cloudwatchlogs.InputLogEvent{
			&cloudwatchlogs.InputLogEvent{
				Message:   aws.String(log),
				Timestamp: aws.Int64(time.Now().UnixNano() / 1000000),
			},
		}

		pres, err := p.cloudwatchlogs().PutLogEvents(req)
		if err != nil {
			return err
		}

		stream.SequenceToken = *pres.NextSequenceToken
		appLogStreams[app] = stream

		return nil
	})
	if err != nil {
		panic(err)
	}
}

func (p *AWSProvider) handleAccountEvents() {
	err := p.processQueue("AccountEvents", func(body string) error {
		var e ecsEvent

		if err := json.Unmarshal([]byte(body), &e); err != nil {
			return err
		}

		switch e.DetailType {
		case "ECS Task State Change":
			var detail detailTaskStateChange

			if err := remarshal(e.Detail, &detail); err != nil {
				return err
			}

			if !strings.HasPrefix(detail.Group, "service:") {
				return nil
			}

			parts := strings.Split(detail.ClusterArn, "/")
			cluster := parts[len(parts)-1]
			service := strings.TrimPrefix(detail.Group, "service:")

			stack := os.Getenv("RACK")
			if strings.Contains(service, "-Service") { // an app service
				stack = strings.Split(strings.TrimPrefix(service, fmt.Sprintf("%s-", os.Getenv("RACK"))), "-Service")[0]
			}

			if detail.LastStatus == "PENDING" {
				res, err := p.ecs().DescribeServices(&ecs.DescribeServicesInput{
					Cluster:  aws.String(cluster),
					Services: []*string{aws.String(service)},
				})
				if err != nil {
					return err
				}

				if len(res.Services) < 1 {
					return fmt.Errorf("could not find service: %s", service)
				}

				stream, err := p.getAppLogStream(stack)
				if err != nil {
					return err
				}

				events := res.Services[0].Events

				if len(events) > 5 {
					events = events[0:5]
				}

				req := &cloudwatchlogs.PutLogEventsInput{
					LogGroupName:  aws.String(stream.LogGroup),
					LogStreamName: aws.String(stream.LogStream),
				}

				if stream.SequenceToken != "" {
					req.SequenceToken = aws.String(stream.SequenceToken)
				}

				for _, e := range events {
					req.LogEvents = append(req.LogEvents, &cloudwatchlogs.InputLogEvent{
						Message:   aws.String(fmt.Sprintf("aws/ecs %s", *e.Message)),
						Timestamp: aws.Int64(e.CreatedAt.UnixNano() / 1000000),
					})
				}

				// havent done this in a while
				// TODO use sort.Slice once we upgrade to 1.8
				for i := 0; i < len(req.LogEvents)-1; i++ {
					for j := i + 1; j < len(req.LogEvents); j++ {
						if *req.LogEvents[i].Timestamp > *req.LogEvents[j].Timestamp {
							req.LogEvents[i], req.LogEvents[j] = req.LogEvents[j], req.LogEvents[i]
						}
					}
				}

				pres, err := p.cloudwatchlogs().PutLogEvents(req)
				if err != nil {
					return err
				}

				stream.SequenceToken = *pres.NextSequenceToken
				appLogStreams[stack] = stream
			}
		}

		return nil
	})
	if err != nil {
		panic(err)
	}
}

func (p *AWSProvider) getAppLogStream(app string) (appLogStream, error) {
	group, err := p.rackResource("LogGroup")
	if err != nil {
		return appLogStream{}, err
	}

	stream := fmt.Sprintf("system/%d", time.Now().UnixNano())

	if _, ok := appLogStreams[app]; !ok {
		_, err := p.cloudwatchlogs().CreateLogStream(&cloudwatchlogs.CreateLogStreamInput{
			LogGroupName:  aws.String(group),
			LogStreamName: aws.String(stream),
		})
		if err != nil {
			return appLogStream{}, err
		}

		appLogStreams[app] = appLogStream{
			LogGroup:  group,
			LogStream: stream,
		}
	}

	return appLogStreams[app], nil
}

func (p *AWSProvider) processQueue(resource string, fn queueHandler) error {
	res, err := p.cloudformation().DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName:         aws.String(os.Getenv("RACK")),
		LogicalResourceId: aws.String(resource),
	})
	if err != nil {
		return err
	}
	if len(res.StackResources) < 1 {
		return fmt.Errorf("invalid stack resource: %s", resource)
	}

	queue := *res.StackResources[0].PhysicalResourceId

	for {
		res, err := p.sqs().ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:              aws.String(queue),
			AttributeNames:        []*string{aws.String("All")},
			MessageAttributeNames: []*string{aws.String("All")},
			MaxNumberOfMessages:   aws.Int64(10),
			VisibilityTimeout:     aws.Int64(20),
			WaitTimeSeconds:       aws.Int64(10),
		})
		if err != nil {
			return err
		}

		for _, m := range res.Messages {
			if err := fn(*m.Body); err != nil {
				fmt.Fprintf(os.Stderr, "processQueue %s handler error: %s\n", resource, err)
			}

			_, err := p.sqs().DeleteMessage(&sqs.DeleteMessageInput{
				QueueUrl:      aws.String(queue),
				ReceiptHandle: m.ReceiptHandle,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "processQueue DeleteMessage error: %s\n", err)
			}
		}
	}
	return nil
}

func remarshal(v interface{}, w interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &w)
}
