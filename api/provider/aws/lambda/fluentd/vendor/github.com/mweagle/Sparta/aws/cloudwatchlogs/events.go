package cloudwatchlogs

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
)

// LogEvent represents an entry to process
type LogEvent struct {
	ID        string `json:"id"`
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}

// CloudWatchLogEvent represents the base64 decoded, gunzip'd
// data
type CloudWatchLogEvent struct {
	MessageType         string     `json:"messageType"`
	Owner               string     `json:"owner"`
	LogGroup            string     `json:"logGroup"`
	LogStream           string     `json:"logStream"`
	SubscriptionFilters []string   `json:"subscriptionFilters"`
	LogEvents           []LogEvent `json:"logEvents"`
}

// AWSLogs is the primary key scoping CloudWatchLogs
// data in the lambda event
type AWSLogs struct {
	// Data is the raw, gzip'd, Base64 encoded payload.  Use
	// DecodedData() to access the log entries
	Data string `json:"data"`
}

// DecodedData returns the Base64 decoded, gunzip'd decoded data per
// http://docs.aws.amazon.com/AmazonCloudWatch/latest/DeveloperGuide/Subscriptions.html
// You should expect to see a response with an array of Amazon Kinesis.
// The Data attribute in the Amazon Kinesis record is Base64 encoded and compressed
// with the gzip format. You can examine the raw data from the command line
// using the following Unix commands:
// echo -n "<Content of Data>" | base64 -d | zcat
func (awsLogs *AWSLogs) DecodedData() (*CloudWatchLogEvent, error) {
	data, err := base64.StdEncoding.DecodeString(awsLogs.Data)
	if err != nil {
		return nil, err
	}
	in := bytes.NewReader(data)
	gzip, err := gzip.NewReader(in)
	if nil != err {
		return nil, err
	}
	defer gzip.Close()
	allData, err := ioutil.ReadAll(gzip)
	if err != nil {
		return nil, err
	}

	var cloudWatchEvent CloudWatchLogEvent
	err = json.Unmarshal(allData, &cloudWatchEvent)
	if nil != err {
		return nil, err
	}
	return &cloudWatchEvent, nil
}

// Event data
type Event struct {
	AWSLogs AWSLogs `json:"awslogs"`
}
