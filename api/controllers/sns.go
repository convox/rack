package controllers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/convox/logger"
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

	if _, err := http.Get(payload["SubscribeURL"]); err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}

	RenderSuccess(w)
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

	log.Logf("proxied=true status=%s", resp.Status)
	w.Write([]byte("ok"))
}
