package aws_test

import (
	"bytes"
	"net/http/httptest"
	"os"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/convox/logger"
	mockaws "github.com/convox/rack/pkg/mock/aws"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/pkg/test/awsutil"
	"github.com/convox/rack/provider/aws"
	"github.com/stretchr/testify/mock"
)

func init() {
	logger.Output = &bytes.Buffer{}
}

type AwsStub struct {
	*aws.Provider
	server *httptest.Server
}

func (a *AwsStub) Close() {
	a.server.Close()
}

// StubAwsProvider creates an httptest server with canned Request / Response
// cycles, and sets CurrentProvider to a new AWS provider that uses
// the test server as the endpoint
func StubAwsProvider(cycles ...awsutil.Cycle) *AwsStub {
	handler := awsutil.NewHandler(cycles)
	s := httptest.NewServer(handler)

	os.Setenv("AWS_ACCESS_KEY_ID", "test-access")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")

	cw := &mockaws.CloudWatchAPI{}
	//input := &cloudwatch.GetMetricDataInput{MetricDataQueries: []*cloudwatch.MetricDataQuery{}}
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

	return &AwsStub{p, s}
}

func testProvider(fn func(p *aws.Provider)) {
	p := &aws.Provider{
		Region: "us-test-1",
	}

	p.Initialize(structs.ProviderOptions{})

	fn(p)
}
