package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/convox/rack/pkg/structs"
	yaml "gopkg.in/yaml.v2"
)

func AwsCredentialsLoad() error {
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		if err := exec.Command("which", "aws").Run(); err != nil {
			return fmt.Errorf("unable to find aws executable in path")
		}

		data, err := awscli("iam", "get-account-summary")
		if err != nil {
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			return fmt.Errorf("aws cli error: %s", lines[len(lines)-1])
		}

		env, err := setupCredentialsStatic()
		if err != nil {
			return err
		}

		if env["AWS_ACCESS_KEY_ID"] == "" {
			env, err = setupCredentialsRole()
			if err != nil {
				return err
			}
		}

		if env["AWS_ACCESS_KEY_ID"] == "" {
			return fmt.Errorf("unable to load credentials from aws cli")
		}

		for k, v := range env {
			os.Setenv(k, v)
		}
	}

	if os.Getenv("AWS_REGION") == "" {
		os.Setenv("AWS_REGION", "us-east-1")
	}

	return nil
}

func AwsErrorCode(err error) string {
	if ae, ok := err.(awserr.Error); ok {
		return ae.Code()
	}

	return ""
}

func CloudformationInstall(cf *cloudformation.CloudFormation, name, template string, params, tags map[string]string, cb func(int, int)) error {
	res, err := http.Get(template)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var t struct {
		Resources map[string]interface{} `json:"Resources" yaml:"Resources"`
	}

	switch filepath.Ext(template) {
	case ".json":
		if err := json.Unmarshal(data, &t); err != nil {
			return err
		}
	case ".yml", ".yaml":
		if err := yaml.Unmarshal(data, &t); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown template extension: %s", filepath.Ext(template))
	}

	total := len(t.Resources)

	cb(0, total)

	token, err := RandomString(20)
	if err != nil {
		return err
	}

	req := &cloudformation.CreateStackInput{
		Capabilities:       []*string{aws.String("CAPABILITY_IAM")},
		ClientRequestToken: aws.String(token),
		Parameters:         []*cloudformation.Parameter{},
		StackName:          aws.String(name),
		Tags:               []*cloudformation.Tag{},
		TemplateURL:        aws.String(template),
	}

	for k, v := range params {
		req.Parameters = append(req.Parameters, &cloudformation.Parameter{
			ParameterKey:   aws.String(k),
			ParameterValue: aws.String(v),
		})
	}

	for k, v := range tags {
		req.Tags = append(req.Tags, &cloudformation.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	if _, err := cf.CreateStack(req); err != nil {
		return err
	}

	for {
		time.Sleep(1 * time.Second)

		res, err := cf.DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: aws.String(name),
		})
		if err != nil {
			return err
		}
		if len(res.Stacks) != 1 {
			return fmt.Errorf("could not describe stack: %s", name)
		}

		s := res.Stacks[0]

		switch *s.StackStatus {
		case "CREATE_FAILED", "DELETE_COMPLETE", "DELETE_FAILED", "DELETE_IN_PROGRESS", "ROLLBACK_COMPLETE", "ROLLBACK_FAILED", "ROLLBACK_IN_PROGRESS":
			return fmt.Errorf("installation failed")
		case "CREATE_COMPLETE":
			return nil
		}

		rres, err := cf.DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
			StackName: aws.String(name),
		})
		if err != nil {
			return err
		}

		current := 0

		for _, r := range rres.StackResources {
			if *r.ResourceStatus == "CREATE_COMPLETE" {
				current += 1
			}
		}

		cb(current, total)
	}

	return nil
}

func CloudWatchLogsSubscribe(ctx context.Context, cf *cloudwatchlogs.CloudWatchLogs, group, stream string, opts structs.LogsOptions) (io.ReadCloser, error) {
	r, w := io.Pipe()

	go CloudWatchLogsStream(ctx, cf, w, group, stream, opts)

	return r, nil
}

