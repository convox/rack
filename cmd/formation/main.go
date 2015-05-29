package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	data, err := ioutil.ReadAll(os.Stdin)

	fmt.Printf("string(data) %+v\n", string(data))

	if err != nil {
		die(err)
	}

	var req formation.Request

	err = json.Unmarshal(data, &req)

	if err != nil {
		die(err)
	}

	err = formation.HandleRequest(req)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return
	}
}
