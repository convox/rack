package monitor

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ActiveState/tail"
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/kinesis"
	"github.com/awslabs/aws-sdk-go/gen/logs"
)

type Logs struct {
	AwsRegion string
	AwsAccess string
	AwsSecret string
	AwsToken  string

	Tick time.Duration
	Logs []string

	CloudwatchGroup  string
	CloudwatchStream string
	Kinesis          string

	lines []Line
	lock  *sync.Mutex

	cloudwatch *logs.Logs
	cwsequence string

	kinesis *kinesis.Kinesis
}

type Line struct {
	Text string
	Time time.Time
}

func (lm *Logs) Monitor() {
	lm.lock = &sync.Mutex{}

	if lm.CloudwatchGroup != "" {
		lm.initializeCloudwatch()
	}

	if lm.Kinesis != "" {
		lm.initializeKinesis()
	}

	for _, log := range lm.Logs {
		go lm.watchLog(log)
	}

	for _ = range time.Tick(lm.Tick) {
		go lm.uploadLogs(500)
	}
}

func (lm *Logs) initializeCloudwatch() {
	creds := aws.Creds(lm.AwsAccess, lm.AwsSecret, lm.AwsToken)

	lm.cloudwatch = logs.New(creds, lm.AwsRegion, nil)

	req := &logs.DescribeLogStreamsRequest{
		Limit:               aws.Integer(1),
		LogGroupName:        aws.String(lm.CloudwatchGroup),
		LogStreamNamePrefix: aws.String(lm.CloudwatchStream),
	}

	res, err := lm.cloudwatch.DescribeLogStreams(req)

	if err == nil && len(res.LogStreams) == 1 {
		if *res.LogStreams[0].LogStreamName == lm.CloudwatchStream {
			lm.cwsequence = *res.LogStreams[0].UploadSequenceToken
		}
	} else {
		stream := &logs.CreateLogStreamRequest{
			LogGroupName:  aws.String(lm.CloudwatchGroup),
			LogStreamName: aws.String(lm.CloudwatchStream),
		}

		err = lm.cloudwatch.CreateLogStream(stream)

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
		}
	}
}

func (lm *Logs) initializeKinesis() {
	creds := aws.Creds(lm.AwsAccess, lm.AwsSecret, lm.AwsToken)

	lm.kinesis = kinesis.New(creds, lm.AwsRegion, nil)
}

func (lm *Logs) addLine(text string, tm time.Time) {
	lm.lock.Lock()
	defer lm.lock.Unlock()

	lm.lines = append(lm.lines, Line{Text: text, Time: tm})
}

func (lm *Logs) watchLog(log string) {
	t, err := tail.TailFile(log, tail.Config{Follow: true, ReOpen: true})

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
	}

	for line := range t.Lines {
		lm.addLine(line.Text, line.Time)
	}
}

func (lm *Logs) uploadLogs(max int) {
	lm.lock.Lock()
	defer lm.lock.Unlock()

	if len(lm.lines) == 0 {
		return
	}

	var lines []Line

	if len(lm.lines) > max {
		lines = lm.lines[0:max]
		lm.lines = lm.lines[max:]
	} else {
		lines = lm.lines
		lm.lines = make([]Line, 0)
	}

	if lm.cloudwatch != nil {
		fmt.Printf("uploading %d lines to cloudwatch\n", len(lines))
		lm.uploadCloudwatch(lines)
	}

	if lm.kinesis != nil {
		fmt.Printf("uploading %d lines to kinesis\n", len(lines))
		lm.uploadKinesis(lines)
	}
}

func (lm *Logs) uploadCloudwatch(lines []Line) {
	stream := lm.CloudwatchStream

	if stream == "" {
		stream = "default"
	}

	events := &logs.PutLogEventsRequest{
		LogEvents:     make([]logs.InputLogEvent, len(lines)),
		LogGroupName:  aws.String(lm.CloudwatchGroup),
		LogStreamName: aws.String(stream),
	}

	if lm.cwsequence != "" {
		events.SequenceToken = aws.String(lm.cwsequence)
	}

	for i, line := range lines {
		events.LogEvents[i].Message = aws.String(line.Text)
		events.LogEvents[i].Timestamp = aws.Long(line.Time.UnixNano() / 1000000)
	}

	res, err := lm.cloudwatch.PutLogEvents(events)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
	}

	if res != nil && res.NextSequenceToken != nil {
		lm.cwsequence = *res.NextSequenceToken
	}
}

func (lm *Logs) uploadKinesis(lines []Line) {
	records := &kinesis.PutRecordsInput{
		Records:    make([]kinesis.PutRecordsRequestEntry, len(lines)),
		StreamName: aws.String(lm.Kinesis),
	}

	for i, line := range lines {
		records.Records[i].Data = []byte(line.Text)
		records.Records[i].PartitionKey = aws.String(string(time.Now().UnixNano()))
	}

	res, err := lm.kinesis.PutRecords(records)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
	}

	for _, r := range res.Records {
		if r.ErrorCode != nil {
			fmt.Printf("error: %s\n", *r.ErrorCode)
		}
	}
}
