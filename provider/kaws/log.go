package kaws

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
)

var sequenceTokens sync.Map

func (p *Provider) Log(app, stream string, ts time.Time, message string) error {
	group := p.appLogGroup(app)

	req := &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(group),
		LogStreamName: aws.String(stream),
		LogEvents: []*cloudwatchlogs.InputLogEvent{
			{
				Timestamp: aws.Int64(ts.UnixNano() / int64(time.Millisecond)),
				Message:   aws.String(message),
			},
		},
	}

	key := fmt.Sprintf("%s/%s", *req.LogGroupName, *req.LogStreamName)

	if tv, ok := sequenceTokens.Load(key); ok {
		if token, ok := tv.(string); ok {
			req.SequenceToken = aws.String(token)
		}
	}

	for {
		res, err := p.CloudWatchLogs.PutLogEvents(req)
		switch helpers.AwsErrorCode(err) {
		case "ResourceNotFoundException":
			if strings.Contains(err.Error(), "log group") {
				if err := p.createLogGroup(app); err != nil {
					return err
				}
			}
			if err := p.createLogStream(group, stream); err != nil {
				return err
			}
		case "InvalidSequenceTokenException":
			token, err := p.nextSequenceToken(group, stream)
			if err != nil {
				return err
			}
			req.SequenceToken = aws.String(token)
		case "":
			sequenceTokens.Store(key, *res.NextSequenceToken)
			return nil
		default:
			return err
		}

		continue
	}

	return nil
}

func (p *Provider) AppLogs(name string, opts structs.LogsOptions) (io.ReadCloser, error) {
	return helpers.CloudWatchLogsSubscribe(p.Context(), p.CloudWatchLogs, p.appLogGroup(name), "", opts)
}

func (p *Provider) BuildLogs(app, id string, opts structs.LogsOptions) (io.ReadCloser, error) {
	b, err := p.BuildGet(app, id)
	if err != nil {
		return nil, err
	}

	switch b.Status {
	case "running":
		return p.ProcessLogs(app, b.Process, opts)
	default:
		u, err := url.Parse(b.Logs)
		if err != nil {
			return nil, err
		}

		switch u.Scheme {
		case "object":
			return p.ObjectFetch(u.Hostname(), u.Path)
		default:
			return nil, fmt.Errorf("unable to read logs for build: %s", id)
		}
	}
}

func (p *Provider) ProcessLogs(app, pid string, opts structs.LogsOptions) (io.ReadCloser, error) {
	ps, err := p.ProcessGet(app, pid)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("service/%s/%s", ps.Name, pid)

	ctx, cancel := context.WithCancel(p.Context())
	go p.watchForProcessTermination(ctx, app, pid, cancel)

	return helpers.CloudWatchLogsSubscribe(ctx, p.CloudWatchLogs, p.appLogGroup(app), key, opts)
}

func (p *Provider) SystemLogs(opts structs.LogsOptions) (io.ReadCloser, error) {
	return helpers.CloudWatchLogsSubscribe(p.Context(), p.CloudWatchLogs, p.appLogGroup("rack"), "", opts)
}

func (p *Provider) appLogGroup(app string) string {
	return fmt.Sprintf("%s-%s", p.Rack, app)
}

func (p *Provider) createLogGroup(app string) error {
	_, err := p.CloudWatchLogs.CreateLogGroup(&cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(p.appLogGroup(app)),
		Tags: map[string]*string{
			"system": aws.String("convox"),
			"rack":   aws.String(p.Rack),
			"app":    aws.String(app),
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) createLogStream(group, stream string) error {
	_, err := p.CloudWatchLogs.CreateLogStream(&cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(group),
		LogStreamName: aws.String(stream),
	})
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) nextSequenceToken(group, stream string) (string, error) {
	res, err := p.CloudWatchLogs.DescribeLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String(group),
		LogStreamNamePrefix: aws.String(stream),
	})
	if err != nil {
		return "", err
	}
	if len(res.LogStreams) != 1 {
		return "", fmt.Errorf("could not describe log stream: %s/%s", group, stream)
	}
	if res.LogStreams[0].UploadSequenceToken == nil {
		return "", fmt.Errorf("could not fetch sequence token for log stream: %s/%s", group, stream)
	}

	return *res.LogStreams[0].UploadSequenceToken, nil
}
