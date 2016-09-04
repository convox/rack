package aws_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/api/structs"
	"github.com/stretchr/testify/assert"
)

func TestLogStream(t *testing.T) {
	provider := StubAwsProvider(
		cycleFormationDescribeStacks,
		cycleLogFilterLogEvents1,
		cycleLogFilterLogEvents2,
	)
	defer provider.Close()

	buf := &bytes.Buffer{}

	err := provider.LogStream("httpd", buf, structs.LogStreamOptions{
		Follow: false,
		Filter: "test",
		Since:  time.Unix(1472946223, 0),
	})

	assert.Nil(t, err)
	assert.Equal(t, "2014-03-28T15:36:18-04:00 event1\n2014-03-28T15:36:18-04:00 event2\n2014-03-28T15:36:18-04:00 event3\n2014-03-28T15:36:18-04:00 event4\n2014-03-28T15:36:18-04:00 event5\n", buf.String())
}

var cycleLogFilterLogEvents1 = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "Logs_20140328.FilterLogEvents",
		Body:       `{"endTime":1472946823000,"filterPattern":"test","interleaved":true,"limit":10000,"logGroupName":"convox-httpd-LogGroup-L4V203L35WRM","startTime":1472946223000}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"events": [
				{
					"ingestionTime": 1396035394997,
					"timestamp": 1396035378988,
					"message": "event2",
					"logStreamName": "stream1",
					"eventId": "31132629274945519779805322857203735586714454643391594505"
				},
				{
					"ingestionTime": 1396035394997,
					"timestamp": 1396035378988,
					"message": "event3",
					"logStreamName": "stream2",
					"eventId": "31132629274945519779805322857203735586814454643391594505"
				},
				{
					"ingestionTime": 1396035394997,
					"timestamp": 1396035378989,
					"message": "event4",
					"logStreamName": "stream3",
					"eventId": "31132629274945519779805322857203735586824454643391594505"
				}
			],
			"searchedLogStreams": [
				{
					"searchedCompletely": false, 
					"logStreamName": "stream1"
				}, 
				{
					"searchedCompletely": false,      
					"logStreamName": "stream2"
				},
				{
					"searchedCompletely": true,
					"logStreamName": "stream3"
				}
			],
			"nextToken": "ZNUEPl7FcQuXbIH4Swk9D9eFu2XBg-ijZIZlvzz4ea9zZRjw-MMtQtvcoMdmq4T29K7Q6Y1e_KvyfpcT_f_tUw"
		}`,
	},
}

var cycleLogFilterLogEvents2 = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "Logs_20140328.FilterLogEvents",
		Body:       `{"endTime":1472946823000,"filterPattern":"test","interleaved":true,"limit":10000,"logGroupName":"convox-httpd-LogGroup-L4V203L35WRM","startTime":1472946223000,"nextToken":"ZNUEPl7FcQuXbIH4Swk9D9eFu2XBg-ijZIZlvzz4ea9zZRjw-MMtQtvcoMdmq4T29K7Q6Y1e_KvyfpcT_f_tUw"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"events": [
				{
					"ingestionTime": 1396035394997,
					"timestamp": 1396035378968,
					"message": "event1",
					"logStreamName": "stream1",
					"eventId": "31132629274945519779805322857203735586714454643391594505"
				},
				{
					"ingestionTime": 1396035394997,
					"timestamp": 1396035378998,
					"message": "event5",
					"logStreamName": "stream2",
					"eventId": "31132629274945519779805322857203735586814454643391594505"
				}
			],
			"searchedLogStreams": [
				{
					"searchedCompletely": true, 
					"logStreamName": "stream1"
				}, 
				{
					"searchedCompletely": true,      
					"logStreamName": "stream2"
				}
			]
		}`,
	},
}
