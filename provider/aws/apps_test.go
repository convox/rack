package aws_test

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/pkg/test/awsutil"

	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("RACK", "convox")
}

func TestAppCancel(t *testing.T) {
	provider := StubAwsProvider(
		cycleAppDescribeStacks,
		cycleAppCancelUpdateStack,
	)
	defer provider.Close()

	err := provider.AppCancel("httpd")

	assert.NoError(t, err)
}

func TestAppGet(t *testing.T) {
	provider := StubAwsProvider(
		cycleAppDescribeStacks,
		cycleDescribeAppStackResources,
	)
	defer provider.Close()

	a, err := provider.AppGet("httpd")

	assert.NoError(t, err)
	assert.EqualValues(t, &structs.App{
		Generation: "1",
		Name:       "httpd",
		Release:    "RVFETUHHKKD",
		Status:     "running",
		Outputs: map[string]string{
			"BalancerWebHost":       "httpd-web-7E5UPCM-1241527783.us-east-1.elb.amazonaws.com",
			"Kinesis":               "convox-httpd-Kinesis-1MAP0GJ6RITJF",
			"LogGroup":              "convox-httpd-LogGroup-L4V203L35WRM",
			"RegistryId":            "132866487567",
			"RegistryRepository":    "convox-httpd-hqvvfosgxt",
			"Settings":              "convox-httpd-settings-139bidzalmbtu",
			"WebPort80Balancer":     "80",
			"WebPort80BalancerName": "httpd-web-7E5UPCM",
		},
		Parameters: map[string]string{
			"WebMemory":              "256",
			"WebCpu":                 "256",
			"Release":                "RVFETUHHKKD",
			"Subnets":                "subnet-13de3139,subnet-b5578fc3,subnet-21c13379",
			"Private":                "Yes",
			"WebPort80ProxyProtocol": "No",
			"VPC":                    "vpc-f8006b9c",
			"Cluster":                "convox-Cluster-1E4XJ0PQWNAYS",
			"Key":                    "arn:aws:kms:us-east-1:132866487567:key/d9f38426-9017-4931-84f8-604ad1524920",
			"Repository":             "",
			"WebPort80Balancer":      "80",
			"SubnetsPrivate":         "subnet-d4e85cfe,subnet-103d5a66,subnet-57952a0f",
			"Environment":            "https://convox-httpd-settings-139bidzalmbtu.s3.amazonaws.com/releases/RVFETUHHKKD/env",
			"WebPort80Certificate":   "",
			"WebPort80Host":          "56694",
			"WebDesiredCount":        "1",
			"WebPort80Secure":        "No",
			"Version":                "20160330143438-command-exec-form",
		},
		Tags: map[string]string{
			"Name":   "httpd",
			"Type":   "app",
			"System": "convox",
			"Rack":   "convox",
		},
	}, a)
}

func TestAppLogs(t *testing.T) {
	oldNow := helpers.TimeNow
	helpers.TimeNow = func() time.Time {
		return time.Date(2025, 4, 9, 22, 0, 0, 0, time.UTC)
	}
	defer func() { helpers.TimeNow = oldNow }()

	mockedNow := helpers.TimeNow()

	provider := StubAwsProvider(
		cycleListAppStackResources,
		cycleLogFilterLogEvents1,
		cycleLogFilterLogEvents2,
	)
	defer provider.Close()

	buf := &bytes.Buffer{}

	r, err := provider.AppLogs("httpd", structs.LogsOptions{
		Follow: options.Bool(false),
		Filter: options.String("test"),
		Prefix: options.Bool(true),
		Since:  options.Duration(mockedNow.Sub(time.Date(2011, 1, 1, 0, 0, 0, 0, time.UTC))),
	})

	io.Copy(buf, r)

	assert.NoError(t, err)
	assert.Equal(t, "2014-03-28T19:36:18Z stream1 event2\n2014-03-28T19:36:18Z stream2 event3\n2014-03-28T19:36:18Z stream3 event4\n2014-03-28T19:36:18Z stream1 event1\n2014-03-28T19:36:18Z stream2 event5\n", buf.String())
}

