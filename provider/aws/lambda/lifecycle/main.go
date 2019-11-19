package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ecs"
)

var (
	AutoScaling = autoscaling.New(session.New(), nil)
	ECS         = ecs.New(session.New(), nil)
)

type Interruption struct {
	Account    string            `json:"account"`
	Detail     map[string]string `json:"detail"`
	DetailType string            `json:"detail-type"`
	ID         string            `json:"id"`
	Region     string            `json:"region"`
	Resources  []string          `json:"resources"`
	Source     string            `json:"source"`
	Time       string            `json:"time"`
	Version    string            `json:"version"`
}

type Termination struct {
	AutoScalingGroupName string
	EC2InstanceID        string
	LifecycleActionToken string
	LifecycleHookName    string
	LifecycleTransition  string
}

func main() {
	lambda.Start(Handler)
}

func Handler(ctx context.Context, event events.SNSEvent) error {
	fmt.Printf("event = %+v\n", event)

	for _, r := range event.Records {
		switch {
		case strings.HasPrefix(r.SNS.Subject, "Auto Scaling"):
			if err := handleAutoscaling(r); err != nil {
				fmt.Printf("err = %+v\n", err)
			}
		default:
			fmt.Printf("unknown event: %v\n", r)
		}
	}

	return nil
}

func handleAutoscaling(r events.SNSEventRecord) error {
	fmt.Println("handleAutoscaling")

	var m Termination

	if err := json.Unmarshal([]byte(r.SNS.Message), &m); err != nil {
		return err
	}

	fmt.Printf("m = %+v\n", m)

	if m.LifecycleTransition == "autoscaling:EC2_INSTANCE_TERMINATING" {
		if err := drainInstance(m.EC2InstanceID); err != nil {
			fmt.Printf("err = %+v\n", err)
		}
	}

	_, err := AutoScaling.CompleteLifecycleAction(&autoscaling.CompleteLifecycleActionInput{
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

func drainInstance(id string) error {
	cluster := os.Getenv("CLUSTER")

	ci, err := containerInstance(cluster, id)
	if err != nil {
		return err
	}

	fmt.Printf("ci = %+v\n", ci)

	cis, err := ECS.UpdateContainerInstancesState(&ecs.UpdateContainerInstancesStateInput{
		ContainerInstances: []*string{
			aws.String(ci),
		},
		Status:  aws.String("DRAINING"),
		Cluster: aws.String(cluster),
	})
	if err != nil {
		return err
	}

	if len(cis.Failures) > 0 {
		return fmt.Errorf("unable to drain instance: %s - %s", ci, *cis.Failures[0].Reason)
	}

	if err := waitForInstanceDrain(cluster, ci); err != nil {
		return err
	}

	fmt.Println("instance has been drained")

	_, err = ECS.DeregisterContainerInstance(&ecs.DeregisterContainerInstanceInput{
		Cluster:           aws.String(cluster),
		ContainerInstance: aws.String(ci),
		Force:             aws.Bool(true),
	})
	if err != nil {
		return err
	}

	return nil
}

func remarshal(v, w interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &w)
}

// stopServicelessTasks stops one-off tasks that do not belog to a ECS service.
// For example, a scheduled task or running a process
func stopServicelessTasks(input *ecs.DescribeTasksInput) error {
	tasks, err := ECS.DescribeTasks(input)
	if err != nil {
		return err
	}

	for _, t := range tasks.Tasks {
		// if the task isn't part of a service and wasn't started by ECS, stop it
		if !strings.HasPrefix(*t.Group, "service:") && !strings.HasPrefix(*t.StartedBy, "ecs-svc") {
			_, err := ECS.StopTask(&ecs.StopTaskInput{
				Cluster: input.Cluster,
				Reason:  aws.String("draining instance for termination"),
				Task:    t.TaskArn,
			})
			if err != nil {
				return err
			}
		}
	}

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
		time.Sleep(1 * time.Second)
	}

	if len(tasks) == 0 {
		fmt.Println("no tasks to wait for")
		return nil
	}

	input := &ecs.DescribeTasksInput{
		Cluster: aws.String(cluster),
		Tasks:   tasks,
	}

	if err := stopServicelessTasks(input); err != nil {
		return err
	}

	fmt.Println("stopped service-less tasks")

	return ECS.WaitUntilTasksStopped(input)
}
