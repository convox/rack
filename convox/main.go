package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/convox/cli/stdcli"
)

func main() {
	app := stdcli.New()
	app.Usage = "command-line application management"
	app.Run(os.Args)
}

func convoxRequest(method, path string) ([]byte, error) {
	host, password, err := currentLogin()

	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s%s", host, path), nil)

	if err != nil {
		return nil, err
	}

	req.SetBasicAuth("convox", string(password))

	res, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	return data, nil
}
