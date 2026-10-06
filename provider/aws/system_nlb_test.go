package aws

import (
	"context"
	"strings"
	"testing"

	"github.com/convox/logger"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/pkg/test/awsutil"
)

func TestValidateNLBParams_InternalOnlyConflict(t *testing.T) {
	p := &Provider{InternalOnly: true}
	err := p.validateNLBParams(structs.SystemUpdateOptions{Parameters: map[string]string{"NLB": "Yes"}})
	if err == nil {
		t.Fatal("expected error enabling NLB on InternalOnly rack")
	}
}

func TestValidateNLBParams_InternalOnlyCompatible(t *testing.T) {
	p := &Provider{InternalOnly: true, Internal: true}
	err := p.validateNLBParams(structs.SystemUpdateOptions{Parameters: map[string]string{"NLBInternal": "Yes"}})
	if err != nil {
		t.Fatalf("InternalOnly+Internal+NLBInternal should pass: %v", err)
	}
}

func TestValidateNLBParams_InternalRequiredForNLBInternal(t *testing.T) {
	p := &Provider{Internal: false}
	err := p.validateNLBParams(structs.SystemUpdateOptions{Parameters: map[string]string{"NLBInternal": "Yes"}})
	if err == nil {
		t.Fatal("expected error enabling NLBInternal without Internal=Yes")
	}
}

func TestValidateNLBParams_InternalAndNLBInternalSameCall(t *testing.T) {
	p := &Provider{Internal: false}
	err := p.validateNLBParams(structs.SystemUpdateOptions{
		Parameters: map[string]string{"Internal": "Yes", "NLBInternal": "Yes"},
	})
	if err != nil {
		t.Fatalf("expected Internal+NLBInternal together to pass: %v", err)
	}
}

func TestValidateNLBParams_NoNLBParamChange(t *testing.T) {
	p := &Provider{}
	err := p.validateNLBParams(structs.SystemUpdateOptions{Parameters: map[string]string{"Foo": "bar"}})
	if err != nil {
		t.Fatalf("non-NLB param change should not error: %v", err)
	}
}

func TestValidateNLBParams_NilParameters(t *testing.T) {
	p := &Provider{}
	err := p.validateNLBParams(structs.SystemUpdateOptions{})
	if err != nil {
		t.Fatalf("nil parameters should not error: %v", err)
	}
}

func TestValidateNLBParams_AllowCIDRTooMany(t *testing.T) {
	p := &Provider{NLB: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBAllowCIDR": "1.0.0.0/24,2.0.0.0/24,3.0.0.0/24,4.0.0.0/24,5.0.0.0/24,6.0.0.0/24",
	}}
	err := p.validateNLBParams(opts)
	if err == nil || !strings.Contains(err.Error(), "accepts at most 5") {
		t.Fatalf("expected limit error, got %v", err)
	}
}

func TestValidateNLBParams_AllowCIDRMalformed(t *testing.T) {
	p := &Provider{NLB: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBAllowCIDR": "not-a-cidr",
	}}
	err := p.validateNLBParams(opts)
	if err == nil || !strings.Contains(err.Error(), "not a valid IPv4 CIDR") {
		t.Fatalf("expected shape error, got %v", err)
	}
}

func TestValidateNLBParams_AllowCIDRDuplicate(t *testing.T) {
	p := &Provider{NLB: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBAllowCIDR": "10.0.0.0/24,10.0.0.0/24",
	}}
	err := p.validateNLBParams(opts)
	if err == nil || !strings.Contains(err.Error(), "duplicate entry: 10.0.0.0/24") {
		t.Fatalf("expected dedup error, got %v", err)
	}
}

func TestValidateNLBParams_AllowCIDRInternalMirror(t *testing.T) {
	p := &Provider{NLB: true, NLBInternal: true, Internal: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBInternalAllowCIDR": "1.0.0.0/24,1.0.0.0/24",
	}}
	err := p.validateNLBParams(opts)
	if err == nil || !strings.Contains(err.Error(), "NLBInternalAllowCIDR contains duplicate entry") {
		t.Fatalf("expected internal dedup error, got %v", err)
	}
}

