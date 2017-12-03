package kinesis

/*
{
    "Records": [
        {
            "kinesis": {
                "partitionKey": "partitionKey-3",
                "kinesisSchemaVersion": "1.0",
                "data": "SGVsbG8sIHRoaXMgaXMgYSB0ZXN0IDEyMy4=",
                "sequenceNumber": "49545115243490985018280067714973144582180062593244200961"
            },
            "eventSource": "aws:kinesis",
            "eventID": "shardId-000000000000:49545115243490985018280067714973144582180062593244200961",
            "invokeIdentityArn": "arn:aws:iam::059493405231:role/testLEBRole",
            "eventVersion": "1.0",
            "eventName": "aws:kinesis:record",
            "eventSourceARN": "arn:aws:kinesis:us-west-2:35667example:stream/examplestream",
            "awsRegion": "us-west-2"
        }
    ]
}
*/

// Kinesis Event data.  TODO - automatically base64 decode `Data`
type Kinesis struct {
	PartitionKey         string `json:"partitionKey"`
	KinesisSchemaVersion string `json:"kinesisSchemaVersion"`
	Data                 string `json:"data"`
	SequenceNumber       string `json:"sequenceNumber"`
}

// EventRecord event data
type EventRecord struct {
	EventSource       string  `json:"eventSource"`
	EventID           string  `json:"eventID"`
	InvokeIdentityARN string  `json:"invokeIdentityArn"`
	EventVersion      string  `json:"eventVersion"`
	EventName         string  `json:"eventName"`
	EventSourceARN    string  `json:"eventSourceARN"`
	AWSRegion         string  `json:"awsRegion"`
	Kinesis           Kinesis `json:"kinesis"`
}

// Event data
type Event struct {
	Records []EventRecord
}
