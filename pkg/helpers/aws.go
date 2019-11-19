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
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
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

func CloudformationDescribe(cf cloudformationiface.CloudFormationAPI, stack string) (*cloudformation.Stack, error) {
	req := &cloudformation.DescribeStacksInput{
		StackName: aws.String(stack),
	}

	res, err := cf.DescribeStacks(req)
	if err != nil {
		return nil, err
	}
	if len(res.Stacks) != 1 {
		return nil, fmt.Errorf("stack not found: %s", stack)
	}

	return res.Stacks[0], nil
}

func CloudformationInstall(cf cloudformationiface.CloudFormationAPI, name, template string, params, tags map[string]string, cb func(int, int)) error {
	req := &cloudformation.CreateChangeSetInput{
		Capabilities:  []*string{aws.String("CAPABILITY_IAM")},
		ChangeSetName: aws.String("init"),
		ChangeSetType: aws.String("CREATE"),
		// ClientRequestToken: aws.String(token),
		Parameters:  []*cloudformation.Parameter{},
		StackName:   aws.String(name),
		Tags:        []*cloudformation.Tag{},
		TemplateURL: aws.String(template),
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

	cres, err := cf.CreateChangeSet(req)
	if err != nil {
		return err
	}

	dreq := &cloudformation.DescribeChangeSetInput{
		ChangeSetName: aws.String("init"),
		StackName:     cres.StackId,
	}

	if err := cf.WaitUntilChangeSetCreateComplete(dreq); err != nil {
		return err
	}

	dres, err := cf.DescribeChangeSet(dreq)
	if err != nil {
		return err
	}

	total := len(dres.Changes)

	cb(0, total)

	token, err := RandomString(20)
	if err != nil {
		return err
	}

	ereq := &cloudformation.ExecuteChangeSetInput{
		ChangeSetName:      aws.String("init"),
		ClientRequestToken: aws.String(token),
		StackName:          cres.StackId,
	}

	if _, err := cf.ExecuteChangeSet(ereq); err != nil {
		return err
	}

	for {
		time.Sleep(10 * time.Second)

		res, err := cf.DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: cres.StackId,
		})
		if err != nil {
			return err
		}
		if len(res.Stacks) != 1 {
			return fmt.Errorf("could not find stack: %s\n", *cres.StackId)
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

func CloudformationParameters(template []byte) (map[string]bool, error) {
	var f struct {
		Parameters map[string]interface{} `yaml:"Parameters"`
	}

	if err := yaml.Unmarshal(template, &f); err != nil {
		return nil, err
	}

	ps := map[string]bool{}

	for p := range f.Parameters {
		ps[p] = true
	}

	return ps, nil
}

func CloudformationUninstall(cf cloudformationiface.CloudFormationAPI, stack string) error {
	_, err := cf.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: aws.String(stack),
	})
	if err != nil {
		return err
	}

	req := &cloudformation.DescribeStacksInput{
		StackName: aws.String(stack),
	}

	if err := cf.WaitUntilStackDeleteComplete(req); err != nil {
		return err
	}

	return nil
}

func CloudformationUpdate(cf cloudformationiface.CloudFormationAPI, stack string, template string, changes map[string]string, tags map[string]string, topic string) error {
	req := &cloudformation.UpdateStackInput{
		Capabilities:     []*string{aws.String("CAPABILITY_IAM")},
		NotificationARNs: []*string{aws.String(topic)},
		StackName:        aws.String(stack),
	}

	params := map[string]bool{}
	pexisting := map[string]bool{}

	s, err := CloudformationDescribe(cf, stack)
	if err != nil {
		return err
	}

	for _, p := range s.Parameters {
		pexisting[*p.ParameterKey] = true
	}

	if template == "" {
		req.UsePreviousTemplate = aws.Bool(true)

		for param := range pexisting {
			params[param] = true
		}
	} else {
		req.TemplateURL = aws.String(template)

		res, err := http.Get(template)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}

		fp, err := CloudformationParameters(body)
		if err != nil {
			return err
		}

		for p := range fp {
			params[p] = true
		}
	}

	sorted := []string{}

	for param := range params {
		sorted = append(sorted, param)
	}

	// sort params for easier testing
	sort.Strings(sorted)

	for _, param := range sorted {
		if value, ok := changes[param]; ok {
			req.Parameters = append(req.Parameters, &cloudformation.Parameter{
				ParameterKey:   aws.String(param),
				ParameterValue: aws.String(value),
			})
		} else if pexisting[param] {
			req.Parameters = append(req.Parameters, &cloudformation.Parameter{
				ParameterKey:     aws.String(param),
				UsePreviousValue: aws.Bool(true),
			})
		}
	}

	req.Tags = s.Tags

	tks := []string{}

	for key := range tags {
		tks = append(tks, key)
	}

	sort.Strings(tks)

	for _, key := range tks {
		req.Tags = append(req.Tags, &cloudformation.Tag{
			Key:   aws.String(key),
			Value: aws.String(tags[key]),
		})
	}

	if _, err := cf.UpdateStack(req); err != nil {
		return err
	}

	// dreq := &cloudformation.DescribeStacksInput{
	//   StackName: aws.String(stack),
	// }

	// if err := cf.WaitUntilStackUpdateComplete(dreq); err != nil {
	//   return err
	// }

	return nil
}

func CloudWatchLogsSubscribe(ctx context.Context, cw cloudwatchlogsiface.CloudWatchLogsAPI, group, stream string, opts structs.LogsOptions) (io.ReadCloser, error) {
	r, w := io.Pipe()

	go CloudWatchLogsStream(ctx, cw, w, group, stream, opts)

	return r, nil
}

func CloudWatchLogsStream(ctx context.Context, cw cloudwatchlogsiface.CloudWatchLogsAPI, w io.WriteCloser, group, stream string, opts structs.LogsOptions) error {
	defer w.Close()

	req := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: aws.String(group),
	}

	if opts.Filter != nil {
		req.FilterPattern = aws.String(*opts.Filter)
	}

	follow := DefaultBool(opts.Follow, true)

	var start int64

	if opts.Since != nil {
		start = time.Now().UTC().Add((*opts.Since)*-1).UnixNano() / int64(time.Millisecond)
		req.StartTime = aws.Int64(start)
	}

	if stream != "" {
		req.LogStreamNames = []*string{aws.String(stream)}
	} else {
		req.Interleaved = aws.Bool(true)
	}

	seen := map[string]bool{}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// check for closed writer
			if _, err := w.Write([]byte{}); err != nil {
				return err
			}

			res, err := cw.FilterLogEvents(req)
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
					seen[*e.EventId] = true
				}

				if e.Timestamp != nil && *e.Timestamp > start {
					start = *e.Timestamp
				}
			}

			sort.Slice(es, func(i, j int) bool { return *es[i].Timestamp < *es[j].Timestamp })

			if _, err := writeLogEvents(w, es, opts); err != nil {
				return err
			}

			req.NextToken = res.NextToken

			if res.NextToken == nil {
				if !follow {
					return nil
				}

				req.StartTime = aws.Int64(start)
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