func TestValidateNLBParams_AllowCIDREmptyOK(t *testing.T) {
	p := &Provider{NLB: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBAllowCIDR": "",
	}}
	if err := p.validateNLBParams(opts); err != nil {
		t.Fatalf("empty NLBAllowCIDR should pass: %v", err)
	}
}

func TestValidateNLBParams_AllowCIDRTrailingCommaOK(t *testing.T) {
	p := &Provider{NLB: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBAllowCIDR": "10.0.0.0/24,",
	}}
	if err := p.validateNLBParams(opts); err != nil {
		t.Fatalf("trailing comma should be tolerated: %v", err)
	}
}

type nlbRack struct {
	nlb, nlbInternal, preserve, preserveInternal bool
	sg                                           string
}

func nlbStubProvider(t *testing.T, r nlbRack, cycles ...awsutil.Cycle) (*Provider, *[]string) {
	t.Helper()

	p, bodies := stubProvider(t, cycles...)
	p.ctx = context.Background()
	p.log = logger.New("ns=aws")
	p.DynamoReleases = "convox-releases"
	p.NLB = r.nlb
	p.NLBInternal = r.nlbInternal
	p.Internal = r.nlbInternal
	p.NLBPreserveClientIP = r.preserve
	p.NLBInternalPreserveClientIP = r.preserveInternal
	p.InstanceSecurityGroup = r.sg

	return p, bodies
}

func cycleNLBRackOutput(key, sg string) awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{Method: "POST", RequestURI: "/", Body: `Action=DescribeStacks&StackName=convox&Version=2010-05-15`},
		Response: awsutil.Response{StatusCode: 200, Body: `<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/"><DescribeStacksResult><Stacks><member>
			<StackName>convox</StackName><StackStatus>UPDATE_COMPLETE</StackStatus>
			<Outputs><member><OutputKey>` + key + `</OutputKey><OutputValue>` + sg + `</OutputValue></member></Outputs>
		</member></Stacks></DescribeStacksResult></DescribeStacksResponse>`},
	}
}

func cycleNLBSecurityGroup(perms string) awsutil.Cycle {
	return awsutil.Cycle{
		Request:  awsutil.Request{Method: "POST", RequestURI: "/", Body: `Action=DescribeSecurityGroups&GroupId.1=sg-custom&Version=2016-11-15`},
		Response: awsutil.Response{StatusCode: 200, Body: `<DescribeSecurityGroupsResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/"><securityGroupInfo><item><groupId>sg-custom</groupId>` + perms + `</item></securityGroupInfo></DescribeSecurityGroupsResponse>`},
	}
}

func nlbRule(list, proto, source string) string {
	return `<` + list + `><item><ipProtocol>` + proto + `</ipProtocol><groups><item><groupId>` + source + `</groupId><userId>123456789012</userId></item></groups></item></` + list + `>`
}

var cycleNLBSecurityGroupNotFound = awsutil.Cycle{
	Request:  awsutil.Request{Method: "POST", RequestURI: "/", Body: `Action=DescribeSecurityGroups&GroupId.1=sg-custom&Version=2016-11-15`},
	Response: awsutil.Response{StatusCode: 400, Body: `<Response><Errors><Error><Code>InvalidGroup.NotFound</Code><Message>The security group 'sg-custom' does not exist</Message></Error></Errors><RequestID>r1</RequestID></Response>`},
}

var cycleNLBNoApps = awsutil.Cycle{
	Request:  awsutil.Request{Method: "POST", RequestURI: "/", Body: `Action=DescribeStacks&Version=2010-05-15`},
	Response: awsutil.Response{StatusCode: 200, Body: `<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/"><DescribeStacksResult><Stacks></Stacks></DescribeStacksResult></DescribeStacksResponse>`},
}

