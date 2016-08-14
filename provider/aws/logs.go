package aws

import (
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/convox/rack/api/structs"
)

func (p *AWSProvider) LogStream(app string, w io.Writer, opts structs.LogStreamOptions) error {
	a, err := p.AppGet(app)
	if err != nil {
		return err
	}

	since := 2 * time.Minute
	if opts.Since.Nanoseconds() > 0 {
		since = opts.Since
	}

	startTime := aws.Int64(time.Now().Add(-since).UnixNano() / int64(time.Millisecond)) // number of milliseconds since Jan 1, 1970 00:00:00 UTC
	nextToken := aws.String("")

	for {
		req := &cloudwatchlogs.FilterLogEventsInput{
			Interleaved:  aws.Bool(true),
			LogGroupName: aws.String(a.Outputs["LogGroup"]),
			StartTime:    startTime,
		}

		if opts.Filter != "" {
			req.FilterPattern = aws.String(opts.Filter)
		}

		startTime, nextToken, err = p.writeLogEvents(req, w)
		if err != nil {
			return err
		}

		for nextToken != nil {
			req.NextToken = nextToken

			startTime, nextToken, err = p.writeLogEvents(req, w)
			if err != nil {
				return err
			}
		}

		// assert that websocket is still alive
		_, err = w.Write([]byte{})
		if err != nil {
			return err
		}

		if !opts.Follow {
			return nil
		}

		// According to http://docs.aws.amazon.com/AmazonCloudWatch/latest/DeveloperGuide/cloudwatch_limits.html
		// the maximum rate of a GetLogEvents request is 10 requests per second per AWS account.
		// Aim for 5 reqs / sec so two clients can tail.
		time.Sleep(200 * time.Millisecond)
	}

	return nil
}

func (p *AWSProvider) writeLogEvents(req *cloudwatchlogs.FilterLogEventsInput, w io.Writer) (*int64, *string, error) {
	resp, err := p.cloudwatchlogs().FilterLogEvents(req)
	if code := awsError(err); code == "ThrottlingException" {
		// Backoff but don't return an error
		fmt.Printf("logs writeLogEvents err=%q\n", code)
		time.Sleep(1 * time.Second)
		return req.StartTime, req.NextToken, nil
	}
	if err != nil {
		fmt.Printf("logs writeLogEvents err=%q\n", err)
		return req.StartTime, req.NextToken, err
	}

	ts := req.StartTime

	for _, e := range resp.Events {
		if *e.Timestamp >= *ts {
			*ts = *e.Timestamp + int64(1)
		}

		sec := *e.Timestamp / 1000
		nsec := *e.Timestamp - (sec * 1000)
		t := time.Unix(sec, nsec)

		line := fmt.Sprintf("%s %s\n", t.Format(time.RFC3339), *e.Message)
		_, err := w.Write([]byte(line))
		if err != nil {
			return ts, req.NextToken, err
		}
	}

	// return latest timestamp and NextToken seen in the response
	return ts, resp.NextToken, nil
}