func TestAppLogsSince(t *testing.T) {
	oldNow := helpers.TimeNow
	helpers.TimeNow = func() time.Time {
		return time.Date(2025, 4, 9, 22, 0, 0, 0, time.UTC)
	}
	defer func() { helpers.TimeNow = oldNow }()

	mockedNow := helpers.TimeNow()
	wantThen := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)

	since := mockedNow.Sub(wantThen)

	provider := StubAwsProvider(
		cycleListAppStackResources,
		cycleLogFilterLogEventsLimit1,
		cycleLogFilterLogEventsLimit2,
	)
	defer provider.Close()

	r, err := provider.AppLogs("httpd", structs.LogsOptions{
		Follow: options.Bool(false),
		Filter: options.String("test"),
		Prefix: options.Bool(true),
		Since:  options.Duration(since),
	})

	buf := &bytes.Buffer{}
	io.Copy(buf, r)

	assert.NoError(t, err)
	assert.Equal(t, "2020-07-29T15:09:38Z stream1 event2\n2020-07-29T15:09:38Z stream2 event3\n2020-07-29T15:09:38Z stream3 event4\n2020-07-29T15:09:38Z stream1 event1\n2020-07-29T15:09:38Z stream2 event5\n", buf.String())
}

var cycleAppCancelUpdateStack = awsutil.Cycle{
	awsutil.Request{
		RequestURI: "/",
		Body:       `/Action=CancelUpdateStack&ClientRequestToken=[0-9]+-[A-Z]+&StackName=convox-httpd&Version=2010-05-15/`,
	},
	awsutil.Response{
		StatusCode: 200,
		Body: `
			<CancelUpdateStackResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
				<ResponseMetadata>
					<RequestId>5ccc7dcd-744c-11e5-be70-1b08c228efb3</RequestId>
				</ResponseMetadata>
			</CancelUpdateStackResponse>
		`,
	},
}