func cycleNLBStackDenied(body string) awsutil.Cycle {
	return awsutil.Cycle{
		Request:  awsutil.Request{Method: "POST", RequestURI: "/", Body: body},
		Response: awsutil.Response{StatusCode: 400, Body: `<ErrorResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/"><Error><Type>Sender</Type><Code>AccessDenied</Code><Message>denied</Message></Error><RequestId>r1</RequestId></ErrorResponse>`},
	}
}

func cyclesNLBApp(scheme, preserve string) []awsutil.Cycle {
	m := `services:\n  api:\n    image: x\n    nlb:\n      - port: 8443\n        scheme: ` + scheme + `\n        preserve_client_ip: ` + preserve + `\n`

	return []awsutil.Cycle{
		{
			Request: awsutil.Request{Method: "POST", RequestURI: "/", Body: `Action=DescribeStacks&Version=2010-05-15`},
			Response: awsutil.Response{StatusCode: 200, Body: `<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/"><DescribeStacksResult><Stacks><member>
				<StackName>convox-myapp</StackName><StackStatus>UPDATE_COMPLETE</StackStatus>
				<Tags>
					<member><Key>System</Key><Value>convox</Value></member>
					<member><Key>Type</Key><Value>app</Value></member>
					<member><Key>Rack</Key><Value>convox</Value></member>
					<member><Key>Generation</Key><Value>2</Value></member>
					<member><Key>Name</Key><Value>myapp</Value></member>
				</Tags>
				<Outputs><member><OutputKey>Release</OutputKey><OutputValue>RAPP</OutputValue></member></Outputs>
			</member></Stacks></DescribeStacksResult></DescribeStacksResponse>`},
		},
		{
			Request:  awsutil.Request{Method: "POST", RequestURI: "/", Operation: "DynamoDB_20120810.GetItem", Body: `{"ConsistentRead":true,"Key":{"id":{"S":"RAPP"}},"TableName":"convox-releases"}`},
			Response: awsutil.Response{StatusCode: 200, Body: `{"Item":{"id":{"S":"RAPP"},"app":{"S":"myapp"},"created":{"S":"20160404.143416.178278576"},"manifest":{"S":"` + m + `"}}}`},
		},
		{
			Request:  awsutil.Request{Method: "POST", RequestURI: "/", Body: `Action=ListStackResources&StackName=convox-myapp&Version=2010-05-15`},
			Response: awsutil.Response{StatusCode: 200, Body: `<ListStackResourcesResponse><ListStackResourcesResult><StackResourceSummaries><member><LogicalResourceId>Settings</LogicalResourceId><PhysicalResourceId>myapp-settings</PhysicalResourceId><ResourceType>AWS::S3::Bucket</ResourceType><ResourceStatus>CREATE_COMPLETE</ResourceStatus></member></StackResourceSummaries></ListStackResourcesResult></ListStackResourcesResponse>`},
		},
		{
			Request:  awsutil.Request{Method: "GET", RequestURI: "/myapp-settings/releases/RAPP/env"},
			Response: awsutil.Response{StatusCode: 200, Body: "FOO=bar"},
		},
		{
			Request:  awsutil.Request{Method: "POST", RequestURI: "/", Body: `Action=ListStackResources&StackName=convox&Version=2010-05-15`},
			Response: awsutil.Response{StatusCode: 200, Body: `<ListStackResourcesResponse><ListStackResourcesResult><StackResourceSummaries><member><LogicalResourceId>EncryptionKey</LogicalResourceId><PhysicalResourceId>arn:aws:kms:us-test-1:123456789012:key/k</PhysicalResourceId><ResourceType>AWS::KMS::Key</ResourceType><ResourceStatus>CREATE_COMPLETE</ResourceStatus></member></StackResourceSummaries></ListStackResourcesResult></ListStackResourcesResponse>`},
		},
	}
}

