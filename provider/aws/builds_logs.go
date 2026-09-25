package aws

// BuildLogs implementation that supports three transports:
//   * docker logs (EC2 builder by default)
//   * CloudWatch Logs (when the build task definition uses the awslogs driver)
//   * returns an error if neither is available

import (
	"fmt"
	"io"
	"net/url"
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

	// EC2 path (docker logs)
	if aws.StringValue(task.LaunchType) == "EC2" {
		return p.dockerBuildLogs(app, id, task)
	}

	// ── create a channel that closes when the task stops ────────────────
	done := make(chan struct{})
	go p.waitTaskStopped(app, id, *task.TaskArn, done) // non-blocking

	// Fargate path
	group, stream, err := p.cwStreamForTask(task, "build")
	if err != nil {
		return nil, err
	}

	return p.followCW(app, id, group, stream, done)
}

// waitTaskStopped closes done once the task stops, first failing a build the task left unfinished.
func (p *Provider) waitTaskStopped(app, id, taskArn string, done chan<- struct{}) {
	for {
		td, err := p.describeTask(taskArn)
		if err == nil && aws.StringValue(td.LastStatus) == "STOPPED" {
			p.failBuildIfStopped(app, id, taskArn)
			close(done)
			return
		}
		time.Sleep(2 * time.Second)
	}
}

// followCW streams log events from an AWS CloudWatch Logs log group and log stream.
func (p *Provider) followCW(app, id, group, stream string, done <-chan struct{}) (io.ReadCloser, error) {
	cw := p.cwlogs()
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		var prevToken, token *string
		for {
			out, err := cw.GetLogEvents(&cloudwatchlogs.GetLogEventsInput{
				LogGroupName:  aws.String(group),
				LogStreamName: aws.String(stream),
				NextToken:     token,
				StartFromHead: aws.Bool(true),
			})

			// stream not yet created → keep retrying until the task stops
			if isNotFound(err) {
				select {
				case <-done:
					p.writeBuildStopLine(pw, app, id)
					return
				default:
				}
				time.Sleep(2 * time.Second)
				continue
			}
			if err != nil {
				pw.CloseWithError(err)
				return
			}

			// emit any new events
			for _, e := range out.Events {
				fmt.Fprintln(pw, aws.StringValue(e.Message))
			}

			prevToken, token = token, out.NextForwardToken

			// nothing new AND task finished → we’re done
			if aws.StringValue(prevToken) == aws.StringValue(token) {
				select {
				case <-done:
					p.writeBuildStopLine(pw, app, id)
					return
				default:
				}
			}

			time.Sleep(2 * time.Second)
		}
	}()

	return pr, nil
}

// writeBuildStopLine ends a build log stream with the reason its task stopped, if the build failed without logs.
func (p *Provider) writeBuildStopLine(w io.Writer, app, id string) {
	b, err := p.BuildGet(app, id)
	if err != nil {
		return
	}

	if line := buildStopLine(b); line != "" {
		_, _ = fmt.Fprint(w, line)
	}
}

// isNotFound checks if the provided error is a "ResourceNotFoundException" error.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	if ae, ok := err.(awserr.Error); ok && ae.Code() == "ResourceNotFoundException" {
		return true
	}
	return false
}

// historicLogs returns logs for completed builds (object:// or plain URL)
func (p *Provider) historicLogs(b *structs.Build) (io.ReadCloser, error) {
	if line := buildStopLine(b); line != "" {
		return io.NopCloser(strings.NewReader(line)), nil
	}

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

// buildStopLine reports why the task of a build that failed without logs stopped.
func buildStopLine(b *structs.Build) string {
	if b.Status != "failed" || b.Logs != "" || b.Reason == "" || b.Tags["task"] == "" {
		return ""
	}

	return fmt.Sprintf("build task stopped: %s\n", b.Reason)
}

// cwStreamForTask retrieves the CloudWatch log group and log stream name for a given ECS task.
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

// dockerBuildLogs tails an EC2 build container once its task starts, or returns what the build stored.
func (p *Provider) dockerBuildLogs(app, id string, task *ecs.Task) (io.ReadCloser, error) {
	if started, _ := p.waitForBuildTask(app, id, *task.TaskArn, time.Now().Add(taskStartTimeout)); started {
		r, err := p.tailDockerLogs(task)
		if err == nil {
			return r, nil
		}

		_ = Logger.At("dockerBuildLogs").Namespace("app=%q id=%q", app, id).Error(err)
	}

	b, err := p.BuildGet(app, id)
	if err != nil {
		return nil, err
	}

	return p.historicLogs(b)
}

// tailDockerLogs attaches to the EC2 builder container and streams all logs.
func (p *Provider) tailDockerLogs(task *ecs.Task) (io.ReadCloser, error) {
	if err := p.waitForContainer(task); err != nil {
		return nil, err
	}

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
