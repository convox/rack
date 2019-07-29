package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	syslog "github.com/RackSec/srslog"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/mweagle/Sparta/aws/cloudwatchlogs"
)

func Handler(ctx context.Context, event cloudwatchlogs.Event) error {
	d, err := event.AWSLogs.DecodedData()
	if err != nil {
		return err
	}

	u, err := url.Parse(os.Getenv("SYSLOG_URL"))
	if err != nil {
		return err
	}

	w, err := syslog.Dial(u.Scheme, u.Host, syslog.LOG_INFO, "convox/syslog")
	if err != nil {
		return err
	}
	defer w.Close()

	w.SetFormatter(contentFormatter(d.LogGroup))

	var failures, successes int

	for _, le := range d.LogEvents {
		if err := w.Info(fmt.Sprintf("%s %d %s", d.LogStream, le.Timestamp, le.Message)); err != nil {
			failures++
		} else {
			successes++
		}
	}

	fmt.Printf("group=%s stream=%s type=%s events=%d success=%d failure=%d\n", d.LogGroup, d.LogStream, d.MessageType, len(d.LogEvents), successes, failures)

	return nil
}

func contentFormatter(group string) syslog.Formatter {
	return func(p syslog.Priority, hostname, tag, content string) string {
		timestamp := time.Now()
		service := "convox/syslog"
		container := "unknown"
		message := content

		if parts := strings.SplitN(content, " ", 3); len(parts) == 3 {
			if pp := strings.SplitN(parts[0], "/", 3); len(pp) == 3 {
				service = fmt.Sprintf("%s/%s", pp[0], pp[1])
				cp := strings.Split(pp[2], "-")
				container = cp[len(cp)-1]
			}

			if i, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				sec := i / 1000
				nsec := i - (sec * 1000)
				timestamp = time.Unix(sec, nsec).UTC()
			}

			message = parts[2]
		}

		line := os.Getenv("SYSLOG_FORMAT")

		line = strings.ReplaceAll(line, "{DATE}", timestamp.Format(time.RFC3339))
		line = strings.ReplaceAll(line, "{GROUP}", group)
		line = strings.ReplaceAll(line, "{SERVICE}", service)
		line = strings.ReplaceAll(line, "{CONTAINER}", container)
		line = strings.ReplaceAll(line, "{MESSAGE}", message)

		return line + "\n"
	}
}

func main() {
	lambda.Start(Handler)
}
