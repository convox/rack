package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/convox/kernel/formation"
)

type Message struct {
	Records []Record
}

type Record struct {
	EventSource          string
	EventVersion         string
	EventSubscriptionArn string
	Sns                  Sns
}

type Sns struct {
	Type             string
	MessageId        string
	TopicArn         string
	Subject          string
	Message          string
	Timestamp        time.Time
	SignatureVersion string
	Signature        string
	SigningCertURL   string
	UnsubscribeURL   string
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "error: %s\n", err)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		die(fmt.Errorf("must specify event as argument"))
	}

	data := []byte(os.Args[1])

	var message Message

	err := json.Unmarshal(data, &message)

	if err != nil {
		die(err)
	}

	fmt.Printf("message = %+v\n", message)

	for _, record := range message.Records {
		var req formation.Request

		err := json.Unmarshal([]byte(record.Sns.Message), &req)

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return
		}

		fmt.Printf("req = %+v\n", req)

		err = formation.HandleRequest(req)

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return
		}
	}
}
