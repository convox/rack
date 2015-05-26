package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ecs"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/sqs"
)

var MessageQueueUrl = os.Getenv("FORMATION_QUEUE")

type Message struct {
	MessageID     *string
	ReceiptHandle *string

	Type             string
	MessageId        string
	TopicArn         string
	Subject          string
	Message          string
	Timestamp        time.Time
	SignatureVersion string
	Signature        string
	SigningCertURL   string
	UnsubscribeURL   string
}

type FormationRequest struct {
	ResourceType string
	RequestType  string

	RequestId          string
	StackId            string
	LogicalResourceId  string
	PhysicalResourceId string
	ResponseURL        string

	ResourceProperties map[string]interface{}
}

type FormationResponse struct {
	RequestId         string
	StackId           string
	LogicalResourceId string

	Data               map[string]string
	PhysicalResourceId string
	Reason             string
	Status             string
}

func ListenFormation() {
	for {
		messages, err := dequeueMessage()

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			continue
		}

		if len(messages) == 0 {
			continue
		}

		for _, message := range messages {
			if message.Subject == "AWS CloudFormation custom resource request" {
				handleFormation(message)
			}
		}

		num, err := ackMessage(messages)

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			continue
		}

		fmt.Printf("success: messages=%d\n", num)
	}
}

func handleECSService(freq FormationRequest) (string, error) {
	switch freq.RequestType {
	case "Create":
		fmt.Println("CREATING SERVICE")
		fmt.Printf("freq %+v\n", freq)

		count, err := strconv.Atoi(freq.ResourceProperties["DesiredCount"].(string))

		if err != nil {
			return "", err
		}

		req := &ecs.CreateServiceInput{
			Cluster:        aws.String(freq.ResourceProperties["Cluster"].(string)),
			DesiredCount:   aws.Long(int64(count)),
			ServiceName:    aws.String(freq.ResourceProperties["Name"].(string)),
			TaskDefinition: aws.String(freq.ResourceProperties["TaskDefinition"].(string)),
			Role:           aws.String(freq.ResourceProperties["Role"].(string)),
		}

		balancers := freq.ResourceProperties["LoadBalancers"].([]interface{})

		req.LoadBalancers = make([]*ecs.LoadBalancer, len(balancers))

		for i, balancer := range balancers {
			parts := strings.Split(balancer.(string), ":")
			name := parts[0]
			port, _ := strconv.Atoi(parts[1])
			req.LoadBalancers[i] = &ecs.LoadBalancer{
				ContainerName:    aws.String("main"),
				LoadBalancerName: aws.String(name),
				ContainerPort:    aws.Long(int64(port)),
			}
		}

		res, err := ECS().CreateService(req)

		if err != nil {
			return "", err
		}

		return *res.Service.ServiceARN, nil
	case "Update":
		fmt.Println("UPDATING SERVICE")
		fmt.Printf("freq %+v\n", freq)

		count, _ := strconv.Atoi(freq.ResourceProperties["DesiredCount"].(string))

		req := &ecs.UpdateServiceInput{
			Cluster:        aws.String(freq.ResourceProperties["Cluster"].(string)),
			Service:        aws.String(freq.ResourceProperties["Name"].(string)),
			DesiredCount:   aws.Long(int64(count)),
			TaskDefinition: aws.String(freq.ResourceProperties["TaskDefinition"].(string)),
		}

		res, err := ECS().UpdateService(req)

		if err != nil {
			return "", err
		}

		return *res.Service.ServiceARN, nil
	case "Delete":
		fmt.Println("DELETING SERVICE")
		fmt.Printf("freq %+v\n", freq)

		cluster := freq.ResourceProperties["Cluster"].(string)
		name := freq.ResourceProperties["Name"].(string)

		req := &ecs.UpdateServiceInput{
			Cluster:      aws.String(cluster),
			Service:      aws.String(name),
			DesiredCount: aws.Long(0),
		}

		_, err := ECS().UpdateService(req)

		// go ahead and mark the delete good if the service is not found
		if ae, ok := err.(aws.APIError); ok {
			if ae.Code == "ServiceNotFoundException" {
				return "", nil
			}
		}

		if err != nil {
			return "", err
		}

		_, err = ECS().DeleteService(&ecs.DeleteServiceInput{
			Cluster: aws.String(cluster),
			Service: aws.String(name),
		})

		if err != nil {
			return "", err
		}

		return "", nil
	}

	return "", fmt.Errorf("unknown RequestType: %s", freq.RequestType)
}

