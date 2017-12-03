package dynamodb

import awsDynamoDB "github.com/aws/aws-sdk-go/service/dynamodb"

/*
{
  "Records": [
    {
      "eventID": "329ef0da9a7d792c7fa83fcdf9194d24",
      "eventName": "INSERT",
      "eventVersion": "1.0",
      "eventSource": "aws:dynamodb",
      "awsRegion": "us-west-2",
      "dynamodb": {
        "Keys": {
        	// Name->Attribute Map
        },
        "NewImage":
          // Name->Attribute Map
        },
        "SequenceNumber": "1645779500000000000731163099",
        "SizeBytes": 136,
        "StreamViewType": "NEW_AND_OLD_IMAGES"
      },
      "eventSourceARN": "arn:aws:dynamodb:us-west-2:000000000000:table/myTable/stream/2015-12-05T16:28:11.869"
    }
  ]
}
*/

// DynamoDB event information
type DynamoDB struct {
	Keys     map[string]awsDynamoDB.AttributeValue
	NewImage map[string]awsDynamoDB.AttributeValue
	OldImage map[string]awsDynamoDB.AttributeValue
}

// EventRecord event data
type EventRecord struct {
	EventID        string   `json:"eventID"`
	EventName      string   `json:"eventName"`
	EventVersion   string   `json:"eventVersion"`
	EventSource    string   `json:"eventSource"`
	EventSourceARN string   `json:"eventSourceARN"`
	AWSRegion      string   `json:"awsRegion"`
	DynamoDB       DynamoDB `json:"dynamodb"`
}

// Event data
type Event struct {
	Records []EventRecord
}