func TestValidateNLBParams_CustomSGRule(t *testing.T) {
	refused := "allowing all traffic from the NLB security group sg-nlb"
	refusedInternal := "allowing all traffic from the NLB security group sg-nlbi"
	outputs := cycleNLBRackOutput("NLBSecurityGroup", "sg-nlb")
	outputsInternal := cycleNLBRackOutput("NLBInternalSecurityGroup", "sg-nlbi")
	rule := cycleNLBSecurityGroup(nlbRule("ipPermissions", "-1", "sg-nlb"))
	noRule := cycleNLBSecurityGroup("")

	tests := []struct {
		name   string
		rack   nlbRack
		params map[string]string
		cycles []awsutil.Cycle
		err    string
		calls  int
	}{
		{"rule present", nlbRack{nlb: true, sg: "sg-custom"}, map[string]string{"NLBPreserveClientIP": "Yes"}, []awsutil.Cycle{outputs, rule}, "", 2},
		{"no rule", nlbRack{nlb: true, sg: "sg-custom"}, map[string]string{"NLBPreserveClientIP": "Yes"}, []awsutil.Cycle{outputs, noRule}, "preserve client IP on the public NLB needs an ingress rule on InstanceSecurityGroup sg-custom allowing all traffic from the NLB security group sg-nlb (convox:NLBSecurityGroup); add that rule and retry", 2},
		{"re-set Yes checked", nlbRack{nlb: true, sg: "sg-custom", preserve: true}, map[string]string{"NLBPreserveClientIP": "Yes"}, []awsutil.Cycle{outputs, noRule}, refused, 2},
		{"other source group", nlbRack{nlb: true, sg: "sg-custom"}, map[string]string{"NLBPreserveClientIP": "Yes"}, []awsutil.Cycle{outputs, cycleNLBSecurityGroup(nlbRule("ipPermissions", "-1", "sg-other"))}, refused, 2},
		{"tcp rule", nlbRack{nlb: true, sg: "sg-custom"}, map[string]string{"NLBPreserveClientIP": "Yes"}, []awsutil.Cycle{outputs, cycleNLBSecurityGroup(nlbRule("ipPermissions", "tcp", "sg-nlb"))}, refused, 2},
		{"egress rule", nlbRack{nlb: true, sg: "sg-custom"}, map[string]string{"NLBPreserveClientIP": "Yes"}, []awsutil.Cycle{outputs, cycleNLBSecurityGroup(nlbRule("ipPermissionsEgress", "-1", "sg-nlb"))}, refused, 2},
		{"ec2 error", nlbRack{nlb: true, sg: "sg-custom"}, map[string]string{"NLBPreserveClientIP": "Yes"}, []awsutil.Cycle{outputs, cycleNLBSecurityGroupNotFound}, "could not read InstanceSecurityGroup sg-custom: InvalidGroup.NotFound", 2},
		{"rack stack error", nlbRack{nlb: true, sg: "sg-custom"}, map[string]string{"NLBPreserveClientIP": "Yes"}, []awsutil.Cycle{cycleNLBStackDenied(`Action=DescribeStacks&StackName=convox&Version=2010-05-15`)}, "could not read rack stack convox: AccessDenied", 1},
		{"missing output", nlbRack{nlb: true, sg: "sg-custom"}, map[string]string{"NLBPreserveClientIP": "Yes"}, []awsutil.Cycle{cycleNLBRackOutput("Other", "x")}, "rack stack has no NLBSecurityGroup output", 1},
		{"internal", nlbRack{nlbInternal: true, sg: "sg-custom"}, map[string]string{"NLBInternalPreserveClientIP": "Yes"}, []awsutil.Cycle{outputsInternal, cycleNLBSecurityGroup(nlbRule("ipPermissions", "-1", "sg-nlbi"))}, "", 2},
		{"internal needs its own group", nlbRack{nlb: true, nlbInternal: true, sg: "sg-custom"}, map[string]string{"NLBInternalPreserveClientIP": "Yes"}, []awsutil.Cycle{outputsInternal, rule}, refusedInternal, 2},
		{"blank sg", nlbRack{nlb: true}, map[string]string{"NLBPreserveClientIP": "Yes"}, nil, "", 0},
		{"custom to blank", nlbRack{nlb: true, sg: "sg-custom"}, map[string]string{"InstanceSecurityGroup": "", "NLBPreserveClientIP": "Yes"}, nil, "", 0},
		{"nlb off", nlbRack{sg: "sg-custom", preserve: true}, map[string]string{"InstanceSecurityGroup": "sg-other"}, nil, "", 0},
		{"nlb enabled with preserve on", nlbRack{sg: "sg-custom", preserve: true}, map[string]string{"NLB": "Yes"}, nil, "cannot enable NLB while NLBPreserveClientIP=Yes on a rack with a custom InstanceSecurityGroup; set NLBPreserveClientIP=No in this command, add an ingress rule to sg-custom allowing all traffic from the new convox:NLBSecurityGroup, then set NLBPreserveClientIP=Yes", 0},
		{"internal nlb enabled with preserve on", nlbRack{nlb: true, sg: "sg-custom", preserveInternal: true}, map[string]string{"Internal": "Yes", "NLBInternal": "Yes"}, nil, "cannot enable NLBInternal while NLBInternalPreserveClientIP=Yes on a rack with a custom InstanceSecurityGroup; set NLBInternalPreserveClientIP=No in this command, add an ingress rule to sg-custom allowing all traffic from the new convox:NLBInternalSecurityGroup, then set NLBInternalPreserveClientIP=Yes", 0},
		{"nlb enabled with preserve off", nlbRack{sg: "sg-custom", preserve: true}, map[string]string{"NLB": "Yes", "NLBPreserveClientIP": "No"}, nil, "", 0},
		{"nlb enabled with sg change and preserve off", nlbRack{sg: "sg-old", preserve: true}, map[string]string{"NLB": "Yes", "NLBPreserveClientIP": "No", "InstanceSecurityGroup": "sg-custom"}, nil, "", 0},
		{"blank to custom with preserve on", nlbRack{nlb: true, preserve: true}, map[string]string{"InstanceSecurityGroup": "sg-custom"}, []awsutil.Cycle{outputs, noRule}, refused, 2},
		{"custom to custom checks new sg", nlbRack{nlb: true, sg: "sg-old", preserve: true}, map[string]string{"InstanceSecurityGroup": "sg-custom"}, []awsutil.Cycle{outputs, rule}, "", 2},
		{"same-call preserve off still checked", nlbRack{nlb: true, preserve: true}, map[string]string{"InstanceSecurityGroup": "sg-custom", "NLBPreserveClientIP": "No"}, []awsutil.Cycle{outputs, noRule}, refused, 2},
		{"both schemes checked on sg change", nlbRack{nlb: true, nlbInternal: true, preserve: true, preserveInternal: true}, map[string]string{"InstanceSecurityGroup": "sg-custom"}, []awsutil.Cycle{outputs, rule, outputsInternal, noRule}, refusedInternal, 4},
		{"internal sg change with preserve on", nlbRack{nlbInternal: true, preserveInternal: true}, map[string]string{"InstanceSecurityGroup": "sg-custom"}, []awsutil.Cycle{outputsInternal, noRule}, refusedInternal, 2},
		{"sg change without preserve", nlbRack{nlb: true}, map[string]string{"InstanceSecurityGroup": "sg-custom"}, []awsutil.Cycle{cycleNLBNoApps}, "", 1},
		{"sg change with app preserve true", nlbRack{nlb: true}, map[string]string{"InstanceSecurityGroup": "sg-custom"}, append(cyclesNLBApp("public", "true"), outputs, noRule), refused, 7},
		{"sg change with internal app preserve true", nlbRack{nlbInternal: true}, map[string]string{"InstanceSecurityGroup": "sg-custom"}, append(cyclesNLBApp("internal", "true"), outputsInternal, noRule), refusedInternal, 7},
		{"sg change with app preserve false", nlbRack{nlb: true}, map[string]string{"InstanceSecurityGroup": "sg-custom"}, cyclesNLBApp("public", "false"), "", 5},
		{"sg change with app preserve unset", nlbRack{nlb: true}, map[string]string{"InstanceSecurityGroup": "sg-custom"}, cyclesNLBApp("public", "null"), "", 5},
		{"app scan error", nlbRack{nlb: true}, map[string]string{"InstanceSecurityGroup": "sg-custom"}, []awsutil.Cycle{cycleNLBStackDenied(`Action=DescribeStacks&Version=2010-05-15`)}, "AccessDenied", 1},
		{"unrelated param", nlbRack{nlb: true, sg: "sg-custom", preserve: true}, map[string]string{"Foo": "bar"}, nil, "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, bodies := nlbStubProvider(t, tt.rack, tt.cycles...)
			err := p.validateNLBParams(structs.SystemUpdateOptions{Parameters: tt.params})
			if tt.err == "" && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.err != "" && (err == nil || !strings.Contains(err.Error(), tt.err)) {
				t.Fatalf("expected error containing %q, got %v", tt.err, err)
			}
			if len(*bodies) != tt.calls {
				t.Fatalf("expected %d AWS calls, got %d", tt.calls, len(*bodies))
			}
		})
	}
}

