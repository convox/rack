package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/lambda"
)

var (
	AutoScaling = autoscaling.New(session.New(), nil)
	ECS         = ecs.New(session.New(), nil)
	ELB         = elb.New(session.New(), nil)
	Lambda      = lambda.New(session.New(), nil)
)

type Event struct {
	Records []Record
}

type Message struct {
	AutoScalingGroupName string
	EC2InstanceID        string
	LifecycleActionToken string
	LifecycleHookName    string
	LifecycleTransition  string
}

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
	var m Message

	if err := json.Unmarshal([]byte(r.Sns.Message), &m); err != nil {
		return err
	}

	fmt.Printf("m = %+v\n", m)

	if m.LifecycleTransition != "autoscaling:EC2_INSTANCE_TERMINATING" {
		return nil
	}

	md, err := metadata()
	if err != nil {
		return err
	}

	fmt.Printf("md = %+v\n", md)

	ci, err := containerInstance(md.Cluster, m.EC2InstanceID)
	if err != nil {
		return err
	}

	fmt.Printf("ci = %+v\n", ci)

	cis, err := ECS.UpdateContainerInstancesState(&ecs.UpdateContainerInstancesStateInput{
		ContainerInstances: []*string{
			aws.String(ci),
		},
		Status:  aws.String("DRAINING"),
		Cluster: aws.String(md.Cluster),
	})
	if err != nil {
		return err
	}

	if len(cis.Failures) > 0 {
		return fmt.Errorf("unable to drain instance: %s - %s", ci, *cis.Failures[0].Reason)
	}

	if err := waitForInstanceDrain(md.Cluster, ci); err != nil {
		return err
	}

	if _, err := ECS.DeregisterContainerInstance(&ecs.DeregisterContainerInstanceInput{
		Cluster:           aws.String(md.Cluster),
		ContainerInstance: aws.String(ci),
		Force:             aws.Bool(true),
	}); err != nil {
		return err
	}

	_, err = AutoScaling.CompleteLifecycleAction(&autoscaling.CompleteLifecycleActionInput{
		AutoScalingGroupName:  aws.String(m.AutoScalingGroupName),
		InstanceId:            aws.String(m.EC2InstanceID),
		LifecycleActionResult: aws.String("CONTINUE"),
		LifecycleActionToken:  aws.String(m.LifecycleActionToken),
		LifecycleHookName:     aws.String(m.LifecycleHookName),
	})
	if err != nil {
		return err
	}

	fmt.Println("success")

	return nil
}

func waitForInstanceDrain(cluster, ci string) error {

	params := &ecs.ListTasksInput{
		Cluster:           aws.String(cluster),
		ContainerInstance: aws.String(ci),
		DesiredStatus:     aws.String("RUNNING"),
	}

	tasks := []*string{}

	for {
		resp, err := ECS.ListTasks(params)
		if err != nil {
			return err
		}

		tasks = append(tasks, resp.TaskArns...)

		if resp.NextToken == nil {
			break
		}

		params.NextToken = resp.NextToken
		time.Sleep(2 * time.Second)
	}

	input := &ecs.DescribeTasksInput{
		Cluster: aws.String(cluster),
		Tasks:   tasks,
	}
	return ECS.WaitUntilTasksStopped(input)
}

func metadata() (*Metadata, error) {
	var md Metadata

	fres, err := Lambda.GetFunction(&lambda.GetFunctionInput{
		FunctionName: aws.String(os.Getenv("AWS_LAMBDA_FUNCTION_NAME")),
	})
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(*fres.Configuration.Description), &md); err != nil {
		return nil, err
	}

	return &md, nil
}

func containerInstance(cluster string, id string) (string, error) {
	lreq := &ecs.ListContainerInstancesInput{
		Cluster:    aws.String(cluster),
		MaxResults: aws.Int64(10),
	}

	for {
		lres, err := ECS.ListContainerInstances(lreq)
		if err != nil {
			return "", err
		}

		dres, err := ECS.DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
			Cluster:            aws.String(cluster),
			ContainerInstances: lres.ContainerInstanceArns,
		})
		if err != nil {
			return "", err
		}

		for _, ci := range dres.ContainerInstances {
			if *ci.Ec2InstanceId == id {
				return *ci.ContainerInstanceArn, nil
			}
		}

		if lres.NextToken == nil {
			break
		}

		lreq.NextToken = lres.NextToken
	}

	return "", fmt.Errorf("could not find cluster instance: %s", id)
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	os.Exit(1)
}
