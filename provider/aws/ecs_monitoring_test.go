package aws

import (
	"io"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/private/protocol/json/jsonutil"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/pkg/test/awsutil"
)

const clearMonitoringWantBody = `{"cluster":"my-cluster","service":"my-service","monitoring":{"metricConfigurations":[]}}`

func testClearMonitoringInput() *clearMonitoringInput {
	return &clearMonitoringInput{
		Cluster:    aws.String("my-cluster"),
		Service:    aws.String("my-service"),
		Monitoring: &clearMonitoringConfig{MetricConfigurations: []*clearMonitoringMetric{}},
	}
}

func TestClearMonitoringInputMarshal(t *testing.T) {
	b, err := jsonutil.BuildJSON(testClearMonitoringInput())
	if err != nil {
		t.Fatalf("BuildJSON error: %v", err)
	}

	if got := string(b); got != clearMonitoringWantBody {
		t.Fatalf("request body mismatch\n got: %s\nwant: %s", got, clearMonitoringWantBody)
	}

	// A nil slice marshals to the omitted form, which AWS does not accept as a clear,
	// so the production code must use a non-nil empty slice to emit [].
	nb, err := jsonutil.BuildJSON(&clearMonitoringInput{
		Cluster:    aws.String("my-cluster"),
		Service:    aws.String("my-service"),
		Monitoring: &clearMonitoringConfig{},
	})
	if err != nil {
		t.Fatalf("BuildJSON error: %v", err)
	}
	if got := string(nb); got != `{"cluster":"my-cluster","service":"my-service","monitoring":{}}` {
		t.Errorf("nil metricConfigurations should omit the key, got: %s", got)
	}
}

func TestClearMonitoringRequestBuild(t *testing.T) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	if err != nil {
		t.Fatalf("session: %v", err)
	}

	op := &request.Operation{Name: "UpdateService", HTTPMethod: "POST", HTTPPath: "/"}
	req := ecs.New(sess).NewRequest(op, testClearMonitoringInput(), &ecs.UpdateServiceOutput{})

	req.Build()
	if req.Error != nil {
		t.Fatalf("build error: %v", req.Error)
	}

	if target := req.HTTPRequest.Header.Get("X-Amz-Target"); target != "AmazonEC2ContainerServiceV20141113.UpdateService" {
		t.Errorf("X-Amz-Target = %q", target)
	}

	body, err := io.ReadAll(req.HTTPRequest.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != clearMonitoringWantBody {
		t.Errorf("request body = %s, want %s", body, clearMonitoringWantBody)
	}
}

func TestServiceHasClassicLB(t *testing.T) {
	classic := &ecs.Service{LoadBalancers: []*ecs.LoadBalancer{{LoadBalancerName: aws.String("my-elb")}}}
	target := &ecs.Service{LoadBalancers: []*ecs.LoadBalancer{{TargetGroupArn: aws.String("arn:...:targetgroup/x/y")}}}
	none := &ecs.Service{}

	if !serviceHasClassicLB(classic) {
		t.Error("classic ELB service should be detected as Classic LB")
	}
	if serviceHasClassicLB(target) {
		t.Error("target-group service must NOT be detected as Classic LB")
	}
	if serviceHasClassicLB(none) {
		t.Error("no-LB service must NOT be detected as Classic LB")
	}
	if serviceHasClassicLB(nil) {
		t.Error("nil service must NOT be detected as Classic LB")
	}
}

func TestECSServiceNamesInStack(t *testing.T) {
	sum := func(rt, pid string) *cloudformation.StackResourceSummary {
		s := &cloudformation.StackResourceSummary{}
		if rt != "" {
			s.ResourceType = aws.String(rt)
		}
		if pid != "" {
			s.PhysicalResourceId = aws.String(pid)
		}
		return s
	}

	srs := []*cloudformation.StackResourceSummary{
		sum("AWS::ECS::Service", "arn:aws:ecs:us-east-1:123456789012:service/my-cluster/web"),
		sum("AWS::Logs::LogGroup", "convox-httpd-LogGroup"),
		sum("Custom::ECSService", "arn:aws:ecs:us-east-1:123456789012:service/worker"),
		sum("AWS::ECS::Service", "bare-name"),
		{},
		sum("AWS::ECS::Service", ""),
	}

	got := ecsServiceNamesInStack(srs)
	want := []string{"web", "worker", "bare-name"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("ecsServiceNamesInStack = %v, want %v", got, want)
	}
}