var cycleAppDescribeStacks = awsutil.Cycle{
	awsutil.Request{"POST", "/", "", `Action=DescribeStacks&StackName=convox-httpd&Version=2010-05-15`},
	awsutil.Response{200, `
		<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
			<DescribeStacksResult>
				<Stacks>
					<member>
						<Tags>
							<member>
								<Value>httpd</Value>
								<Key>Name</Key>
							</member>
							<member>
								<Value>app</Value>
								<Key>Type</Key>
							</member>
							<member>
								<Value>convox</Value>
								<Key>System</Key>
							</member>
							<member>
								<Value>convox</Value>
								<Key>Rack</Key>
							</member>
						</Tags>
						<StackId>arn:aws:cloudformation:us-east-1:132866487567:stack/convox-httpd/53df3c30-f763-11e5-bd5d-50d5cd148236</StackId>
						<StackStatus>UPDATE_COMPLETE</StackStatus>
						<StackName>convox-httpd</StackName>
						<LastUpdatedTime>2016-03-31T17:12:16.275Z</LastUpdatedTime>
						<NotificationARNs/>
						<CreationTime>2016-03-31T17:09:28.583Z</CreationTime>
						<Parameters>
							<member>
								<ParameterValue>https://convox-httpd-settings-139bidzalmbtu.s3.amazonaws.com/releases/RVFETUHHKKD/env</ParameterValue>
								<ParameterKey>Environment</ParameterKey>
							</member>
							<member>
								<ParameterValue/>
								<ParameterKey>WebPort80Certificate</ParameterKey>
							</member>
							<member>
								<ParameterValue>No</ParameterValue>
								<ParameterKey>WebPort80ProxyProtocol</ParameterKey>
							</member>
							<member>
								<ParameterValue>256</ParameterValue>
								<ParameterKey>WebCpu</ParameterKey>
							</member>
							<member>
								<ParameterValue>256</ParameterValue>
								<ParameterKey>WebMemory</ParameterKey>
							</member>
							<member>
								<ParameterValue>arn:aws:kms:us-east-1:132866487567:key/d9f38426-9017-4931-84f8-604ad1524920</ParameterValue>
								<ParameterKey>Key</ParameterKey>
							</member>
							<member>
								<ParameterValue/>
								<ParameterKey>Repository</ParameterKey>
							</member>
							<member>
								<ParameterValue>80</ParameterValue>
								<ParameterKey>WebPort80Balancer</ParameterKey>
							</member>
							<member>
								<ParameterValue>56694</ParameterValue>
								<ParameterKey>WebPort80Host</ParameterKey>
							</member>
							<member>
								<ParameterValue>vpc-f8006b9c</ParameterValue>
								<ParameterKey>VPC</ParameterKey>
							</member>
							<member>
								<ParameterValue>1</ParameterValue>
								<ParameterKey>WebDesiredCount</ParameterKey>
							</member>
							<member>
								<ParameterValue>convox-Cluster-1E4XJ0PQWNAYS</ParameterValue>
								<ParameterKey>Cluster</ParameterKey>
							</member>
							<member>
								<ParameterValue>subnet-d4e85cfe,subnet-103d5a66,subnet-57952a0f</ParameterValue>
								<ParameterKey>SubnetsPrivate</ParameterKey>
							</member>
							<member>
								<ParameterValue>RVFETUHHKKD</ParameterValue>
								<ParameterKey>Release</ParameterKey>
							</member>
							<member>
								<ParameterValue>No</ParameterValue>
								<ParameterKey>WebPort80Secure</ParameterKey>
							</member>
							<member>
								<ParameterValue>subnet-13de3139,subnet-b5578fc3,subnet-21c13379</ParameterValue>
								<ParameterKey>Subnets</ParameterKey>
							</member>
							<member>
								<ParameterValue>20160330143438-command-exec-form</ParameterValue>
								<ParameterKey>Version</ParameterKey>
							</member>
							<member>
								<ParameterValue>Yes</ParameterValue>
								<ParameterKey>Private</ParameterKey>
							</member>
						</Parameters>
						<DisableRollback>false</DisableRollback>
						<Capabilities>
							<member>CAPABILITY_IAM</member>
						</Capabilities>
						<Outputs>
							<member>
								<OutputValue>httpd-web-7E5UPCM-1241527783.us-east-1.elb.amazonaws.com</OutputValue>
								<OutputKey>BalancerWebHost</OutputKey>
							</member>
							<member>
								<OutputValue>convox-httpd-Kinesis-1MAP0GJ6RITJF</OutputValue>
								<OutputKey>Kinesis</OutputKey>
							</member>
							<member>
								<OutputValue>convox-httpd-LogGroup-L4V203L35WRM</OutputValue>
								<OutputKey>LogGroup</OutputKey>
							</member>
							<member>
								<OutputValue>132866487567</OutputValue>
								<OutputKey>RegistryId</OutputKey>
							</member>
							<member>
								<OutputValue>convox-httpd-hqvvfosgxt</OutputValue>
								<OutputKey>RegistryRepository</OutputKey>
							</member>
							<member>
								<OutputValue>convox-httpd-settings-139bidzalmbtu</OutputValue>
								<OutputKey>Settings</OutputKey>
							</member>
							<member>
								<OutputValue>80</OutputValue>
								<OutputKey>WebPort80Balancer</OutputKey>
							</member>
							<member>
								<OutputValue>httpd-web-7E5UPCM</OutputValue>
								<OutputKey>WebPort80BalancerName</OutputKey>
							</member>
						</Outputs>
					</member>
				</Stacks>
			</DescribeStacksResult>
			<ResponseMetadata>
				<RequestId>d5220387-f76d-11e5-912c-531803b112a4</RequestId>
			</ResponseMetadata>
		</DescribeStacksResponse>
	`},
}