func TestValidateNLBParams_DisableBlockedByApp(t *testing.T) {
	p, bodies := nlbStubProvider(t, nlbRack{nlb: true}, cyclesNLBApp("public", "false")...)
	err := p.validateNLBParams(structs.SystemUpdateOptions{Parameters: map[string]string{"NLB": "No"}})
	if err == nil || !strings.Contains(err.Error(), "cannot disable NLB: apps myapp/api still declare public nlb ports") {
		t.Fatalf("expected disable blocked by app, got %v", err)
	}
	if len(*bodies) != 5 {
		t.Fatalf("expected 5 AWS calls, got %d", len(*bodies))
	}
}

func TestValidateNLBParams_AllowCIDRLeadingSpaceRejected(t *testing.T) {
	p := &Provider{NLB: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBAllowCIDR": "10.0.0.0/24, 10.0.0.1/32",
	}}
	err := p.validateNLBParams(opts)
	if err == nil || !strings.Contains(err.Error(), "leading or trailing whitespace") {
		t.Fatalf("expected whitespace rejection, got %v", err)
	}
}

func TestValidateNLBParams_AllowCIDRInvalidOctet(t *testing.T) {
	p := &Provider{NLB: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBAllowCIDR": "256.0.0.0/8",
	}}
	err := p.validateNLBParams(opts)
	if err == nil || !strings.Contains(err.Error(), "not a valid IPv4 CIDR") {
		t.Fatalf("expected invalid-octet rejection, got %v", err)
	}
}

func TestValidateNLBParams_AllowCIDRInvalidMask(t *testing.T) {
	p := &Provider{NLB: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBAllowCIDR": "10.0.0.0/33",
	}}
	err := p.validateNLBParams(opts)
	if err == nil || !strings.Contains(err.Error(), "not a valid IPv4 CIDR") {
		t.Fatalf("expected invalid-mask rejection, got %v", err)
	}
}

func TestValidateNLBParams_AllowCIDRNonCanonicalRejected(t *testing.T) {
	p := &Provider{NLB: true}
	opts := structs.SystemUpdateOptions{Parameters: map[string]string{
		"NLBAllowCIDR": "10.0.0.1/24",
	}}
	err := p.validateNLBParams(opts)
	if err == nil || !strings.Contains(err.Error(), "not canonical") {
		t.Fatalf("expected non-canonical rejection, got %v", err)
	}
}
