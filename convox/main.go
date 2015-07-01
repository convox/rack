package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/convox/cli/stdcli"
)

var Version = "0.8"

func main() {
	app := stdcli.New()
	app.Version = Version
	app.Usage = "command-line application management"
	app.Run(os.Args)
}

func ConvoxGet(path string) ([]byte, error) {
	client := convoxClient()

	req, err := convoxRequest("GET", path, nil)

	if err != nil {
		stdcli.Error(err)
		return nil, err
	}

	res, err := client.Do(req)

	if err != nil {
		stdcli.Error(err)
		return nil, err
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		stdcli.Error(err)
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf(strings.TrimSpace(string(data)))
	}

	return data, nil
}

func ConvoxPost(path string, body string) ([]byte, error) {
	client := convoxClient()

	req, err := convoxRequest("POST", path, strings.NewReader(body))

	if err != nil {
		stdcli.Error(err)
		return nil, err
	}

	res, err := client.Do(req)

	if err != nil {
		stdcli.Error(err)
		return nil, err
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		stdcli.Error(err)
		return nil, err
	}

	return data, nil
}

func ConvoxPostForm(path string, form url.Values) ([]byte, error) {
	client := convoxClient()

	req, err := convoxRequest("POST", path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if err != nil {
		return nil, err
	}

	// Use RoundTrip to avoid following the redirect without the Auth header
	res, err := client.Transport.RoundTrip(req)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	if res.StatusCode == 200 || res.StatusCode == 302 {
		return data, nil
	}

	return nil, fmt.Errorf(strings.TrimSpace(string(data)))
}

func ConvoxDelete(path string) ([]byte, error) {
	client := convoxClient()

	req, err := convoxRequest("DELETE", path, nil)

	if err != nil {
		stdcli.Error(err)
		return nil, err
	}

	res, err := client.Do(req)

	if err != nil {
		stdcli.Error(err)
		return nil, err
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		stdcli.Error(err)
		return nil, err
	}

	return data, nil
}

func convoxClient() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	return client
}

func convoxRequest(method, path string, body io.Reader) (*http.Request, error) {
	host, password, err := currentLogin()

	if err != nil {
		stdcli.Error(err)
		return nil, err
	}

	req, err := http.NewRequest(method, fmt.Sprintf("https://%s%s", host, path), body)

	if err != nil {
		stdcli.Error(err)
		return nil, err
	}

	req.SetBasicAuth("convox", string(password))
	req.Header.Add("Content-Type", "application/json")

	return req, nil
}
