package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
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

	var message Message

	err = json.Unmarshal(data, &message)

	if err != nil {
		die(err)
	}

	fmt.Printf("message %+v\n", message)

	for _, record := range message.Records {
		var req formation.Request

		fmt.Printf("record.Sns.Message %+v\n", record.Sns.Message)

		err = json.Unmarshal([]byte(record.Sns.Message), &req)

		fmt.Printf("req %+v\n", req)

		physical := ""

		switch req.ResourceType {
		case "Custom::ECSService":
			physical, err = formation.HandleECSService(req)
		case "Custom::ECSTaskDefinition":
			physical, err = formation.HandleECSTaskDefinition(req)
		}

		res := formation.Response{
			RequestId:          req.RequestId,
			StackId:            req.StackId,
			LogicalResourceId:  req.LogicalResourceId,
			PhysicalResourceId: physical,
			Status:             "SUCCESS",
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			res.Reason = err.Error()
			res.Status = "FAILED"
		}

		fmt.Printf("res %+v\n", res)

		err = sendResponse(req.ResponseURL, res)

		if err != nil {
			die(err)
		}
	}
}

func sendResponse(url string, r formation.Response) error {
	data, err := json.Marshal(r)

	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", "", bytes.NewBuffer(data))

	if err != nil {
		return err
	}

	fmt.Printf("url %+v\n", url)

	// golang's http methods munge the %3A in amazon urls so we build it manually using Opaque
	parts := strings.SplitN(url, "/", 4)
	req.URL.Scheme = parts[0][0 : len(parts[0])-1]
	req.URL.Host = parts[2]
	req.URL.Opaque = fmt.Sprintf("//%s/%s", parts[2], parts[3])

	client := &http.Client{}

	res, err := client.Do(req)

	if err != nil {
		return err
	}

	defer res.Body.Close()

	rb, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return err
	}

	fmt.Printf("response: %s\n", string(rb))

	return nil
}
