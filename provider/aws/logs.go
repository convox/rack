package aws

import (
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/convox/rack/api/structs"
)

func (p *AWSProvider) LogStream(app string, w io.Writer, opts structs.LogStreamOptions) error {
	a, err := p.AppGet(app)
	if err != nil {
		return err
	}

	return p.subscribeLogs(w, a.Outputs["LogGroup"], opts)
}

func (p *AWSProvider) subscribeLogs(w io.Writer, group string, opts structs.LogStreamOptions) error {
	since := opts.Since

	if since.IsZero() {
		since = time.Now().Add(10 * time.Minute)
	}

	// number of milliseconds since Jan 1, 1970 00:00:00 UTC
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
		EndTime:      aws.Int64(start + (1000 * 60 * 10)),
		Limit:        aws.Int64(10000),
	}

	if filter != "" {
		req.FilterPattern = aws.String(filter)
	}

	events := []*cloudwatchlogs.FilteredLogEvent{}

	for {
		res, err := p.cloudwatchlogs().FilterLogEvents(req)
		if ae, ok := err.(awserr.Error); ok && ae.Code() == "ThrottlingException" {
			// backoff
			log.Error(err)
			time.Sleep(200 * time.Millisecond)
			continue
		}
		if err != nil {
			log.Error(err)
			return 0, err
		}

		events = append(events, res.Events...)

		if res.NextToken == nil {
			break
		}

		req.NextToken = res.NextToken
	}

	latest, err := p.writeLogEvents(w, events)
	if err != nil {
		log.Error(err)
		return 0, err
	}

	log.Successf("end=%d", latest)
	return latest, nil
}

func (p *AWSProvider) writeLogEvents(w io.Writer, events []*cloudwatchlogs.FilteredLogEvent) (int64, error) {
	if len(events) == 0 {
		return 0, nil
	}

	log := Logger.At("writeLogEvents").Namespace("events=%d", len(events)).Start()

	sorted := cloudwatchEvents(events)
	sort.Sort(sorted)

	latest := int64(0)

	for _, e := range sorted {
		if *e.Timestamp > latest {
			latest = *e.Timestamp
		}

		sec := *e.Timestamp / 1000
		nsec := *e.Timestamp - (sec * 1000)
		t := time.Unix(sec, nsec).UTC()
		line := fmt.Sprintf("%s %s\n", t.Format(time.RFC3339), *e.Message)

		if _, err := w.Write([]byte(line)); err != nil {
			log.Error(err)
			return 0, err
		}
	}

	log.Success()
	return latest, nil
}

type cloudwatchEvents []*cloudwatchlogs.FilteredLogEvent

func (e cloudwatchEvents) Len() int           { return len(e) }
func (e cloudwatchEvents) Less(i, j int) bool { return *(e[i].Timestamp) < *(e[j].Timestamp) }
func (e cloudwatchEvents) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
