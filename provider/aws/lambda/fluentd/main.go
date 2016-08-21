package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/jasonmoo/lambda_proc"
	"github.com/mweagle/Sparta/aws/cloudwatchlogs"
)

type fluentURL struct {
	Host string
	Port int
}

func main() {
	lambda_proc.Run(func(context *lambda_proc.Context, eventJSON json.RawMessage) (interface{}, error) {
		fluentURL, err := getfluentURL(context.FunctionName)
		fmt.Fprintf(os.Stderr, "fluentd connection config=%s %d\n", fluentURL.Host, fluentURL.Port)

		logger, err := fluent.New(fluent.Config{FluentPort: fluentURL.Port, FluentHost: fluentURL.Host})

		if err != nil {
			fmt.Fprintf(os.Stderr, "fluentd connection error=%s\n", err)
			return nil, err
		}
		defer logger.Close()

		var event cloudwatchlogs.Event
		err = json.Unmarshal([]byte(eventJSON), &event)
		if err != nil {
			fmt.Fprintf(os.Stderr, "json.Unmarshal err=%s\n", err)
			return nil, err
		}

		d, err := event.AWSLogs.DecodedData()
		if err != nil {
			fmt.Fprintf(os.Stderr, "AWSLogs.DecodedData err=%s\n", err)
			return nil, err
		}

		logs, errs := 0, 0
		for _, e := range d.LogEvents {

			event, err := decodeLogLine(e.Message)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error decoding log line err=%s\n", err)
				continue
			}

			tag := d.LogGroup

			err = logger.Post(tag, event)
			if err != nil {
				fmt.Fprintf(os.Stderr, "FluentD Post: %s\n", err)
				return nil, err
			}
		}

		return fmt.Sprintf("LogGroup=%s LogStream=%s MessageType=%s NumLogEvents=%d logs=%d errs=%d", d.LogGroup, d.LogStream, d.MessageType, len(d.LogEvents), logs, errs), nil
	})
}

func decodeLogLine(msg string) (map[string]interface{}, error) {
	s := strings.Split(msg, " ")
	log_group := s[0]
	event := strings.Join(s[1:], " ")
	s = strings.Split(log_group, ":")
	container_name, convox_metadata := s[0], s[1]
	s = strings.Split(convox_metadata, "/")
	release, container_id := s[0], s[1]

	var decoded_json map[string]interface{}
	err := json.Unmarshal([]byte(event), &decoded_json)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding json: %s\n", err)
		return nil, err
	}

	decoded_json["convox_release"] = release
	decoded_json["container_name"] = container_name
	decoded_json["ecs_container_id"] = container_id

	return decoded_json, nil
}

func parseURL(cf_url string) (string, int) {
	parsed_url, err := url.Parse(cf_url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "url.Parse url=%s\n", cf_url)
	}

	fluentHost, fluentPort_string, _ := net.SplitHostPort(parsed_url.Host)
	fluentPort, err := strconv.Atoi(fluentPort_string)
	if err != nil {
		fmt.Fprintf(os.Stderr, "strconv.ParseInt - Failed parsing int out of port string=%s\n", fluentPort_string)
	}

	return fluentHost, fluentPort
}

func getfluentURL(name string) (fluentURL, error) {
	data, err := ioutil.ReadFile("/tmp/url")
	if err != nil {
		fmt.Fprintf(os.Stderr, "URL Cache empty=%s\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "Found cached url=%s\n", string(data))
		fluentHost, fluentPort := parseURL(string(data))
		return fluentURL{Host: fluentHost, Port: fluentPort}, nil
	}

	cf := cloudformation.New(session.New(&aws.Config{}))

	resp, err := cf.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(name),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "cf.DescribeStacks err=%s\n", err)
		return fluentURL{}, err
	}

	if len(resp.Stacks) == 1 {
		for _, p := range resp.Stacks[0].Parameters {
			if *p.ParameterKey == "Url" {
				cf_url := *p.ParameterValue

				fluentHost, fluentPort := parseURL(cf_url)

				err = ioutil.WriteFile("/tmp/url", []byte(cf_url), 0644)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error writing URL cache for url=%s err=%s\n", cf_url, err)
				} else {
					fmt.Fprintf(os.Stderr, "Wrote URL Cache w/ url=%s\n", cf_url)
				}

				return fluentURL{Host: fluentHost, Port: fluentPort}, nil
			}
		}
	}

	return fluentURL{}, fmt.Errorf("Could not find stack %s Url Parameter", name)
}
