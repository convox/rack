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
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/api/structs"
)

func (p *AWSProvider) LogStream(app string, w io.Writer, opts structs.LogStreamOptions) error {
	logGroup, err := p.stackResource(fmt.Sprintf("%s-%s", p.Rack, app), "LogGroup")
	if err != nil {
		return err
	}

	return p.subscribeLogs(w, *logGroup.PhysicalResourceId, opts)
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

var (
	releases        = make(map[string]string) // task definition to release id map
	taskDefinitions = make(map[string]string) // task to task definition map
)

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

		prefix := ""

		// turn log stream with name `prefix-name/container-name/ecs-task-id`
		// into a log line prefix like `web:RRQSKTNDULA/0b4127231289`
		if strings.HasPrefix(*e.LogStreamName, "convox/") {
			parts := strings.Split(*e.LogStreamName, "/")

			if len(parts) == 3 {
				name := parts[1]
				taskID := parts[2]
				shortID := strings.Replace(taskID, "-", "", -1)[0:12]

				// if task has never been seen, get its task definition
				if _, ok := taskDefinitions[taskID]; !ok {
					t, err := p.ecs().DescribeTasks(&ecs.DescribeTasksInput{
						Cluster: aws.String(os.Getenv("CLUSTER")),
						Tasks:   []*string{aws.String(taskID)},
					})
					if err != nil {
						return 0, err
					}

					if len(t.Tasks) > 0 {
						taskDefinitions[taskID] = *t.Tasks[0].TaskDefinitionArn
					}
				}

				// if task definition has never been seen, get its RELEASE env var
				if tdARN, ok := taskDefinitions[taskID]; ok {
					if _, ok := releases[tdARN]; !ok {
						td, err := p.ecs().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
							TaskDefinition: aws.String(tdARN),
						})
						if err != nil {
							return 0, err
						}

						if len(td.TaskDefinition.ContainerDefinitions) > 0 {
							for _, e := range td.TaskDefinition.ContainerDefinitions[0].Environment {
								if *e.Name == "RELEASE" {
									releases[tdARN] = *e.Value
								}
							}
						}
					}
				}

				// lookup cached task definition and release
				tdARN := taskDefinitions[taskID]
				release := releases[tdARN]

				if release == "" {
					release = "RUNKNOWN"
				}

				prefix = fmt.Sprintf("%s:%s/%s ", name, release, shortID)
			}
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

type cloudwatchEvents []*cloudwatchlogs.FilteredLogEvent

func (e cloudwatchEvents) Len() int           { return len(e) }
func (e cloudwatchEvents) Less(i, j int) bool { return *(e[i].Timestamp) < *(e[j].Timestamp) }
func (e cloudwatchEvents) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
