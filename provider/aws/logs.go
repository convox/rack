package aws

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/convox/rack/structs"
)

func (p *AWSProvider) subscribeLogs(group string, opts structs.LogsOptions) (io.ReadCloser, error) {
	r, w := io.Pipe()

	go p.streamLogs(w, group, opts)

	return r, nil
}

func (p *AWSProvider) streamLogs(w io.WriteCloser, group string, opts structs.LogsOptions) error {
	defer w.Close()

	since := opts.Since

	if since.IsZero() {
		since = time.Now().Add(10 * time.Minute)
	}

	start := since.UnixNano() / int64(time.Millisecond)

	for {
		s, err := p.fetchLogs(w, group, opts.Filter, start)
		if err != nil {
			return err
		}

		if !opts.Follow {
			return nil
		}

		start = s + 1

		time.Sleep(200 * time.Millisecond)
	}
}

// fetch logs until we run out of NextTokens, writing them the whole way
func (p *AWSProvider) fetchLogs(w io.Writer, group, filter string, start int64) (int64, error) {
	log := Logger.At("fetchLogs").Namespace("start=%d", start).Start()

	req := &cloudwatchlogs.FilterLogEventsInput{
		Interleaved:  aws.Bool(true),
		LogGroupName: aws.String(group),
		StartTime:    aws.Int64(start),
	}

	if filter != "" {
		req.FilterPattern = aws.String(filter)
	}

	end := start + 1

	for {
		res, err := p.cloudwatchlogs().FilterLogEvents(req)
		if ae, ok := err.(awserr.Error); ok && ae.Code() == "ThrottlingException" {
			// backoff
			log.Error(err)
			time.Sleep(1 * time.Second)
			continue
		}
		if err != nil {
			log.Error(err)
			return 0, err
		}

		latest, err := p.writeLogEvents(w, res.Events)
		if err != nil {
			log.Error(err)
			return 0, err
		}

		log = log.Namespace("events=%d", len(res.Events))

		if latest >= end {
			end = latest + 1
		}

		if res.NextToken == nil {
			break
		}

		req.NextToken = res.NextToken
	}

	log.Successf("end=%d", end)
	return end, nil
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
					return 0, err
				}

				prefix = fmt.Sprintf("%s/%s:%s/%s ", name, parts[1], release, arnToPid(parts[2]))
			}
		case "system":
			prefix = fmt.Sprintf("system:%s/", os.Getenv("RELEASE"))
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
