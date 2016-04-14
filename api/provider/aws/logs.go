package aws

import (
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

	startTime := time.Now().Add(-2*time.Minute).UnixNano() / int64(time.Millisecond) // number of milliseconds since Jan 1, 1970 00:00:00 UTC for 1 minute ago

	for {
		req := &cloudwatchlogs.FilterLogEventsInput{
			Interleaved:  aws.Bool(true),
			LogGroupName: aws.String(a.Outputs["LogGroup"]),
			StartTime:    aws.Int64(startTime),
		}

		resp, err := p.cloudwatchlogs().FilterLogEvents(req)
		if err != nil {
			return err
		}

		err = writeEvents(w, resp.Events, &startTime)
		if err != nil {
			return err
		}

		nextToken := resp.NextToken

		for nextToken != nil {
			req.NextToken = nextToken

			resp, err := p.cloudwatchlogs().FilterLogEvents(req)
			if err != nil {
				return err
			}

			err = writeEvents(w, resp.Events, &startTime)
			if err != nil {
				return err
			}

			nextToken = resp.NextToken
		}

		_, err = w.Write([]byte{}) // keepalive
		if err != nil {
			return err
		}

		time.Sleep(20 * time.Millisecond)
	}

	return nil
}

func writeEvents(w io.Writer, events []*cloudwatchlogs.FilteredLogEvent, ts *int64) error {
	for _, e := range events {
		if *e.Timestamp > *ts {
			*ts = *e.Timestamp + int64(1)
		}

		_, err := w.Write([]byte(*e.Message + "\n"))
		if err != nil {
			return err
		}
	}

	return nil
}
