package controllers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/sns"
	"github.com/convox/rack/api/models"
)

func SNSHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Println("type: ", req.Header.Get("X-Amz-Sns-Message-Type"))
	fmt.Println("arn: ", req.Header.Get("X-Amz-Sns-Topic-Arn"))

	defer req.Body.Close()
	body, err := ioutil.ReadAll(req.Body)

	if err != nil {
		fmt.Println("err1", err)
		http.Error(w, "Fuck", 500)
		return
	}

	var payload map[string]string
	err = json.Unmarshal(body, &payload)

	if err != nil {
		fmt.Println("err2", err)
		http.Error(w, "Fuck", 500)
		return
	}

	if payload["Type"] == "Notification" {
		fmt.Println(payload["Message"])
		w.Write([]byte(""))
		return
	}

	params := &sns.ConfirmSubscriptionInput{
		Token:    aws.String(payload["Token"]),
		TopicArn: aws.String(payload["TopicArn"]),
	}
	fmt.Println("params", params)
	resp, err := models.SNS().ConfirmSubscription(params)

	if err != nil {
		fmt.Println("err3", err)
		http.Error(w, "Fuck", 500)
		return
	}

	fmt.Printf("%+v\n", resp)

	w.Write([]byte(""))
}
