package aws

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/convox/rack/cache"
)

var (
	ecsEvents = map[string]bool{}
	started   = time.Now().UTC()
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
func (p *Provider) workerEvents() {
	go p.handleAccountEvents()
	go p.handleCloudformationEvents()
	go p.handleECSEvents()
}

func (p *Provider) handleAccountEvents() {
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

func (p *Provider) handleCloudformationEvents() {
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
		if message["StackName"] == p.Rack {
			return nil
		}

		app := strings.TrimPrefix(message["StackName"], fmt.Sprintf("%s-", p.Rack))

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

		cache.Set("appLogStreams", app, stream, 5*time.Minute)

		return nil
	})
	if err != nil {
		panic(err)
	}
}

func (p *Provider) handleECSEvents() {
	log := Logger.At("handleECSEvents")

	prefix := fmt.Sprintf("%s-", p.Rack)

	for {
		time.Sleep(time.Duration(p.EcsPollInterval) * time.Second)

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
					// log.Error(err)
					continue
				}

				stream.SequenceToken = *pres.NextSequenceToken

				cache.Set("appLogStreams", app, stream, 5*time.Minute)
			}

			// fmt.Printf("sres = %+v\n", sres)

			if lres.NextToken == nil {
				break
			}

			lreq.NextToken = lres.NextToken
		}
	}
}

func (p *Provider) getAppLogStream(app string) (appLogStream, error) {
	group, ok := cache.Get("appLogGroups", app).(string)
	if !ok {
		g, err := p.appResource(app, "LogGroup")
		if err != nil {
			return appLogStream{}, err
		}
		group = g
		cache.Set("appLogGroups", app, group, 5*time.Minute)
	}

	name := fmt.Sprintf("system/%d", time.Now().UnixNano())

	stream, ok := cache.Get("appLogStreams", app).(appLogStream)
	if !ok {
		_, err := p.cloudwatchlogs().CreateLogStream(&cloudwatchlogs.CreateLogStreamInput{
			LogGroupName:  aws.String(group),
			LogStreamName: aws.String(name),
		})
		if err != nil {
			return appLogStream{}, err
		}

		stream = appLogStream{
			LogGroup:  group,
			LogStream: name,
		}

		cache.Set("appLogStreams", app, stream, 5*time.Minute)
	}

	return stream, nil
}

func (p *Provider) processQueue(resource string, fn queueHandler) error {
	res, err := p.cloudformation().DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName:         aws.String(p.Rack),
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