var cycleLogFilterLogEventsLimit1 = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "Logs_20140328.FilterLogEvents",
		Body: `{
			"filterPattern": "test",
			"interleaved": true,
			"logGroupName": "convox-httpd-LogGroup-L4V203L35WRM",
			"startTime": 1546300800000
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"events": [
				{
					"ingestionTime": 1396035394997,
					"timestamp": 1596035378988,
					"message": "event2",
					"logStreamName": "stream1",
					"eventId": "31132629274945519779805322857203735586714454643391594505"
				},
				{
					"ingestionTime": 1396035394997,
					"timestamp": 1596035378988,
					"message": "event3",
					"logStreamName": "stream2",
					"eventId": "31132629274945519779805322857203735586814454643391594505"
				},
				{
					"ingestionTime": 1396035394997,
					"timestamp": 1596035378989,
					"message": "event4",
					"logStreamName": "stream3",
					"eventId": "31132629274945519779805322857203735586824454643391594505"
				}
			],
			"searchedLogStreams": [
				{
					"searchedCompletely": false,
					"logStreamName": "stream1"
				},
				{
					"searchedCompletely": false,
					"logStreamName": "stream2"
				}
			],
			"nextToken": "ZNUEPl7FcQuXbIH4Swk9D9eFu2XBg-ijZIZlvzz4ea9zZRjw-MMtQtvcoMdmq4T29K7Q6Y1e_KvyfpcT_f_tUw"
		}`,
	},
}

var cycleLogFilterLogEventsLimit2 = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "Logs_20140328.FilterLogEvents",
		Body: `{
			"filterPattern": "test",
			"interleaved": true,
			"logGroupName": "convox-httpd-LogGroup-L4V203L35WRM",
			"nextToken": "ZNUEPl7FcQuXbIH4Swk9D9eFu2XBg-ijZIZlvzz4ea9zZRjw-MMtQtvcoMdmq4T29K7Q6Y1e_KvyfpcT_f_tUw",
			"startTime": 1546300800000
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"events": [
				{
					"ingestionTime": 1396035394997,
					"timestamp": 1596035378968,
					"message": "event1",
					"logStreamName": "stream1",
					"eventId": "31132629274945519779805322857203735586714454643391594506"
				},
				{
					"ingestionTime": 1396035394997,
					"timestamp": 1596035378998,
					"message": "event5",
					"logStreamName": "stream2",
					"eventId": "31132629274945519779805322857203735586814454643391594507"
				}
			],
			"searchedLogStreams": [
				{
					"searchedCompletely": true,
					"logStreamName": "stream1"
				},
				{
					"searchedCompletely": false,
					"logStreamName": "stream2"
				}
			]
		}`,
	},
}

var cycleLogFilterLogEvents1 = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "Logs_20140328.FilterLogEvents",
		Body: `{
			"filterPattern": "test",
			"interleaved": true,
			"logGroupName": "convox-httpd-LogGroup-L4V203L35WRM",
			"startTime": 1293840000000
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"events": [
				{
					"ingestionTime": 1396035394997,
					"timestamp": 1396035378988,
					"message": "event2",
					"logStreamName": "stream1",
					"eventId": "31132629274945519779805322857203735586714454643391594505"
				},
				{
					"ingestionTime": 1396035394997,
					"timestamp": 1396035378988,
					"message": "event3",
					"logStreamName": "stream2",
					"eventId": "31132629274945519779805322857203735586814454643391594505"
				},
				{
					"ingestionTime": 1396035394997,
					"timestamp": 1396035378989,
					"message": "event4",
					"logStreamName": "stream3",
					"eventId": "31132629274945519779805322857203735586824454643391594505"
				}
			],
			"searchedLogStreams": [
				{
					"searchedCompletely": false,
					"logStreamName": "stream1"
				},
				{
					"searchedCompletely": false,
					"logStreamName": "stream2"
				},
				{
					"searchedCompletely": true,
					"logStreamName": "stream3"
				}
			],
			"nextToken": "ZNUEPl7FcQuXbIH4Swk9D9eFu2XBg-ijZIZlvzz4ea9zZRjw-MMtQtvcoMdmq4T29K7Q6Y1e_KvyfpcT_f_tUw"
		}`,
	},
}

var cycleLogFilterLogEvents2 = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "Logs_20140328.FilterLogEvents",
		Body: `{
			"filterPattern": "test",
			"interleaved": true,
			"logGroupName": "convox-httpd-LogGroup-L4V203L35WRM",
			"nextToken": "ZNUEPl7FcQuXbIH4Swk9D9eFu2XBg-ijZIZlvzz4ea9zZRjw-MMtQtvcoMdmq4T29K7Q6Y1e_KvyfpcT_f_tUw",
			"startTime": 1293840000000
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"events": [
				{
					"ingestionTime": 1396035394997,
					"timestamp": 1396035378968,
					"message": "event1",
					"logStreamName": "stream1",
					"eventId": "31132629274945519779805322857203735586714454643391594506"
				},
				{
					"ingestionTime": 1396035394997,
					"timestamp": 1396035378998,
					"message": "event5",
					"logStreamName": "stream2",
					"eventId": "31132629274945519779805322857203735586814454643391594507"
				}
			],
			"searchedLogStreams": [
				{
					"searchedCompletely": true,
					"logStreamName": "stream1"
				},
				{
					"searchedCompletely": false,
					"logStreamName": "stream2"
				}
			]
		}`,
	},
}

