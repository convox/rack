package models

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/s3"
)

func awserrCode(err error) string {
	if ae, ok := err.(awserr.Error); ok {
		return ae.Code()
	}

	return ""
}

func buildEnvironment() string {
	env := []string{
		fmt.Sprintf("AWS_REGION=%s", os.Getenv("AWS_REGION")),
		fmt.Sprintf("AWS_ACCESS=%s", os.Getenv("AWS_ACCESS")),
		fmt.Sprintf("AWS_SECRET=%s", os.Getenv("AWS_SECRET")),
		fmt.Sprintf("GITHUB_TOKEN=%s", os.Getenv("GITHUB_TOKEN")),
	}
	return strings.Join(env, "\n")
}

func cs(s *string, def string) string {
	if s != nil {
		return *s
	} else {
		return def
	}
}

func ct(t *time.Time) time.Time {
	if t != nil {
		return *t
	} else {
		return time.Time{}
	}
}

func coalesce(s *dynamodb.AttributeValue, def string) string {
	if s != nil {
		return *s.S
	} else {
		return def
	}
}

func first(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}

func flattenTags(tags []cloudformation.Tag) map[string]string {
	f := make(map[string]string)

	for _, tag := range tags {
		f[*tag.Key] = *tag.Value
	}

	return f
}

type Template struct {
	Parameters map[string]TemplateParameter
}

type TemplateParameter struct {
	Default     string
	Description string
	Type        string
}

func formationParameters(formation string) (map[string]TemplateParameter, error) {
	var t Template

	err := json.Unmarshal([]byte(formation), &t)

	if err != nil {
		return nil, err
	}

	return t.Parameters, nil
}

var idAlphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func generateId(prefix string, size int) string {
	b := make([]rune, size)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(len(idAlphabet)))
		if err != nil {
			panic(err)
		}
		b[i] = idAlphabet[idx.Int64()]
	}
	return prefix + string(b)
}

func humanStatus(original string) string {
	switch original {
	case "":
		return "new"
	case "CREATE_IN_PROGRESS":
		return "creating"
	case "CREATE_COMPLETE":
		return "running"
	case "DELETE_FAILED":
		return "running"
	case "DELETE_IN_PROGRESS":
		return "deleting"
	case "ROLLBACK_IN_PROGRESS":
		return "rollback"
	case "ROLLBACK_COMPLETE":
		return "failed"
	case "UPDATE_IN_PROGRESS":
		return "updating"
	case "UPDATE_COMPLETE_CLEANUP_IN_PROGRESS":
		return "updating"
	case "UPDATE_COMPLETE":
		return "running"
	case "UPDATE_ROLLBACK_IN_PROGRESS":
		return "rollback"
	case "UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS":
		return "rollback"
	case "UPDATE_ROLLBACK_COMPLETE":
		return "running"
	case "UPDATE_ROLLBACK_FAILED":
		return "running"
	default:
		fmt.Printf("unknown status: %s\n", original)
		return "unknown"
	}
}

func linkParts(link string) (string, string, error) {
	parts := strings.Split(link, ":")

	switch len(parts) {
	case 1:
		return parts[0], parts[0], nil
	case 2:
		return parts[0], parts[1], nil
	}

	return "", "", fmt.Errorf("invalid link name")
}

func prettyJson(raw string) (string, error) {
	var parsed map[string]interface{}

	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		if syntax, ok := err.(*json.SyntaxError); ok {
			lines := strings.Split(raw, "\n")
			lineno := len(strings.Split(raw[0:syntax.Offset], "\n")) - 1
			start := lineno - 3
			end := lineno + 3
			output := "\n"

			if start < 0 {
				start = 0
			}

			if end >= len(lines) {
				end = len(lines) - 1
			}

			for i := start; i <= end; i++ {
				output += fmt.Sprintf("%03d: %s\n", i, lines[i])
			}

			output += err.Error()

			return "", fmt.Errorf(output)
		}

		return "", err
	}

	bp, err := json.MarshalIndent(parsed, "", "  ")

	if err != nil {
		return "", err
	}

	return string(bp), nil
}

func printLines(data string) {
	lines := strings.Split(data, "\n")

	for i, line := range lines {
		fmt.Printf("%d: %s\n", i, line)
	}
}

func s3Delete(bucket, key string) error {
	req := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err := S3().DeleteObject(req)

	return err
}

func s3Get(bucket, key string) ([]byte, error) {
	req := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	res, err := S3().GetObject(req)

	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(res.Body)
}

func S3Put(bucket, key string, data []byte, public bool) error {
	req := &s3.PutObjectInput{
		Body:          bytes.NewReader(data),
		Bucket:        aws.String(bucket),
		ContentLength: aws.Int64(int64(len(data))),
		Key:           aws.String(key),
	}

	if public {
		req.ACL = aws.String("public-read")
	}

	_, err := S3().PutObject(req)

	return err
}

func S3PutFile(bucket, key string, f io.ReadSeeker, public bool) error {
	// seek to end of f to determine length, then seek back to beginning for upload
	l, err := f.Seek(0, 2)

	if err != nil {
		return err
	}

	_, err = f.Seek(0, 0)

	if err != nil {
		return err
	}

	req := &s3.PutObjectInput{
		Body:          f,
		Bucket:        aws.String(bucket),
		ContentLength: aws.Int64(l),
		Key:           aws.String(key),
	}

	if public {
		req.ACL = aws.String("public-read")
	}

	_, err = S3().PutObject(req)

	if err != nil {
		return err
	}

	// seek back to beginning in case something else needs to read f
	_, err = f.Seek(0, 0)

	return err
}

func stackParameters(stack *cloudformation.Stack) map[string]string {
	parameters := make(map[string]string)

	for _, parameter := range stack.Parameters {
		parameters[*parameter.ParameterKey] = *parameter.ParameterValue
	}

	return parameters
}

func stackOutputs(stack *cloudformation.Stack) map[string]string {
	outputs := make(map[string]string)

	for _, output := range stack.Outputs {
		outputs[*output.OutputKey] = *output.OutputValue
	}

	return outputs
}

func stackTags(stack *cloudformation.Stack) map[string]string {
	tags := make(map[string]string)

	for _, tag := range stack.Tags {
		tags[*tag.Key] = *tag.Value
	}

	return tags
}

func templateHelpers() template.FuncMap {
	return template.FuncMap{
		"upper": func(s string) string {
			return UpperName(s)
		},
	}
}

func DashName(name string) string {
	// Myapp -> myapp; MyApp -> my-app
	re := regexp.MustCompile("([a-z])([A-Z])") // lower case letter followed by upper case

	k := re.ReplaceAllString(name, "${1}-${2}")
	return strings.ToLower(k)
}

func UpperName(name string) string {
	// myapp -> Myapp; my-app -> MyApp
	us := strings.ToUpper(name[0:1]) + name[1:]

	for {
		i := strings.Index(us, "-")

		if i == -1 {
			break
		}

		s := us[0:i]

		if len(us) > i+1 {
			s += strings.ToUpper(us[i+1 : i+2])
		}

		if len(us) > i+2 {
			s += us[i+2:]
		}

		us = s
	}

	return us
}
