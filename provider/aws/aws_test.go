package aws_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/convox/logger"
	mockaws "github.com/convox/rack/pkg/mock/aws"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/pkg/test/awsutil"
	"github.com/convox/rack/provider/aws"
	"github.com/stretchr/testify/mock"
)

func init() {
	logger.Output = io.Discard
}

type AwsStub struct {
	*aws.Provider
	server  *httptest.Server
	handler *awsutil.Handler

	mu       sync.Mutex
	requests []string
}

func (a *AwsStub) Close() {
	a.server.Close()
}

func (a *AwsStub) Requests() []string {
	a.mu.Lock()
	defer a.mu.Unlock()

	return append([]string{}, a.requests...)
}

// StubAwsProvider creates an httptest server with canned Request / Response
// cycles, and sets CurrentProvider to a new AWS provider that uses
// the test server as the endpoint
func StubAwsProvider(cycles ...awsutil.Cycle) *AwsStub {
	handler := awsutil.NewHandler(cycles)
	stub := &AwsStub{handler: handler}

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(body))

		stub.mu.Lock()
		stub.requests = append(stub.requests, r.Header.Get("X-Amz-Target")+" "+string(body))
		stub.mu.Unlock()

		handler.ServeHTTP(w, r)
	}))

	os.Setenv("AWS_ACCESS_KEY_ID", "test-access")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")

	cw := &mockaws.CloudWatchAPI{}
	output := &cloudwatch.GetMetricDataOutput{MetricDataResults: []*cloudwatch.MetricDataResult{}}

	cw.On("GetMetricData", mock.Anything).Return(output, nil)

	p := &aws.Provider{
		Region:         "us-test-1",
		Endpoint:       s.URL,
		BuildCluster:   "cluster-test",
		Cluster:        "cluster-test",
		Development:    true,
		DynamoBuilds:   "convox-builds",
		DynamoReleases: "convox-releases",
		Password:       "password",
		Rack:           "convox",
		SettingsBucket: "convox-settings",
		SkipCache:      true,
		CloudWatch:     cw,
	}

	stub.Provider = p
	stub.server = s

	return stub
}

func testProvider(fn func(p *aws.Provider)) {
	p := &aws.Provider{
		Region: "us-test-1",
	}

	p.Initialize(structs.ProviderOptions{})

	fn(p)
}