var cycleListAppStackResources = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "",
		Body:       `Action=ListStackResources&StackName=convox-httpd&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<ListStackResourcesResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
  <ListStackResourcesResult>
    <StackResourceSummaries>
    <member>
      <PhysicalResourceId>convox-httpd-LogGroup-L4V203L35WRM</PhysicalResourceId>
      <ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
      <LogicalResourceId>LogGroup</LogicalResourceId>
      <Timestamp>2016-10-22T02:53:23.817Z</Timestamp>
      <ResourceType>AWS::Logs::LogGroup</ResourceType>
    </member>
    </StackResourceSummaries>
  </ListStackResourcesResult>
  <ResponseMetadata>
    <RequestId>50ce1445-9805-11e6-8ba2-2b306877d289</RequestId>
  </ResponseMetadata>
</ListStackResourcesResponse>
		`,
	},
}

var cycleDescribeAppStackResources = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "",
		Body:       `Action=DescribeStackResources&StackName=convox-httpd&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<DescribeStackResourcesResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
  <DescribeStackResourcesResult>
    <StackResources>
    <member>
      <PhysicalResourceId>convox-httpd-LogGroup-L4V203L35WRM</PhysicalResourceId>
      <ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
      <LogicalResourceId>LogGroup</LogicalResourceId>
      <Timestamp>2016-10-22T02:53:23.817Z</Timestamp>
      <ResourceType>AWS::Logs::LogGroup</ResourceType>
    </member>
    </StackResources>
  </DescribeStackResourcesResult>
  <ResponseMetadata>
    <RequestId>50ce1445-9805-11e6-8ba2-2b306877d289</RequestId>
  </ResponseMetadata>
</DescribeStackResourcesResponse>
		`,
	},
}

func TestAppUpdateTags(t *testing.T) {
	tests := []struct {
		name string
		tags string
		want []string
	}{
		{"set", "CostCenter=abc", []string{"CostCenter=abc", "Generation=2", "Name=web", "Rack=convox", "System=convox", "Type=app", "Version=20260929232050"}},
		{"first occurrence wins", "CostCenter=abc,CostCenter=def", []string{"CostCenter=abc", "Generation=2", "Name=web", "Rack=convox", "System=convox", "Type=app", "Version=20260929232050"}},
		{"partial noop", "CostCenter=X,Team=web", []string{"CostCenter=X", "Generation=2", "Name=web", "Rack=convox", "System=convox", "Team=web", "Type=app", "Version=20260929232050"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := StubAwsProvider(
				cycleAppTagsDescribeStacks,
				cycleRackTagsDescribeStacks,
				cycleAppTagsDescribeStacks,
				cycleAppTagsUpdateStack([]string{"TaskTags"}, tt.want),
			)
			defer provider.Close()

			err := provider.AppUpdate("web", structs.AppUpdateOptions{Parameters: map[string]string{"Tags": tt.tags}})

			assert.NoError(t, err)
			assert.Equal(t, 0, provider.handler.Remaining())
		})
	}
}

func TestAppUpdateTagsWithParam(t *testing.T) {
	provider := StubAwsProvider(
		cycleAppTagsDescribeStacks,
		cycleRackTagsDescribeStacks,
		cycleAppTagsDescribeStacks,
		cycleAppTagsUpdateStack([]string{"TaskTags=Yes"}, []string{"CostCenter=abc", "Generation=2", "Name=web", "Rack=convox", "System=convox", "Type=app", "Version=20260929232050"}),
	)
	defer provider.Close()

	err := provider.AppUpdate("web", structs.AppUpdateOptions{Parameters: map[string]string{"Tags": "CostCenter=abc", "TaskTags": "Yes"}})

	assert.NoError(t, err)
	assert.Equal(t, 0, provider.handler.Remaining())
}

