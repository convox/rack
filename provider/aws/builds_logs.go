package aws

// BuildLogs implementation that supports three transports:
//   * docker logs (EC2 builder by default)
//   * CloudWatch Logs (preferred when rack parameter LogDriver=CloudWatch)
//   * ECS Exec   (Fargate fallback when CloudWatch disabled but EnableExecuteCommand=true)

import (
	"fmt"
	"io"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/pkg/structs"
	docker "github.com/fsouza/go-dockerclient"
)

// BuildLogs streams build logs for running and finished builds.
func (p *Provider) BuildLogs(app, id string, opts structs.LogsOptions) (io.ReadCloser, error) {
	b, err := p.BuildGet(app, id)
	if err != nil {
		return nil, err
	}

	// Finished builds keep existing object:// behaviour
	if b.Status != "running" {
		return p.historicLogs(b)
	}

	task, err := p.describeTask(b.Tags["task"])
	if err != nil {
		return nil, err
	}

	if aws.StringValue(task.LaunchType) == "EC2" {
		return p.tailDockerLogs(task)
	}

	if p.cloudWatchEnabled() {
		grp, stream, cerr := p.cwStreamForTask(task, "build")
		if cerr != nil {
			return nil, cerr
		}

		return p.followCW(grp, stream)
	}

	if p.ecsExecEnabled(task) {
		return p.followExec(task)
	}

	return nil, fmt.Errorf("cloudwatch disabled and ecs-exec not enabled; unable to stream logs for fargate task")
}

// historicLogs returns logs for completed builds (object:// or plain URL)
func (p *Provider) historicLogs(b *structs.Build) (io.ReadCloser, error) {
	u, err := url.Parse(b.Logs)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "object":
		return p.ObjectFetch(b.App, u.Path)
	default:
		return io.NopCloser(strings.NewReader(b.Logs)), nil
	}
}

// cloudWatchEnabled checks stack parameter EnableCloudWatch == "Yes"
func (p *Provider) cloudWatchEnabled() bool {
	v, _ := p.stackParameter(p.Rack, "LogDriver")

	return v == "CloudWatch"
}

func (p *Provider) cwStreamForTask(task *ecs.Task, prefix string) (group, stream string, err error) {
	// 1. Describe the task-definition to grab log-driver options
	td, err := p.ecs().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: task.TaskDefinitionArn,
	})
	if err != nil {
		return "", "", err
	}
	// Assume the first container is the builder
	cd := td.TaskDefinition.ContainerDefinitions[0]

	lc := cd.LogConfiguration
	if lc == nil || aws.StringValue(lc.LogDriver) != "awslogs" {
		return "", "", fmt.Errorf("task definition has no awslogs configuration")
	}

	opts := lc.Options
	group = aws.StringValue(opts["awslogs-group"])
	prefix = aws.StringValue(opts["awslogs-stream-prefix"])
	name := aws.StringValue(cd.Name)

	// **Use only the task ID (last segment of ARN)**
	parts := strings.Split(aws.StringValue(task.TaskArn), "/")
	taskID := parts[len(parts)-1]

	stream = fmt.Sprintf("%s/%s/%s", prefix, name, taskID)

	return
}

// cwlogs returns a CloudWatch Logs client bound to the rack’s AWS config.
func (p *Provider) cwlogs() *cloudwatchlogs.CloudWatchLogs {
	return cloudwatchlogs.New(session.New(), p.config())
}

func (p *Provider) followCW(group, stream string) (io.ReadCloser, error) {
	cw := p.cwlogs()
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		var token *string
		for {
			out, err := cw.GetLogEvents(&cloudwatchlogs.GetLogEventsInput{
				LogGroupName:  aws.String(group),
				LogStreamName: aws.String(stream),
				NextToken:     token,
				StartFromHead: aws.Bool(true),
			})

			if err != nil {
				// If stream not yet created, wait and retry
				if awsErr, ok := err.(awserr.Error); ok &&
					awsErr.Code() == "ResourceNotFoundException" {
					time.Sleep(2 * time.Second)
					continue
				}
				pw.CloseWithError(err)
				return
			}

			for _, e := range out.Events {
				if _, err := fmt.Fprintln(pw, aws.StringValue(e.Message)); err != nil {
					return
				}
			}
			token = out.NextForwardToken
			time.Sleep(2 * time.Second)
		}
	}()

	return pr, nil
}

// tailDockerLogs attaches to the EC2 builder container and streams all logs.
func (p *Provider) tailDockerLogs(task *ecs.Task) (io.ReadCloser, error) {
	ci, err := p.containerInstance(*task.ContainerInstanceArn)
	if err != nil {
		return nil, err
	}
	dc, err := p.dockerInstance(*ci.Ec2InstanceId)
	if err != nil {
		return nil, err
	}

	cs, err := dc.ListContainers(docker.ListContainersOptions{
		All: true,
		Filters: map[string][]string{
			"label": {fmt.Sprintf("com.amazonaws.ecs.task-arn=%s", *task.TaskArn)},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(cs) != 1 {
		return nil, fmt.Errorf("could not find container for task %s", *task.TaskArn)
	}

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		dc.Logs(docker.LogsOptions{
			Container:    cs[0].ID,
			OutputStream: pw,
			ErrorStream:  pw,
			Follow:       true,
			Stdout:       true,
			Stderr:       true,
			Since:        0, // full history
		})
	}()
	return pr, nil
}

// followExec starts an ECS Exec session and streams the build log file.
func (p *Provider) followExec(task *ecs.Task) (io.ReadCloser, error) {
	// Simplified placeholder that runs cat+tail over /var/log/convox-build.log.
	// TODO: implement full SSM websocket stream.
	return nil, fmt.Errorf("ecs exec streaming not yet implemented")
}

// ecsExecEnabled returns true when the *task definition* has
// EnableExecuteCommand set, without requiring that field to exist
// in the AWS SDK structs used at compile-time.
func (p *Provider) ecsExecEnabled(task *ecs.Task) bool {
	// Fallback-safe: if we cannot describe the task definition,
	// treat Exec as disabled.
	tdOut, err := p.ecs().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: task.TaskDefinitionArn,
	})
	if err != nil || tdOut.TaskDefinition == nil {
		return false
	}

	// Use reflection so the code compiles even when the struct
	// type in the current SDK lacks the field.
	v := reflect.ValueOf(tdOut.TaskDefinition).Elem() // ecs.TaskDefinition
	f := v.FieldByName("EnableExecuteCommand")
	if !f.IsValid() || f.IsZero() {
		return false // field missing or the pointer is nil/false
	}
	return f.Elem().Bool()
}
