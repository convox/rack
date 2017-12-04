package aws_test

import (
	"os"
	"testing"

	"github.com/convox/rack/test/awsutil"
	"github.com/convox/rack/structs"

	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("RACK", "convox")
}

func TestAppCancel(t *testing.T) {
	provider := StubAwsProvider(
		cycleAppCancelUpdateStack,
	)
	defer provider.Close()

	err := provider.AppCancel("httpd")

	assert.NoError(t, err)
}

func TestAppGet(t *testing.T) {
	provider := StubAwsProvider(
		cycleAppDescribeStacks,
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
			"VPC":                  "vpc-f8006b9c",
			"Cluster":              "convox-Cluster-1E4XJ0PQWNAYS",
			"Key":                  "arn:aws:kms:us-east-1:132866487567:key/d9f38426-9017-4931-84f8-604ad1524920",
			"Repository":           "",
			"WebPort80Balancer":    "80",
			"SubnetsPrivate":       "subnet-d4e85cfe,subnet-103d5a66,subnet-57952a0f",
			"Environment":          "https://convox-httpd-settings-139bidzalmbtu.s3.amazonaws.com/releases/RVFETUHHKKD/env",
			"WebPort80Certificate": "",
			"WebPort80Host":        "56694",
			"WebDesiredCount":      "1",
			"WebPort80Secure":      "No",
			"Version":              "20160330143438-command-exec-form",
		},
		Tags: map[string]string{
			"Name":   "httpd",
			"Type":   "app",
			"System": "convox",
			"Rack":   "convox",
		},
	}, a)
}

var cycleAppCancelUpdateStack = awsutil.Cycle{
	awsutil.Request{
		RequestURI: "/",
		Body:       `Action=CancelUpdateStack&StackName=convox-httpd&Version=2010-05-15`,
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
