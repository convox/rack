package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	syslog "github.com/RackSec/srslog"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/jasonmoo/lambda_proc"
	"github.com/mweagle/Sparta/aws/cloudwatchlogs"
)

func main() {
	lambda_proc.Run(func(context *lambda_proc.Context, eventJSON json.RawMessage) (interface{}, error) {
		stackName := getStackName(context.FunctionName)
		syslogUrl, err := readOrDescribeURL(stackName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "readOrDescribeURL err=%s\n", err)
			return nil, err
		}

		u, err := url.Parse(syslogUrl)
		if err != nil {
			fmt.Fprintf(os.Stderr, "url.Parse url=%s err=%s\n", syslogUrl, err)
			return nil, err
		}

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

		w, err := syslog.Dial(u.Scheme, u.Host, syslog.LOG_INFO, "convox/syslog")
		if err != nil {
			fmt.Fprintf(os.Stderr, "syslog.Dial scheme=%s host=%s err=%s\n", u.Scheme, u.Host, err)
			return nil, err
		}
		defer w.Close()

		w.SetFormatter(contentFormatter(d.LogGroup))

		logs, errs := 0, 0
		for _, e := range d.LogEvents {
			err := w.Info(fmt.Sprintf("%s %d %s", d.LogStream, e.Timestamp, e.Message))
			if err != nil {
				errs += 1
			} else {
				logs += 1
			}
		}

		return fmt.Sprintf("LogGroup=%s LogStream=%s MessageType=%s NumLogEvents=%d logs=%d errs=%d", d.LogGroup, d.LogStream, d.MessageType, len(d.LogEvents), logs, errs), nil
	})
}

func contentFormatter(group string) syslog.Formatter {
	return func(p syslog.Priority, hostname, tag, content string) string {
		timestamp := time.Now()
		hostname = group
		program := "convox/syslog"
		tag = "unknown"

		parts := strings.SplitN(content, " ", 3)

		if len(parts) == 3 {
			pp := strings.SplitN(parts[0], "/", 3)
			if len(pp) == 3 {
				program = fmt.Sprintf("%s/%s", pp[0], pp[1])
				tp := strings.Split(pp[2], "-")
				tag = tp[len(tp)-1]
			}

			i, err := strconv.ParseInt(parts[1], 10, 64)
			if err == nil {
				sec := i / 1000
				nsec := i - (sec * 1000)
				timestamp = time.Unix(sec, nsec).UTC()
			}

			content = parts[2]
		}

		return fmt.Sprintf("<%d>%d %s %s %s %s - - %s\n", 22, 1, timestamp.Format(time.RFC3339), hostname, program, tag, content)
	}
}

func getStackName(functionName string) string {
	i := strings.Index(functionName, "-Function")
	if i == -1 {
		return functionName
	} else {
		return functionName[:i]
	}
}

func readOrDescribeURL(name string) (string, error) {
	data, err := ioutil.ReadFile("/tmp/url")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ioutil.ReadFile err=%s\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "ioutil.ReadFile url=%s\n", string(data))
		return string(data), nil
	}

	cf := cloudformation.New(session.New(&aws.Config{}))

	resp, err := cf.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(name),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "cf.DescribeStacks err=%s\n", err)
		return "", err
	}

	fmt.Fprintf(os.Stderr, "cf.DescribeStacks resp=%+v\n", resp)

	if len(resp.Stacks) == 1 {
		for _, p := range resp.Stacks[0].Parameters {
			if *p.ParameterKey == "Url" {
				url := *p.ParameterValue

				err := ioutil.WriteFile("/tmp/url", []byte(url), 0644)
				if err != nil {
					fmt.Fprintf(os.Stderr, "ioutil.WriteFile url=%s err=%s\n", url, err)
				} else {
					fmt.Fprintf(os.Stderr, "ioutil.WriteFile url=%s\n", url)
				}

				return url, nil
			}
		}
	}

	return "", fmt.Errorf("Could not find stack %s Url Parameter", name)
}