func TestAppUpdateTagsEqualParamChanged(t *testing.T) {
	provider := StubAwsProvider(
		cycleAppTagsDescribeStacks,
		cycleRackTagsDescribeStacks,
		cycleAppTagsDescribeStacks,
		cycleAppTagsUpdateStack([]string{"TaskTags=Yes"}, []string{"CostCenter=X", "Generation=2", "Name=web", "Rack=convox", "System=convox", "Type=app", "Version=20260929232050"}),
	)
	defer provider.Close()

	err := provider.AppUpdate("web", structs.AppUpdateOptions{Parameters: map[string]string{"Tags": "CostCenter=X", "TaskTags": "Yes"}})

	assert.NoError(t, err)
	assert.Equal(t, 0, provider.handler.Remaining())
}

func TestAppUpdateNoTags(t *testing.T) {
	provider := StubAwsProvider(
		cycleAppTagsDescribeStacks,
		cycleAppTagsUpdateStack([]string{"TaskTags=Yes"}, []string{"CostCenter=X", "Generation=2", "Name=web", "Rack=convox", "System=convox", "Type=app", "Version=20260929232050"}),
	)
	defer provider.Close()

	err := provider.AppUpdate("web", structs.AppUpdateOptions{Parameters: map[string]string{"TaskTags": "Yes"}})

	assert.NoError(t, err)
	assert.Equal(t, 0, provider.handler.Remaining())
}

func TestAppUpdateTagsNoop(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]string
	}{
		{"same value", map[string]string{"Tags": "CostCenter=X"}},
		{"first occurrence on stack", map[string]string{"Tags": "CostCenter=X,CostCenter=Y"}},
		{"param at current value", map[string]string{"Tags": "CostCenter=X", "TaskTags": "No"}},
		{"param not on stack", map[string]string{"Tags": "CostCenter=X", "Foo": "bar"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := StubAwsProvider(
				cycleAppTagsDescribeStacks,
				cycleRackTagsDescribeStacks,
			)
			defer provider.Close()

			err := provider.AppUpdate("web", structs.AppUpdateOptions{Parameters: tt.params})

			assert.NoError(t, err)
			assert.Equal(t, 0, provider.handler.Remaining())
		})
	}
}

func TestAppUpdateTagsNoopLock(t *testing.T) {
	provider := StubAwsProvider(
		cycleAppTagsDescribeStacks,
		cycleRackTagsDescribeStacks,
		cycleAppTagsUpdateTerminationProtection,
	)
	defer provider.Close()

	err := provider.AppUpdate("web", structs.AppUpdateOptions{Lock: options.Bool(true), Parameters: map[string]string{"Tags": "CostCenter=X"}})

	assert.NoError(t, err)
	assert.Equal(t, 0, provider.handler.Remaining())
}

