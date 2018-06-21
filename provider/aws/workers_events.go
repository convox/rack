package aws

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sqs"
)

var (
	appLogGroups  = map[string]string{}
	appLogStreams = map[string]appLogStream{}
	ecsEvents     = map[string]bool{}
	started       = time.Now().UTC()
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
	go p.handleECSEvents()
}

func (p *AWSProvider) handleAccountEvents() {
	err := p.processQueue("AccountEvents", func(body string) error {
		var e ecsEvent

		if err := json.Unmarshal([]byte(body), &e); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		panic(err)
	}
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

		// ignore rack events for now
		if message["StackName"] == os.Getenv("RACK") {
			return nil
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

		log := fmt.Sprintf("aws/cfm %s %s %s", message["ResourceStatus"], message["LogicalResourceId"], message["ResourceStatusReason"])

		req.LogEvents = []*cloudwatchlogs.InputLogEvent{
			&cloudwatchlogs.InputLogEvent{
				Message:   aws.String(log),
				Timestamp: aws.Int64(time.Now().UnixNano() / int64(time.Millisecond)),
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

func (p *AWSProvider) handleECSEvents() {
	log := Logger.At("handleECSEvents")

	prefix := fmt.Sprintf("%s-", p.Rack)
	logInterval, err := strconv.Atoi(os.Getenv("LOG_SERVICE_EVENTS_INTERVAL_SECONDS"))
	if err != nil {
		log.Logf("Invalid int value for environment variable LOG_SERVICE_EVENTS_INTERVAL_SECONDS, defaulting to 1")
		logInterval = 1
	}

	for {
		time.Sleep(logInterval * time.Second)

		lreq := &ecs.ListServicesInput{
			Cluster: aws.String(p.Cluster),
		}

		for {
			lres, err := p.ecs().ListServices(lreq)
			if err != nil {
				log.Error(err)
				break
			}

			sres, err := p.ecs().DescribeServices(&ecs.DescribeServicesInput{
				Cluster:  aws.String(p.Cluster),
				Services: lres.ServiceArns,
			})
			if err != nil {
				log.Error(err)
				break
			}

			for _, s := range sres.Services {
				name := *s.ServiceName

				if !strings.HasPrefix(name, prefix) || !strings.Contains(name, "-Service") {
					continue
				}

				app := strings.Split(strings.TrimPrefix(name, prefix), "-Service")[0]

				stream, err := p.getAppLogStream(app)
				if err != nil {
					log.Error(err)
					continue
				}

				req := &cloudwatchlogs.PutLogEventsInput{
					LogGroupName:  aws.String(stream.LogGroup),
					LogStreamName: aws.String(stream.LogStream),
				}

				if stream.SequenceToken != "" {
					req.SequenceToken = aws.String(stream.SequenceToken)
				}

				for _, e := range s.Events {
					if _, ok := ecsEvents[*e.Id]; !ok {
						if e.CreatedAt.After(started) {
							req.LogEvents = append(req.LogEvents, &cloudwatchlogs.InputLogEvent{
								Message:   aws.String(fmt.Sprintf("aws/ecs %s", *e.Message)),
								Timestamp: aws.Int64(time.Now().UTC().UnixNano() / int64(time.Millisecond)),
							})
						}
						ecsEvents[*e.Id] = true
					}
				}

				if len(req.LogEvents) == 0 {
					continue
				}

				sort.Slice(req.LogEvents, func(i, j int) bool {
					return *req.LogEvents[i].Timestamp < *req.LogEvents[j].Timestamp
				})

				pres, err := p.cloudwatchlogs().PutLogEvents(req)
				if err != nil {
					log.Error(err)
					continue
				}

				stream.SequenceToken = *pres.NextSequenceToken
				appLogStreams[app] = stream
			}

			// fmt.Printf("sres = %+v\n", sres)

			if lres.NextToken == nil {
				break
			}

			lreq.NextToken = lres.NextToken
		}
	}
}

func (p *AWSProvider) getAppLogStream(app string) (appLogStream, error) {
	group, ok := appLogGroups[app]
	if !ok {
		g, err := p.appResource(app, "LogGroup")
		if err != nil {
			return appLogStream{}, err
		}
		group = g
		appLogGroups[app] = group
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
