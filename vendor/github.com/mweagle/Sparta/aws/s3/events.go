package s3

// EventOwnerIdentity event data
type EventOwnerIdentity struct {
	PrincpalID string `json:"principalId"`
}

// Bucket event data
type Bucket struct {
	Name          string             `json:"name"`
	Arn           string             `json:"arn"`
	OwnerIdentity EventOwnerIdentity `json:"ownerIdentity"`
}

// Object event data
type Object struct {
	Key       string `json:"key"`
	Sequencer string `json:"sequencer"`
}

// S3 event information
type S3 struct {
	SchemaVersion   string `json:"s3SchemaVersion"`
	ConfigurationID string `json:"configurationId"`
	Bucket          Bucket `json:"bucket"`
	Object          Object `json:"object"`
}

// EventRecord event data
type EventRecord struct {
	Region       string `json:"awsRegion"`
	EventName    string `json:"eventName"`
	EventTime    string `json:"eventTime"`
	EventSource  string `json:"eventSource"`
	EventVersion string `json:"eventVersion"`
	S3           S3     `json:"s3"`
}

// Event data
type Event struct {
	Records []EventRecord
}
