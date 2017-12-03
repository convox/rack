package sns

/*
{
  "Records": [
    {
      "EventSource": "aws:sns",
      "EventVersion": "1.0",
      "EventSubscriptionArn": "arn:aws:sns:us-west-2:123412341234:topicName:d6f6d83d-ee9e-457c-b556-25dc693b561e",
      "Sns": {
        "Type": "Notification",
        "MessageId": "03c9bf63-0696-522d-abc3-f1bf491fc599",
        "TopicArn": "arn:aws:sns:us-west-2:123412341234:topicName",
        "Subject": "asdfasdf",
        "Message": "asdfasdfasdf",
        "Timestamp": "2015-12-05T02:34:49.729Z",
        "SignatureVersion": "1",
        "Signature": "XXXXXXXX==",
        "SigningCertUrl": "https://sns.us-west-2.amazonaws.com/SimpleNotificationService-bb750dd426d95ee9390147a5624348ee.pem",
        "UnsubscribeUrl": "https://sns.us-west-2.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=arn:aws:sns:us-west-2:123412341234:topicName:d6f6d83d-ee9e-457c-b556-25dc693b561e",
        "MessageAttributes": {
          "AWS.SNS.MOBILE.MPNS.Type": {
            "Type": "String",
            "Value": "token"
          },
          "AWS.SNS.MOBILE.MPNS.NotificationClass": {
            "Type": "String",
            "Value": "realtime"
          },
          "AWS.SNS.MOBILE.WNS.Type": {
            "Type": "String",
            "Value": "wns/badge"
          }
        }
      }
    }
  ]
}
*/

// MessageAttributeValues type/value data for the parent key
type MessageAttributeValues struct {
	Type  string `json:"Type"`
	Value string `json:"Value"`
}

// Sns event information
type Sns struct {
	Type              string `json:"Type"`
	MessageID         string `json:"MessageId"`
	TopicArn          string `json:"TopicArn"`
	Subject           string `json:"Subject"`
	Message           string `json:"Message"`
	Timestamp         string `json:"Timestamp"`
	SignatureVersion  string `json:"SignatureVersion"`
	Signature         string `json:"Signature"`
	SigningCertURL    string `json:"SigningCertUrl"`
	UnsubscribeURL    string `json:"UnsubscribeUrl"`
	MessageAttributes map[string]MessageAttributeValues
}

// EventRecord event data
type EventRecord struct {
	EventSource          string `json:"EventSource"`
	EventVersion         string `json:"EventVersion"`
	EventSubscriptionArn string `json:"EventSubscriptionArn"`
	Sns                  Sns    `json:"Sns"`
}

// Event data
type Event struct {
	Records []EventRecord
}
