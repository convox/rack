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
	"github.com/convox/rack/pkg/cache"
)

var (
	ecsEvents = map[string]bool{}
	started   = time.Now().UTC()
)

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

		fmt.Println("message")
		keys := []string{}
		for k := range message {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("%s: %s\n", k, message[k])
		}

		stack := message["StackName"]

		group, err := p.getStackLogGroup(stack)
		if err != nil {
			return err
		}

		stream := "system/cloudformation"

		req := &cloudwatchlogs.PutLogEventsInput{
			LogGroupName:  aws.String(group),
			LogStreamName: aws.String(stream),
		}

		if token, ok := cache.Get("logStreamSequenceToken", fmt.Sprintf("%s/%s", group, stream)).(string); ok {
			req.SequenceToken = aws.String(token)
		}

		log := fmt.Sprintf("aws/cfm %s %s %s %s", stack, message["ResourceStatus"], message["LogicalResourceId"], message["ResourceStatusReason"])

		req.LogEvents = []*cloudwatchlogs.InputLogEvent{
			&cloudwatchlogs.InputLogEvent{
				Message:   aws.String(log),
				Timestamp: aws.Int64(time.Now().UnixNano() / int64(time.Millisecond)),
			},
		}

		token, err := p.putLogEvents(req)
		if err != nil {
			return err
		}

		cache.Set("logStreamSequenceToken", fmt.Sprintf("%s/%s", group, stream), token, 4*time.Hour)

		return nil
	})
	if err != nil {
		panic(err)
	}
}

func (p *Provider) handleECSEvents() {
	for {
		time.Sleep(time.Duration(p.EcsPollInterval) * time.Second)

		if err := p.pollECSEvents(); err != nil {
			fmt.Printf("err = %+v\n", err)
		}
	}
}

func (p *Provider) pollECSEvents() error {
	prefix := fmt.Sprintf("%s-", p.Rack)

	lreq := &ecs.ListServicesInput{
		Cluster: aws.String(p.Cluster),
	}

	for {
		lres, err := p.ecs().ListServices(lreq)
		if err != nil {
			break
		}

		sres, err := p.ecs().DescribeServices(&ecs.DescribeServicesInput{
			Cluster:  aws.String(p.Cluster),
			Services: lres.ServiceArns,
		})
		if err != nil {
			return err
		}

		for _, s := range sres.Services {
			name := *s.ServiceName

			if !strings.HasPrefix(name, prefix) || !strings.Contains(name, "-Service") {
				continue
			}

			app := strings.Split(strings.TrimPrefix(name, prefix), "-Service")[0]

			stack := p.rackStack(app)

			group, err := p.getStackLogGroup(stack)
			if err != nil {
				return err
			}

			stream := "system/ecs"

			req := &cloudwatchlogs.PutLogEventsInput{
				LogGroupName:  aws.String(group),
				LogStreamName: aws.String(stream),
			}

			if token, ok := cache.Get("logStreamSequenceToken", fmt.Sprintf("%s/%s", group, stream)).(string); ok {
				req.SequenceToken = aws.String(token)
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

			token, err := p.putLogEvents(req)
			if err != nil {
				return err
			}

			cache.Set("logStreamSequenceToken", fmt.Sprintf("%s/%s", group, stream), token, 4*time.Hour)
		}

		if lres.NextToken == nil {
			break
		}

		lreq.NextToken = lres.NextToken
	}

	return nil
}

func (p *Provider) getStackLogGroup(stack string) (string, error) {
	if group, ok := cache.Get("stackLogGroup", stack).(string); ok {
		return group, nil
	}

	s, err := p.describeStack(stack)
	if err != nil {
		return "", err
	}

	if s.ParentId != nil {
		return p.getStackLogGroup(*s.ParentId)
	}

	r, err := p.stackResource(stack, "LogGroup")
	if err != nil {
		if strings.Contains(err.Error(), "resource not found") {
			return p.getStackLogGroup(p.Rack)
		}
		return "", err
	}

	g := *r.PhysicalResourceId

	cache.Set("stackLogGroup", stack, g, 5*time.Minute)

	return g, nil
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
