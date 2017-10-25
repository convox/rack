package models

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/convox/rack/api/cache"
	"github.com/convox/rack/provider"
)

func awserrCode(err error) string {
	if ae, ok := err.(awserr.Error); ok {
		return ae.Code()
	}

	return ""
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
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(idAlphabet))))
		if err != nil {
			panic(err)
		}
		b[i] = idAlphabet[idx.Int64()]
	}
	return prefix + string(b)
}

func generateSelfSignedCertificate(host string) ([]byte, []byte, error) {
	rkey, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		return nil, nil, err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   host,
			Organization: []string{"convox"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{host},
	}

	data, err := x509.CreateCertificate(rand.Reader, &template, &template, &rkey.PublicKey, rkey)

	if err != nil {
		return nil, nil, err
	}

	pub := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: data})
	key := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rkey)})

	return pub, key, nil
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
		return "error"
	default:
		fmt.Printf("unknown status: %s\n", original)
		return "unknown"
	}
}

// PrettyJSON returns JSON string in a human-readable format
func PrettyJSON(raw string) (string, error) {
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

func rackResource(name string) (string, error) {
	return stackResource(os.Getenv("RACK"), name)
}

func stackResource(stack, resource string) (string, error) {
	res, err := CloudFormation().DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName:         aws.String(stack),
		LogicalResourceId: aws.String(resource),
	})
	if err != nil {
		return "", err
	}
	if len(res.StackResources) < 1 {
		return "", fmt.Errorf("no stack resource for: %s", resource)
	}
	if res.StackResources[0].PhysicalResourceId == nil {
		return "", fmt.Errorf("no stack resource for: %s", resource)
	}

	return *res.StackResources[0].PhysicalResourceId, nil
}

// StackLogGroup returns the cloudwatch log group for an app or rack
func StackLogGroup(app string) (string, error) {
	if g, ok := cache.Get("appLogGroup", app).(string); ok {
		return g, nil
	}

	stackName := os.Getenv("RACK")
	if app != stackName {
		stackName = fmt.Sprintf("%s-%s", os.Getenv("RACK"), app)
	}

	g, err := stackResource(stackName, "LogGroup")
	if err != nil {
		return "", err
	}

	err = cache.Set("appLogGroup", app, g, 10*time.Minute)
	if err != nil {
		return "", err
	}

	return g, nil
}

func shortNameToStackName(appName string) string {
	rack := os.Getenv("RACK")

	if rack == appName {
		// Do no prefix the rack itself.
		return appName
	}

	return rack + "-" + appName
}

func templateHelpers() template.FuncMap {
	return template.FuncMap{
		"coalesce": func(ss ...string) string {
			for _, s := range ss {
				if s != "" {
					return s
				}
			}

			return ""
		},
		"env": func(s string) string {
			return os.Getenv(s)
		},
		"itoa": func(i int) string {
			return strconv.Itoa(i)
		},
		"upper": func(s string) string {
			return UpperName(s)
		},
		"value": func(s string) template.HTML {
			return template.HTML(fmt.Sprintf("%q", s))
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

// TestProvider is a global test provider
var TestProvider = &provider.MockProvider{}

// Provider returns the appropriate provider interface based on the env
func Provider() provider.Provider {
	if os.Getenv("PROVIDER") == "test" {
		return TestProvider
	}

	return provider.FromEnv()
}

// Test provides a wrapping helper for running model tests
func Test(t *testing.T, fn func()) {
	tp := TestProvider
	TestProvider = &provider.MockProvider{}
	defer func() { TestProvider = tp }()
	fn()
	TestProvider.AssertExpectations(t)
}

// DescribeContainerInstances lists and describes all the ECS instances.
// It handles pagination for clusters > 100 instances.
func DescribeContainerInstances() (*ecs.DescribeContainerInstancesOutput, error) {
	instances := []*ecs.ContainerInstance{}
	var nextToken string

	for {
		res, err := ECS().ListContainerInstances(&ecs.ListContainerInstancesInput{
			Cluster:   aws.String(os.Getenv("CLUSTER")),
			NextToken: &nextToken,
		})
		if err != nil {
			return nil, err
		}

		dres, err := ECS().DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
			Cluster:            aws.String(os.Getenv("CLUSTER")),
			ContainerInstances: res.ContainerInstanceArns,
		})
		if err != nil {
			return nil, err
		}

		instances = append(instances, dres.ContainerInstances...)

		// No more container results
		if res.NextToken == nil {
			break
		}

		// set the nextToken to be used for the next iteration
		nextToken = *res.NextToken
	}

	return &ecs.DescribeContainerInstancesOutput{
		ContainerInstances: instances,
	}, nil
}
