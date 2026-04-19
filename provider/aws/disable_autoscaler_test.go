package aws

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/eventbridge"
	"github.com/convox/rack/pkg/test/awsutil"
)

func stubAutoscalerClients(t *testing.T, cycles ...awsutil.Cycle) (*cloudformation.CloudFormation, *eventbridge.EventBridge, *httptest.Server) {
	t.Helper()
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
	t.Setenv("AWS_REGION", "us-test-1")
	t.Setenv("AWS_SESSION_TOKEN", "")

	handler := awsutil.NewHandler(cycles)
	srv := httptest.NewServer(handler)

	s, err := session.NewSession(&awssdk.Config{
		Region:                        awssdk.String("us-test-1"),
		Endpoint:                      awssdk.String(srv.URL),
		DisableSSL:                    awssdk.Bool(true),
		CredentialsChainVerboseErrors: awssdk.Bool(true),
	})
	if err != nil {
		t.Fatalf("NewSession: %s", err)
	}
	return cloudformation.New(s), eventbridge.New(s), srv
}

// Regex body match on the CloudFormation cycles pins the exact action
// (Action=DescribeStackResource) and logical resource ID. This prevents
// an impl that mistakenly calls DescribeStacks or DescribeStackResources
// (plural) from matching these cycles. CF uses the query protocol
// (form-encoded POST bodies), not the JSON protocol, so Operation is empty.

var cycleDescribeAutoscalerEventResourceFound = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "POST",
		RequestURI: "/",
		Operation:  "",
		Body:       `/Action=DescribeStackResource&.*LogicalResourceId=InstancesAutoscalerEvent/`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `<DescribeStackResourceResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
<DescribeStackResourceResult>
<StackResourceDetail>
<StackName>test-rack</StackName>
<LogicalResourceId>InstancesAutoscalerEvent</LogicalResourceId>
<PhysicalResourceId>test-rack-InstancesAutoscalerEvent-ABC123</PhysicalResourceId>
<ResourceType>AWS::Events::Rule</ResourceType>
<ResourceStatus>CREATE_COMPLETE</ResourceStatus>
</StackResourceDetail>
</DescribeStackResourceResult>
</DescribeStackResourceResponse>`,
	},
}

var cycleDescribeAutoscalerEventResourceMissing = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "POST",
		RequestURI: "/",
		Operation:  "",
		Body:       `/Action=DescribeStackResource&.*LogicalResourceId=InstancesAutoscalerEvent/`,
	},
	Response: awsutil.Response{
		StatusCode: 400,
		Body: `<ErrorResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
<Error>
<Type>Sender</Type>
<Code>ValidationError</Code>
<Message>Resource InstancesAutoscalerEvent does not exist for stack test-rack</Message>
</Error>
</ErrorResponse>`,
	},
}

var cycleDescribeAutoscalerEventResource5xx = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "POST",
		RequestURI: "/",
		Operation:  "",
		Body:       `/Action=DescribeStackResource&.*LogicalResourceId=InstancesAutoscalerEvent/`,
	},
	Response: awsutil.Response{
		StatusCode: 500,
		Body:       `<ErrorResponse><Error><Code>InternalFailure</Code><Message>boom</Message></Error></ErrorResponse>`,
	},
}

// Regex body match on the DisableRule cycles pins the rule name extracted
// from the DescribeStackResource response. If the helper passes the wrong
// value (e.g., StackName or empty), the cycle won't match, awsutil returns
// 404, DisableRule errors, and the helper logs "skip autoscaler disable"
// instead of "disabled autoscaler rule" — failing the happy-path test.
var cycleDisableRuleSuccess = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "POST",
		RequestURI: "/",
		Operation:  "AWSEvents.DisableRule",
		Body:       `/"Name":"test-rack-InstancesAutoscalerEvent-ABC123"/`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{}`,
	},
}

var cycleDisableRuleNotFound = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "POST",
		RequestURI: "/",
		Operation:  "AWSEvents.DisableRule",
		Body:       `/"Name":"test-rack-InstancesAutoscalerEvent-ABC123"/`,
	},
	Response: awsutil.Response{
		StatusCode: 404,
		Body:       `{"__type":"ResourceNotFoundException","message":"Rule not found"}`,
	},
}

func TestDisableAutoscalerRuleHappy(t *testing.T) {
	cf, ev, srv := stubAutoscalerClients(t,
		cycleDescribeAutoscalerEventResourceFound,
		cycleDisableRuleSuccess,
	)
	defer srv.Close()

	var buf bytes.Buffer
	disableAutoscalerRule(cf, ev, "test-rack", &buf)

	out := buf.String()
	if !strings.Contains(out, "disabled autoscaler rule: test-rack-InstancesAutoscalerEvent-ABC123") {
		t.Errorf("expected success log, got: %q", out)
	}
	if strings.Contains(out, "skip autoscaler disable") {
		t.Errorf("unexpected skip log on happy path: %q", out)
	}
}

func TestDisableAutoscalerRuleResourceMissing(t *testing.T) {
	cf, ev, srv := stubAutoscalerClients(t,
		cycleDescribeAutoscalerEventResourceMissing,
	)
	defer srv.Close()

	var buf bytes.Buffer
	disableAutoscalerRule(cf, ev, "test-rack", &buf)

	out := buf.String()
	if !strings.Contains(out, "skip autoscaler disable") {
		t.Errorf("expected skip log for missing resource, got: %q", out)
	}
}

func TestDisableAutoscalerRuleDisableRuleFails(t *testing.T) {
	cf, ev, srv := stubAutoscalerClients(t,
		cycleDescribeAutoscalerEventResourceFound,
		cycleDisableRuleNotFound,
	)
	defer srv.Close()

	var buf bytes.Buffer
	disableAutoscalerRule(cf, ev, "test-rack", &buf)

	out := buf.String()
	if !strings.Contains(out, "skip autoscaler disable") {
		t.Errorf("expected skip log when DisableRule fails, got: %q", out)
	}
}

func TestDisableAutoscalerRuleDescribeReturns5xx(t *testing.T) {
	cf, ev, srv := stubAutoscalerClients(t,
		cycleDescribeAutoscalerEventResource5xx,
	)
	defer srv.Close()

	var buf bytes.Buffer
	disableAutoscalerRule(cf, ev, "test-rack", &buf)

	out := buf.String()
	if !strings.Contains(out, "skip autoscaler disable") {
		t.Errorf("expected skip log on 5xx, got: %q", out)
	}
}