func CloudWatchLogsStream(ctx context.Context, cf *cloudwatchlogs.CloudWatchLogs, w io.WriteCloser, group, stream string, opts structs.LogsOptions) error {
	defer w.Close()

	req := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: aws.String(group),
	}

	if stream != "" {
		req.LogStreamNames = []*string{aws.String(stream)}
	} else {
		req.Interleaved = aws.Bool(true)
	}

	if opts.Filter != nil {
		req.FilterPattern = aws.String(*opts.Filter)
	}

	var start int64

	if opts.Since != nil {
		start = time.Now().UTC().Add((*opts.Since)*-1).UnixNano() / int64(time.Millisecond)
		req.StartTime = aws.Int64(start)
	}

	var seen = map[string]bool{}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// check for closed writer
			if _, err := w.Write([]byte{}); err != nil {
				return err
			}

			res, err := cf.FilterLogEvents(req)
			if err != nil {
				switch AwsErrorCode(err) {
				case "ThrottlingException", "ResourceNotFoundException":
					time.Sleep(1 * time.Second)
					continue
				default:
					return err
				}
			}

			es := []*cloudwatchlogs.FilteredLogEvent{}

			for _, e := range res.Events {
				if !seen[*e.EventId] {
					es = append(es, e)
				}
			}

			seen = map[string]bool{}

			for _, e := range res.Events {
				seen[*e.EventId] = true
			}

			sleep := time.Duration(0)

			if len(es) == 0 {
				sleep = time.Duration(200 * time.Millisecond)
			}

			latest, err := writeLogEvents(w, es, opts)
			if err != nil {
				return err
			}

			if latest > start {
				start = latest //+ 1
			}

			if res.NextToken != nil {
				req.NextToken = res.NextToken
				req.StartTime = nil
				continue
			}

			req.NextToken = nil

			if !DefaultBool(opts.Follow, true) {
				return nil
			}

			if start > 0 {
				req.StartTime = aws.Int64(start)
				time.Sleep(sleep)
			}
		}
	}
}

func awscli(args ...string) ([]byte, error) {
	return exec.Command("aws", args...).CombinedOutput()
}

func setupCredentialsStatic() (map[string]string, error) {
	rb, err := awscli("configure", "get", "region")
	if err != nil {
		return map[string]string{}, nil
	}

	ab, err := awscli("configure", "get", "aws_access_key_id")
	if err != nil {
		return map[string]string{}, nil
	}

	sb, err := awscli("configure", "get", "aws_secret_access_key")
	if err != nil {
		return map[string]string{}, nil
	}

	env := map[string]string{
		"AWS_REGION":            strings.TrimSpace(string(rb)),
		"AWS_ACCESS_KEY_ID":     strings.TrimSpace(string(ab)),
		"AWS_SECRET_ACCESS_KEY": strings.TrimSpace(string(sb)),
	}

	return env, nil
}

func setupCredentialsRole() (map[string]string, error) {
	rb, err := awscli("configure", "get", "role_arn")
	if err != nil {
		return nil, err
	}

	role := strings.TrimSpace(string(rb))

	if role == "" {
		return map[string]string{}, nil
	}

	data, err := awscli("sts", "assume-role", "--role-arn", role, "--role-session-name", "convox-cli")
	if err != nil {
		return nil, err
	}

	var creds struct {
		Credentials struct {
			AccessKeyID     string `json:"AccessKeyId"`
			SecretAccessKey string
			SessionToken    string
		}
	}

	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}

	rgb, err := awscli("configure", "get", "region")
	if err != nil {
		return map[string]string{}, nil
	}

	env := map[string]string{
		"AWS_REGION":            strings.TrimSpace(string(rgb)),
		"AWS_ACCESS_KEY_ID":     creds.Credentials.AccessKeyID,
		"AWS_SECRET_ACCESS_KEY": creds.Credentials.SecretAccessKey,
		"AWS_SESSION_TOKEN":     creds.Credentials.SessionToken,
	}

	return env, nil
}

func writeLogEvents(w io.Writer, events []*cloudwatchlogs.FilteredLogEvent, opts structs.LogsOptions) (int64, error) {
	if len(events) == 0 {
		return 0, nil
	}

	// sort.Slice(events, func(i, j int) bool { return *events[i].Timestamp < *events[j].Timestamp })

	latest := int64(0)

	for _, e := range events {
		if *e.Timestamp > latest {
			latest = *e.Timestamp
		}

		prefix := ""

		if DefaultBool(opts.Prefix, false) {
			sec := *e.Timestamp / 1000
			nsec := (*e.Timestamp % 1000) * 1000
			t := time.Unix(sec, nsec).UTC()

			prefix = fmt.Sprintf("%s %s ", t.Format(time.RFC3339), *e.LogStreamName)
		}

		line := fmt.Sprintf("%s%s\n", prefix, *e.Message)

		if _, err := w.Write([]byte(line)); err != nil {
			return 0, err
		}
	}

	return latest, nil
}