func TestAppUpdateTagsRejected(t *testing.T) {
	reserved := "Tags cannot set reserved keys: App, Generation, Name, Rack, System, Type, Version"
	empty := "Tags cannot be empty; removing app tags is not supported"
	format := "invalid Tags parameter. expected format: 'key1=val1,key2=val2'"
	prefix := "Tags keys and values cannot use the aws: prefix"

	tests := []struct {
		name   string
		app    string
		tags   string
		lock   bool
		cycles []awsutil.Cycle
		err    string
	}{
		{name: "empty", tags: "", err: empty},
		{name: "no equals", tags: "CostCenter", err: format},
		{name: "empty key", tags: "=x", err: format},
		{name: "App", tags: "App=x", err: reserved},
		{name: "app", tags: "app=x", err: reserved},
		{name: "Generation", tags: "Generation=1", err: reserved},
		{name: "generation", tags: "generation=1", err: reserved},
		{name: "Name", tags: "Name=x", err: reserved},
		{name: "NAME", tags: "NAME=x", err: reserved},
		{name: "Rack", tags: "Rack=x", err: reserved},
		{name: "rack", tags: "rack=x", err: reserved},
		{name: "System", tags: "System=x", err: reserved},
		{name: "system", tags: "system=x", err: reserved},
		{name: "Type", tags: "Type=x", err: reserved},
		{name: "type", tags: "type=x", err: reserved},
		{name: "Version", tags: "Version=x", err: reserved},
		{name: "version", tags: "version=x", err: reserved},
		{name: "reserved with lock", tags: "Rack=x", lock: true, err: reserved},
		{name: "reserved before later bad segment", tags: "Rack=x,foo", err: reserved},
		{name: "aws key", tags: "aws:foo=x", err: prefix},
		{name: "AWS key", tags: "AWS:foo=x", err: prefix},
		{name: "bad key", tags: "R&D=x", err: "invalid Tags key: R&D (allowed: letters, numbers, spaces and _.:/=+-@, up to 128 characters)"},
		{name: "empty value", tags: "CostCenter=", err: empty},
		{name: "aws value", tags: "k=aws:x", err: prefix},
		{name: "bad value", tags: "CostCenter=R&D", err: "invalid Tags value for CostCenter: R&D (allowed: letters, numbers, spaces and _.:/=+-@, up to 256 characters)"},
		{name: "bad duplicate", tags: "A=1,A=R&D", err: "invalid Tags value for A: R&D (allowed: letters, numbers, spaces and _.:/=+-@, up to 256 characters)"},
		{name: "case within input", tags: "CostCenter=a,costcenter=b", err: "Tags key costcenter conflicts with CostCenter; tag keys are case-insensitive"},
		{name: "gen1", app: "httpd", tags: "CostCenter=g1", cycles: []awsutil.Cycle{cycleAppDescribeStacks}, err: "Tags is only supported on generation 2 apps"},
		{name: "case against app stack", tags: "costcenter=y", cycles: []awsutil.Cycle{cycleAppTagsDescribeStacks}, err: "Tags key costcenter conflicts with CostCenter; tag keys are case-insensitive"},
		{name: "case against rack", tags: "env=x", lock: true, cycles: []awsutil.Cycle{cycleAppTagsDescribeStacks, cycleRackTagsDescribeStacks}, err: "Tags key env conflicts with Env; tag keys are case-insensitive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := StubAwsProvider(tt.cycles...)
			defer provider.Close()

			opts := structs.AppUpdateOptions{Parameters: map[string]string{"Tags": tt.tags}}
			if tt.lock {
				opts.Lock = options.Bool(true)
			}

			app := tt.app
			if app == "" {
				app = "web"
			}

			err := provider.AppUpdate(app, opts)

			assert.EqualError(t, err, tt.err)
		})
	}
}

func TestAppGetTags(t *testing.T) {
	provider := StubAwsProvider(
		cycleDescribeAppStack("api", map[string]string{"Generation": "2", "Name": "api", "Rack": "convox", "System": "convox", "Type": "app", "Version": "20260929232050", "Zeta": "z", "Env": "prod", "CostCenter": "abc", "Alpha": "a", "aws:servicecatalog:x": "1"}, nil),
		cycleRackTagsDescribeStacks,
	)
	defer provider.Close()

	a, err := provider.AppGet("api")

	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"Tags": "Alpha=a,CostCenter=abc,Zeta=z"}, a.Parameters)
	assert.Equal(t, 0, provider.handler.Remaining())
}

func TestAppGetTagsInheritedOnly(t *testing.T) {
	provider := StubAwsProvider(
		cycleDescribeAppStack("api", map[string]string{"Generation": "2", "Name": "api", "Rack": "convox", "System": "convox", "Type": "app", "Env": "prod", "aws:servicecatalog:x": "1", "AWS:k": "1", "rack": "x"}, nil),
		cycleRackTagsDescribeStacks,
	)
	defer provider.Close()

	a, err := provider.AppGet("api")

	assert.NoError(t, err)
	_, ok := a.Parameters["Tags"]
	assert.False(t, ok)
	assert.Equal(t, 0, provider.handler.Remaining())
}

