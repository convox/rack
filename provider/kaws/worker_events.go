package kaws

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	eventMessageSplitter = regexp.MustCompile(`([^=]+)='([^']*)'\n`)
	stackTagCache        = map[string]map[string]string{}
)

func (p *Provider) workerEvents() error {
	if p.EventQueue == "" {
		return fmt.Errorf("no queue url")
	}

	for {
		err := p.processQueue(p.EventQueue, 5, func(m *sqs.Message) error {
			attrs, err := topicAttributes(*m.Body)
			if err != nil {
				return err
			}

			stack := attrs["StackName"]

			tags, err := p.stackTagsCached(stack)
			if err != nil {
				return err
			}

			switch stack {
			case p.Rack:
				fmt.Printf(fmt.Sprintf("system/cfn %s %s %s\n", attrs["ResourceStatus"], attrs["LogicalResourceId"], attrs["ResourceStatusReason"]))

				switch attrs["ResourceType"] {
				case "AWS::CloudFormation::Stack":
					switch attrs["ResourceStatus"] {
					case "ROLLBACK_COMPLETE", "UPDATE_COMPLETE":
						fmt.Println("stack finished")

						os, err := p.stackOutputs(p.Rack)
						if err != nil {
							return err
						}

						opts := structs.SystemUpdateOptions{
							Version: options.String(os["Version"]),
						}

						if err := p.Provider.SystemUpdate(opts); err != nil {
							return err
						}
					}
				}
			default:
				app := tags["app"]

				p.Log(app, fmt.Sprintf("system/cfn/%s", attrs["StackName"]), time.Now().UTC(), fmt.Sprintf("%s %s %s", attrs["ResourceStatus"], attrs["LogicalResourceId"], attrs["ResourceStatusReason"]))

				switch attrs["ResourceType"] {
				case "AWS::CloudFormation::Stack":
					if err := p.stackStatusUpdate(attrs["StackName"], attrs["ResourceStatus"]); err != nil {
						return err
					}
				}
			}

			return nil
		})
		if err != nil {
			fmt.Printf("err = %+v\n", err)
			time.Sleep(10 * time.Second)
		}
	}
}

func (p *Provider) processQueue(queue string, max int, handler func(m *sqs.Message) error) error {
	for {
		res, err := p.SQS.ReceiveMessage(&sqs.ReceiveMessageInput{
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
			filter, err := filterMessage(m, max)
			if err != nil {
				fmt.Printf("err = %+v\n", err)
				continue
			}

			if !filter {
				if err := handler(m); err != nil {
					fmt.Printf("err = %+v\n", err)
					continue
				}
			}

			_, err = p.SQS.DeleteMessage(&sqs.DeleteMessageInput{
				QueueUrl:      aws.String(p.EventQueue),
				ReceiptHandle: m.ReceiptHandle,
			})
			if err != nil {
				fmt.Printf("err = %+v\n", err)
				continue
			}
		}
	}
}

func (p *Provider) stackStatusUpdate(stack, status string) error {
	c, err := p.convoxClient()
	if err != nil {
		return err
	}

	tags, err := p.stackTagsCached(stack)
	if err != nil {
		return err
	}

	app := tags["app"]
	kind := tags["type"]
	name := tags["name"]

	s, err := c.ConvoxV1().Stacks(p.AppNamespace(app)).Get(tags["stack"], am.GetOptions{})
	if err != nil {
		return err
	}

	switch status {
	case "CREATE_COMPLETE", "UPDATE_COMPLETE", "ROLLBACK_COMPLETE", "UPDATE_ROLLBACK_COMPLETE":
		s.Status = "Running"
	case "CREATE_FAILED", "DELETE_FAILED", "ROLLBACK_FAILED", "UPDATE_ROLLBACK_FAILED":
		s.Status = "Failed"
	case "CREATE_IN_PROGRESS":
		s.Status = "Creating"
	case "DELETE_IN_PROGRESS":
		s.Status = "Deleting"
	case "DELETE_COMPLETE":
		s.ObjectMeta.Finalizers = []string{}
	case "ROLLBACK_IN_PROGRESS", "UPDATE_ROLLBACK_IN_PROGRESS":
		s.Status = "Rollback"
	case "UPDATE_IN_PROGRESS":
		s.Status = "Updating"
	}

	switch status {
	case "CREATE_COMPLETE", "UPDATE_COMPLETE", "ROLLBACK_COMPLETE", "UPDATE_IN_PROGRESS", "UPDATE_ROLLBACK_COMPLETE":
		os, err := p.stackOutputs(stack)
		if err != nil {
			return err
		}

		s.Outputs = os

		cm, err := p.Provider.Cluster.CoreV1().ConfigMaps(p.AppNamespace(app)).Get(fmt.Sprintf("%s-%s", kind, name), am.GetOptions{})
		if err != nil {
			return err
		}
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		for k, v := range os {
			cm.Data[outputToEnvironment(k)] = v
		}
		if _, err := p.Provider.Cluster.CoreV1().ConfigMaps(p.AppNamespace(app)).Update(cm); err != nil {
			return err
		}
	}

	if _, err := c.ConvoxV1().Stacks(p.AppNamespace(app)).Update(s); err != nil {
		return err
	}

	return nil
}

func filterMessage(m *sqs.Message, max int) (bool, error) {
	if cs := m.Attributes["ApproximateReceiveCount"]; cs != nil {
		c, err := strconv.Atoi(*cs)
		if err != nil {
			return false, err
		}

		if c >= max {
			return true, nil
		}
	}

	if m.Body == nil {
		return true, nil
	}

	return false, nil
}

func (p *Provider) stackTagsCached(stack string) (map[string]string, error) {
	tags, ok := stackTagCache[stack]
	if ok {
		return tags, nil
	}

	tags, err := p.stackTags(stack)
	if err != nil {
		return nil, err
	}

	stackTagCache[stack] = tags

	return tags, nil
}

func topicAttributes(body string) (map[string]string, error) {
	var parts map[string]string

	if err := json.Unmarshal([]byte(body), &parts); err != nil {
		return nil, err
	}

	attrs := map[string]string{}

	mas := eventMessageSplitter.FindAllStringSubmatch(parts["Message"], -1)

	for _, ma := range mas {
		attrs[ma[1]] = ma[2]
	}

	return attrs, nil
}
