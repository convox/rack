package controllers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/sns"
	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/rack/api/models"
)

func SNSConfirm(w http.ResponseWriter, r *http.Request) {
	log := logger.New("ns=kernel").At("SNSConfirm")

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}

	var payload map[string]string
	err = json.Unmarshal(body, &payload)

	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}

	params := &sns.ConfirmSubscriptionInput{
		Token:    aws.String(payload["Token"]),
		TopicArn: aws.String(payload["TopicArn"]),
	}
	resp, err := models.SNS().ConfirmSubscription(params)

	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}

	log.Log("confirmed=true subscriptionArn=%q", *resp.SubscriptionArn)
	w.Write([]byte("ok"))
}

func SNSProxy(w http.ResponseWriter, r *http.Request) {
	log := logger.New("ns=kernel").At("SNSProxy")

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}

	var payload map[string]string
	err = json.Unmarshal(body, &payload)

	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}

	url := r.FormValue("endpoint")
	resp, err := http.Post(url, "application/json", strings.NewReader(payload["Message"]))
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}

	log.Log("proxied=true status=%s", resp.Status)
	w.Write([]byte("ok"))
}