func TestECSServiceName(t *testing.T) {
	cases := map[string]string{
		"my-service": "my-service",
		"arn:aws:ecs:us-east-1:123456789012:service/my-service":            "my-service",
		"arn:aws:ecs:us-east-1:123456789012:service/my-cluster/my-service": "my-service",
		"": "",
	}

	for in, want := range cases {
		if got := ecsServiceName(in); got != want {
			t.Errorf("ecsServiceName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestChunkStrings(t *testing.T) {
	got := chunkStrings([]string{"a", "b", "c", "d", "e"}, 2)
	want := [][]string{{"a", "b"}, {"c", "d"}, {"e"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("chunkStrings = %v, want %v", got, want)
	}
	if len(chunkStrings(nil, 10)) != 0 {
		t.Error("chunkStrings(nil) should be empty")
	}
}

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = old
	b, _ := io.ReadAll(r)
	return string(b)
}

// Locks the containment guarantee: a Classic-LB service is cleared and a target-group
// service in the same app is never touched.
func TestClearGen1ServiceMonitoringClearsOnlyClassicLB(t *testing.T) {
	os.Setenv("AWS_ACCESS_KEY_ID", "test-access")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")

	listResources := `<ListStackResourcesResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
  <ListStackResourcesResult>
    <StackResourceSummaries>
      <member>
        <PhysicalResourceId>arn:aws:ecs:us-test-1:123456789012:service/cluster-test/web</PhysicalResourceId>
        <LogicalResourceId>ServiceWeb</LogicalResourceId>
        <ResourceType>AWS::ECS::Service</ResourceType>
      </member>
      <member>
        <PhysicalResourceId>arn:aws:ecs:us-test-1:123456789012:service/cluster-test/api</PhysicalResourceId>
        <LogicalResourceId>ServiceApi</LogicalResourceId>
        <ResourceType>AWS::ECS::Service</ResourceType>
      </member>
    </StackResourceSummaries>
  </ListStackResourcesResult>
</ListStackResourcesResponse>`

	describe := `{"services":[
    {"serviceName":"web","loadBalancers":[{"loadBalancerName":"web-elb","containerName":"web","containerPort":80}]},
    {"serviceName":"api","loadBalancers":[{"targetGroupArn":"arn:aws:elasticloadbalancing:us-test-1:123456789012:targetgroup/api/abc","containerName":"api","containerPort":80}]}
  ],"failures":[]}`

	handler := awsutil.NewHandler([]awsutil.Cycle{
		{
			Request:  awsutil.Request{RequestURI: "/", Body: `Action=ListStackResources&StackName=convox-httpd&Version=2010-05-15`},
			Response: awsutil.Response{StatusCode: 200, Body: listResources},
		},
		{
			Request:  awsutil.Request{RequestURI: "/", Body: "ignore"},
			Response: awsutil.Response{StatusCode: 200, Body: describe},
		},
		{
			Request:  awsutil.Request{RequestURI: "/", Body: "ignore"},
			Response: awsutil.Response{StatusCode: 200, Body: `{}`},
		},
	})
	s := httptest.NewServer(handler)
	defer s.Close()

	p := &Provider{Region: "us-test-1", Endpoint: s.URL, Cluster: "cluster-test", Rack: "convox", SkipCache: true}

	out := captureStdout(func() { p.clearGen1ServiceMonitoring("httpd") })

	if !strings.Contains(out, `service="web" err=""`) {
		t.Errorf("expected Classic-LB service \"web\" to be cleared successfully, log:\n%s", out)
	}
	if strings.Contains(out, `service="api"`) {
		t.Errorf("target-group service \"api\" must NOT be touched, log:\n%s", out)
	}
}