func handleECSTaskDefinition(freq FormationRequest) (string, error) {
	switch freq.RequestType {
	case "Create", "Update":
		fmt.Printf("%sing TASK\n", freq.RequestType)
		fmt.Printf("freq %+v\n", freq)

		cpu, _ := strconv.Atoi(freq.ResourceProperties["CPU"].(string))
		memory, _ := strconv.Atoi(freq.ResourceProperties["Memory"].(string))

		req := &ecs.RegisterTaskDefinitionInput{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					CPU:       aws.Long(int64(cpu)),
					Essential: aws.Boolean(true),
					Image:     aws.String(freq.ResourceProperties["Image"].(string)),
					Memory:    aws.Long(int64(memory)),
					Name:      aws.String("main"),
				},
			},
			Family: aws.String(freq.ResourceProperties["Name"].(string)),
		}

		if command := freq.ResourceProperties["Command"].(string); command != "" {
			req.ContainerDefinitions[0].Command = []*string{aws.String("sh"), aws.String("-c"), aws.String(command)}
		}

		// set environment
		// env := LoadEnvironment([]byte(r.Env))

		// for key, val := range env {
		//   req.ContainerDefinitions[0].Environment = append(req.ContainerDefinitions[0].Environment, &ecs.KeyValuePair{
		//     Name:  aws.String(key),
		//     Value: aws.String(val),
		//   })
		// }

		// set portmappings
		ports := freq.ResourceProperties["PortMappings"].([]interface{})

		req.ContainerDefinitions[0].PortMappings = make([]*ecs.PortMapping, len(ports))

		for i, port := range ports {
			parts := strings.Split(port.(string), ":")
			host, _ := strconv.Atoi(parts[0])
			container, _ := strconv.Atoi(parts[1])

			req.ContainerDefinitions[0].PortMappings[i] = &ecs.PortMapping{
				ContainerPort: aws.Long(int64(container)),
				HostPort:      aws.Long(int64(host)),
			}
		}

		res, err := ECS().RegisterTaskDefinition(req)

		if err != nil {
			return "", err
		}

		return *res.TaskDefinition.TaskDefinitionARN, nil
	case "Delete":
		fmt.Println("DELETING TASK")
		fmt.Printf("freq %+v\n", freq)

		// TODO: currently unsupported by ECS
		// res, err := ECS().DeregisterTaskDefinition(&ecs.DeregisterTaskDefinitionInput{TaskDefinition: aws.String(freq.PhysicalResourceId)})

		return "", nil
	}

	return "", fmt.Errorf("unknown RequestType: %s", freq.RequestType)
}

func handleFormation(message Message) {
	var freq FormationRequest

	err := json.Unmarshal([]byte(message.Message), &freq)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return
	}

	physical := ""

	switch freq.ResourceType {
	case "Custom::ECSService":
		physical, err = handleECSService(freq)
	case "Custom::ECSTaskDefinition":
		physical, err = handleECSTaskDefinition(freq)
	}

	fmt.Printf("physical %+v\n", physical)
	fmt.Printf("err %+v\n", err)

	fres := FormationResponse{
		RequestId:          freq.RequestId,
		StackId:            freq.StackId,
		LogicalResourceId:  freq.LogicalResourceId,
		PhysicalResourceId: physical,
		Status:             "SUCCESS",
		Data: map[string]string{
			"Output1": "Value1",
		},
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		fres.Reason = err.Error()
		fres.Status = "FAILED"
	}

	data, err := json.Marshal(fres)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return
	}

	req, _ := http.NewRequest("PUT", "", bytes.NewBuffer(data))

	// golang's http methods munge the %3A in amazon urls so we build it manually using Opaque
	rurl := freq.ResponseURL
	parts := strings.SplitN(rurl, "/", 4)
	req.URL.Scheme = parts[0][0 : len(parts[0])-1]
	req.URL.Host = parts[2]
	req.URL.Opaque = fmt.Sprintf("//%s/%s", parts[2], parts[3])

	client := &http.Client{}

	res, err := client.Do(req)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return
	}

	rr, _ := ioutil.ReadAll(res.Body)

	fmt.Printf("string(rr) %+v\n", string(rr))
}

func dequeueMessage() ([]Message, error) {
	req := &sqs.ReceiveMessageInput{
		MaxNumberOfMessages: aws.Long(10),
		QueueURL:            aws.String(MessageQueueUrl),
		WaitTimeSeconds:     aws.Long(10),
	}

	res, err := SQS().ReceiveMessage(req)

	if err != nil {
		return nil, err
	}

	messages := make([]Message, len(res.Messages))

	var message Message

	for i, m := range res.Messages {
		err = json.Unmarshal([]byte(*m.Body), &message)

		if err != nil {
			return nil, err
		}

		message.MessageID = m.MessageID
		message.ReceiptHandle = m.ReceiptHandle

		messages[i] = message
	}

	return messages, nil
}

func ackMessage(messages []Message) (int, error) {
	dreq := &sqs.DeleteMessageBatchInput{
		QueueURL: aws.String(MessageQueueUrl),
	}

	dreq.Entries = make([]*sqs.DeleteMessageBatchRequestEntry, len(messages))

	for i, message := range messages {
		dreq.Entries[i] = &sqs.DeleteMessageBatchRequestEntry{
			ID:            message.MessageID,
			ReceiptHandle: message.ReceiptHandle,
		}
	}

	_, err := SQS().DeleteMessageBatch(dreq)

	return len(messages), err
}
