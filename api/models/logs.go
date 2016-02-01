package models

import (
	"fmt"
	"os"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/rds"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/ddollar/logger"
)

/*
App logs are written to many streams, one per container
Periodically describe the streams for a group
For new or updating streams launch a goroutine to get and output the events
*/
func subscribeCloudWatchLogs(group string, output chan []byte, quit chan bool) {
	fmt.Printf("subscribeCloudWatchLogs group=%s\n", group)

	horizonTime := time.Now().Add(-2 * time.Minute)
	activeStreams := map[string]bool{}

	for {
		res, err := CloudWatchLogs().DescribeLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
			LogGroupName: aws.String(group),
			OrderBy:      aws.String(cloudwatchlogs.OrderByLastEventTime),
			Descending:   aws.Bool(true),
		})

		if err != nil {
			fmt.Printf("ERROR: %+v\n", err)
			return
		}

		for _, s := range res.LogStreams {
			if activeStreams[*s.LogStreamName] {
				continue
			}

			if s.LastEventTimestamp == nil {
				continue
			}

			sec := *s.LastEventTimestamp / 1000                   // convert ms since epoch to sec
			nsec := (*s.LastEventTimestamp - (sec * 1000)) * 1000 // convert remainder to nsec
			lastEventTime := time.Unix(sec, nsec)

			// fmt.Printf("subscribeCloudWatchLogs horizonTime=%+v lastEventTime=%+v lastEventTimestamp=%d sec=%+v nsec=%+v\n", horizonTime, lastEventTime, *s.LastEventTimestamp, sec, nsec)

			if lastEventTime.After(horizonTime) {
				activeStreams[*s.LogStreamName] = true
				go subscribeCloudWatchLogsStream(group, *s.LogStreamName, horizonTime, output, quit)
			}
		}

		time.Sleep(1000 * time.Millisecond)
	}
}

func subscribeCloudWatchLogsStream(group, stream string, startTime time.Time, output chan []byte, quit chan bool) {
	log := logger.New("at=subscribe-cloudwatch").Start()
	fmt.Printf("subscribeCloudWatchLogsStream group=%s stream=%s startTime=%s\n", group, stream, startTime)

	startTimeMs := startTime.Unix() * 1000 // ms since epoch

	req := cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String(group),
		LogStreamName: aws.String(stream),
	}

	for {
		select {
		case <-quit:
			log.Log("qutting")
			return
		default:
			req.StartTime = &startTimeMs

			res, err := CloudWatchLogs().GetLogEvents(&req)

			if err != nil {
				fmt.Printf("err3 %+v\n", err)
				return
			}

			for _, event := range res.Events {
				output <- []byte(fmt.Sprintf("%s\n", string(*event.Message)))
				startTimeMs = *event.Timestamp + 1
			}

			time.Sleep(1000 * time.Millisecond)
		}
	}
}

func subscribeKinesis(stream string, output chan []byte, quit chan bool) {
	sreq := &kinesis.DescribeStreamInput{
		StreamName: aws.String(stream),
	}
	sres, err := Kinesis().DescribeStream(sreq)

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR(subscribeKinesis): %s\n", err)
		return
	}

	shards := make([]string, len(sres.StreamDescription.Shards))

	for i, s := range sres.StreamDescription.Shards {
		shards[i] = *s.ShardId
	}

	for _, shard := range shards {
		go subscribeKinesisShard(stream, shard, output, quit)
	}
}

func subscribeKinesisShard(stream, shard string, output chan []byte, quit chan bool) {
	log := logger.New("at=subscribe-kinesis").Start()

	ireq := &kinesis.GetShardIteratorInput{
		ShardId:           aws.String(shard),
		ShardIteratorType: aws.String("LATEST"),
		StreamName:        aws.String(stream),
	}

	ires, err := Kinesis().GetShardIterator(ireq)

	if err != nil {
		log.Error(err)
		return
	}

	iter := *ires.ShardIterator

	for {
		select {
		case <-quit:
			log.Log("qutting")
			return
		default:
			greq := &kinesis.GetRecordsInput{
				ShardIterator: aws.String(iter),
			}
			gres, err := Kinesis().GetRecords(greq)

			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR(subscribeKinesisShard): %s\n", err)
				return
			}

			iter = *gres.NextShardIterator

			for _, record := range gres.Records {
				output <- []byte(fmt.Sprintf("%s\n", string(record.Data)))
			}

			time.Sleep(500 * time.Millisecond)
		}
	}
}

func subscribeRDS(prefix, id string, output chan []byte, quit chan bool) {
	// Get latest log file details via pagination tokens
	details := rds.DescribeDBLogFilesDetails{}
	marker := ""
	log := logger.New("at=subscribe-kinesis").Start()

	for {
		params := &rds.DescribeDBLogFilesInput{
			DBInstanceIdentifier: aws.String(id),
			MaxRecords:           aws.Int64(100),
		}

		if marker != "" {
			params.Marker = aws.String(marker)
		}

		res, err := RDS().DescribeDBLogFiles(params)

		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR(subscribeRDS.DescribeDBLogFiles): %s\n", err)
			return
		}

		if res.Marker == nil {
			files := res.DescribeDBLogFiles
			details = *files[len(files)-1]

			break
		}

		marker = *res.Marker
	}

	// Get last 50 log lines
	params := &rds.DownloadDBLogFilePortionInput{
		DBInstanceIdentifier: aws.String(id),
		LogFileName:          aws.String(*details.LogFileName),
		NumberOfLines:        aws.Int64(50),
	}

	res, err := RDS().DownloadDBLogFilePortion(params)

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR(subscribeRDS.DownloadDBLogFilePortion): %s\n", err)
		return
	}

	output <- []byte(fmt.Sprintf("%s: %s\n", prefix, *res.LogFileData))

	params.Marker = aws.String(*res.Marker)

	for {
		select {
		case <-quit:
			log.Log("qutting")
			return
		default:
			res, err := RDS().DownloadDBLogFilePortion(params)

			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR(subscribeRDS.DownloadDBLogFilePortion): %s\n", err)
				return
			}

			if *params.Marker != *res.Marker {
				params.Marker = aws.String(*res.Marker)

				output <- []byte(fmt.Sprintf("%s: %s\n", prefix, *res.LogFileData))
			}

			time.Sleep(1000 * time.Millisecond)
		}
	}
}
