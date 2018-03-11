package aws

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/convox/rack/structs"
)

func (p *AWSProvider) subscribeLogs(group string, opts structs.LogsOptions) (io.ReadCloser, error) {
	r, w := io.Pipe()

	go p.streamLogs(w, group, opts)

	return r, nil
}

func (p *AWSProvider) streamLogs(w io.WriteCloser, group string, opts structs.LogsOptions) error {
	log := Logger.At("streamLogs").Namespace("group=%s", group)

	defer w.Close()

	req := &cloudwatchlogs.FilterLogEventsInput{
		Interleaved:  aws.Bool(true),
		LogGroupName: aws.String(group),
	}

	if opts.Filter != "" {
		log = log.Namespace("filter=%s", opts.Filter)
		req.FilterPattern = aws.String(opts.Filter)
	}

	var start int64

	if !opts.Since.IsZero() {
		start = opts.Since.UnixNano() / int64(time.Millisecond)
		log = log.Namespace("start=%d", start)
		req.StartTime = aws.Int64(start)
	}

	for {
		// check for closed connection
		if _, err := w.Write([]byte{}); err != nil {
			break
		}

		time.Sleep(1 * time.Second)

		res, err := p.cloudwatchlogs().FilterLogEvents(req)
		if err != nil {
			return err
		}

		latest, err := p.writeLogEvents(w, res.Events)
		if err != nil {
			return nil
		}

		if latest > start {
			start = latest + 1
		}

		log.Success()

		if res.NextToken != nil {
			req.NextToken = res.NextToken
			continue
		}

		req.NextToken = nil

		if !opts.Follow {
			break
		}

		if start > 0 {
			log = log.Replace("start", fmt.Sprintf("%d", start))
			req.StartTime = aws.Int64(start)
		}
	}

	log.Successf("done=true")

	return nil
}

func (p *AWSProvider) writeLogEvents(w io.Writer, events []*cloudwatchlogs.FilteredLogEvent) (int64, error) {
	if len(events) == 0 {
		return 0, nil
	}

	log := Logger.At("writeLogEvents").Namespace("events=%d", len(events)).Start()

	sort.Slice(events, func(i, j int) bool { return *events[i].Timestamp < *events[j].Timestamp })

	latest := int64(0)

	for _, e := range events {
		if *e.Timestamp > latest {
			latest = *e.Timestamp
		}

		prefix := ""

		switch name := strings.Split(*e.LogStreamName, "/")[0]; name {
		case "service", "timer":
			parts := strings.Split(*e.LogStreamName, "/")

			if len(parts) >= 3 {
				release, err := p.taskRelease(parts[2])
				if err != nil {
					prefix = fmt.Sprintf("%s/%s:%s ", name, parts[1], arnToPid(parts[2]))
				} else {
					prefix = fmt.Sprintf("%s/%s:%s/%s ", name, parts[1], release, arnToPid(parts[2]))
				}
			}
		case "system":
			prefix = "system/"
		}

		sec := *e.Timestamp / 1000
		nsec := *e.Timestamp - (sec * 1000)
		t := time.Unix(sec, nsec).UTC()

		line := fmt.Sprintf("%s %s%s\n", t.Format(time.RFC3339), prefix, *e.Message)

		if _, err := w.Write([]byte(line)); err != nil {
			log.Error(err)
			return 0, err
		}
	}

	log.Success()

	return latest, nil
}