func TestAppGetTagsGen1(t *testing.T) {
	provider := StubAwsProvider(
		cycleDescribeAppStack("api", map[string]string{"Name": "api", "Rack": "convox", "System": "convox", "Type": "app", "CostCenter": "X"}, nil),
	)
	defer provider.Close()

	a, err := provider.AppGet("api")

	assert.NoError(t, err)
	_, ok := a.Parameters["Tags"]
	assert.False(t, ok)
	assert.Equal(t, 0, provider.handler.Remaining())
}

func cycleDescribeAppStack(app string, tags, params map[string]string) awsutil.Cycle {
	return cycleDescribeStack("convox-"+app, tags, params)
}

func cycleDescribeStack(name string, tags, params map[string]string) awsutil.Cycle {
	var tx, px strings.Builder

	for k, v := range tags {
		fmt.Fprintf(&tx, "<member><Key>%s</Key><Value>%s</Value></member>", k, v)
	}

	for k, v := range params {
		fmt.Fprintf(&px, "<member><ParameterKey>%s</ParameterKey><ParameterValue>%s</ParameterValue></member>", k, v)
	}

	return awsutil.Cycle{
		Request: awsutil.Request{RequestURI: "/", Body: fmt.Sprintf("Action=DescribeStacks&StackName=%s&Version=2010-05-15", name)},
		Response: awsutil.Response{
			StatusCode: 200,
			Body: fmt.Sprintf(`<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
				<DescribeStacksResult><Stacks><member>
					<StackName>%s</StackName>
					<StackStatus>UPDATE_COMPLETE</StackStatus>
					<Tags>%s</Tags>
					<Parameters>%s</Parameters>
				</member></Stacks></DescribeStacksResult>
			</DescribeStacksResponse>`, name, tx.String(), px.String()),
		},
	}
}

var cycleAppTagsDescribeStacks = cycleDescribeAppStack("web",
	map[string]string{"CostCenter": "X", "Generation": "2", "Name": "web", "Rack": "convox", "System": "convox", "Type": "app", "Version": "20260929232050"},
	map[string]string{"TaskTags": "No"},
)

var cycleRackTagsDescribeStacks = cycleDescribeStack("convox",
	map[string]string{"System": "convox", "Type": "rack", "CostCenter": "R", "Env": "prod"},
	nil,
)

var cycleAppTagsUpdateTerminationProtection = awsutil.Cycle{
	Request: awsutil.Request{RequestURI: "/", Body: "Action=UpdateTerminationProtection&EnableTerminationProtection=true&StackName=convox-web&Version=2010-05-15"},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `<UpdateTerminationProtectionResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
			<UpdateTerminationProtectionResult><StackId>arn:aws:cloudformation:us-test-1:123456789012:stack/convox-web/1</StackId></UpdateTerminationProtectionResult>
		</UpdateTerminationProtectionResponse>`,
	},
}

func cycleAppTagsUpdateStack(params, tags []string) awsutil.Cycle {
	v := url.Values{}
	v.Set("Action", "UpdateStack")
	v.Set("Capabilities.member.1", "CAPABILITY_IAM")
	v.Set("NotificationARNs.member.1", "")
	v.Set("StackName", "convox-web")
	v.Set("UsePreviousTemplate", "true")
	v.Set("Version", "2010-05-15")

	for i, p := range params {
		k, val, set := strings.Cut(p, "=")
		v.Set(fmt.Sprintf("Parameters.member.%d.ParameterKey", i+1), k)
		if set {
			v.Set(fmt.Sprintf("Parameters.member.%d.ParameterValue", i+1), val)
		} else {
			v.Set(fmt.Sprintf("Parameters.member.%d.UsePreviousValue", i+1), "true")
		}
	}

	for i, t := range tags {
		k, val, _ := strings.Cut(t, "=")
		v.Set(fmt.Sprintf("Tags.member.%d.Key", i+1), k)
		v.Set(fmt.Sprintf("Tags.member.%d.Value", i+1), val)
	}

	return awsutil.Cycle{
		Request: awsutil.Request{RequestURI: "/", Body: v.Encode()},
		Response: awsutil.Response{
			StatusCode: 200,
			Body: `<UpdateStackResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
				<UpdateStackResult><StackId>arn:aws:cloudformation:us-test-1:123456789012:stack/convox-web/1</StackId></UpdateStackResult>
			</UpdateStackResponse>`,
		},
	}
}
