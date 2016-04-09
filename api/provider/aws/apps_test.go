package aws_test

import (
	"os"
	"testing"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/structs"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("RACK", "convox")
}

func TestAppGet(t *testing.T) {
	aws := StubAwsProvider(
		describeStacksCycle,
	)
	defer aws.Close()

	defer func() {
		//TODO: remove: as we arent updating all tests we need to set current provider back to a
		//clean default one (I miss rspec before)
		provider.CurrentProvider = new(provider.TestProviderRunner)
	}()

	a, err := provider.AppGet("httpd")

	assert.Nil(t, err)
	assert.EqualValues(t, &structs.App{
		Name:    "httpd",
		Release: "RVFETUHHKKD",
		Status:  "running",
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
			"Release":                "RVFETUHHKKD",
			"Subnets":                "subnet-13de3139,subnet-b5578fc3,subnet-21c13379",
			"Private":                "Yes",
			"WebPort80ProxyProtocol": "No",
			"VPC":                  "vpc-f8006b9c",
			"Cluster":              "convox-Cluster-1E4XJ0PQWNAYS",
			"Cpu":                  "200",
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

func TestAppGetUnbound(t *testing.T) {
	aws := StubAwsProvider(
		describeStacksUnbound400Cycle,
		describeStacksUnboundCycle,
	)
	defer aws.Close()

	defer func() {
		//TODO: remove: as we arent updating all tests we need to set current provider back to a
		//clean default one (I miss rspec before)
		provider.CurrentProvider = new(provider.TestProviderRunner)
	}()

	a, err := provider.AppGet("httpd-old")

	assert.Nil(t, err)
	assert.EqualValues(t, &structs.App{
		Name:    "httpd-old",
		Release: "RBJFRKXUHTD",
		Status:  "running",
		Outputs: map[string]string{
			"Kinesis":               "httpd-old-Kinesis-1E7IWRINRFHLF",
			"LogGroup":              "httpd-old-LogGroup-P27NBY2OI3CP",
			"RegistryId":            "132866487567",
			"RegistryRepository":    "httpd-old-wcuacldvzi",
			"Settings":              "httpd-old-settings-17w6y79y4ppel",
			"WebPort80Balancer":     "80",
			"WebPort80BalancerName": "httpd-old",
			"BalancerWebHost":       "httpd-old-132500142.us-east-1.elb.amazonaws.com",
		},
		Parameters: map[string]string{
			"Key":                  "arn:aws:kms:us-east-1:132866487567:key/d9f38426-9017-4931-84f8-604ad1524920",
			"Repository":           "",
			"Environment":          "https://httpd-old-settings-17w6y79y4ppel.s3.amazonaws.com/releases/RBJFRKXUHTD/env",
			"VPC":                  "vpc-f8006b9c",
			"Cluster":              "convox-Cluster-1E4XJ0PQWNAYS",
			"Cpu":                  "200",
			"Version":              "20160330143438-command-exec-form",
			"WebPort80Balancer":    "80",
			"WebPort80Host":        "37636",
			"Release":              "RBJFRKXUHTD",
			"WebPort80Secure":      "No",
			"WebPort80Certificate": "",
			"WebMemory":            "256",
			"WebDesiredCount":      "1",
			"Subnets":              "subnet-13de3139,subnet-b5578fc3,subnet-21c13379",
		},
		Tags: map[string]string{
			"Type":   "app",
			"System": "convox",
			"Rack":   "convox",
		},
	}, a)
}

var describeStacksUnbound400Cycle = awsutil.Cycle{
	awsutil.Request{"/", "", `Action=DescribeStacks&StackName=convox-httpd-old&Version=2010-05-15`},
	awsutil.Response{400, `<ErrorResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
  <Error>
    <Type>Sender</Type>
    <Code>ValidationError</Code>
    <Message>Stack with id convox-httpd-old does not exist</Message>
  </Error>
  <RequestId>e451bda1-f773-11e5-aaca-ed87e77a45b8</RequestId>
</ErrorResponse>`},
}

var describeStacksUnboundCycle = awsutil.Cycle{
	awsutil.Request{"/", "", `Action=DescribeStacks&StackName=httpd-old&Version=2010-05-15`},
	awsutil.Response{200, `<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
  <DescribeStacksResult>
    <Stacks>
      <member>
        <Tags>
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
        <StackId>arn:aws:cloudformation:us-east-1:132866487567:stack/httpd-old/0338bdc0-f776-11e5-ae6e-500c524294d2</StackId>
        <StackStatus>UPDATE_COMPLETE</StackStatus>
        <StackName>httpd-old</StackName>
        <LastUpdatedTime>2016-03-31T19:27:16.620Z</LastUpdatedTime>
        <NotificationARNs/>
        <CreationTime>2016-03-31T19:23:13.702Z</CreationTime>
        <Parameters>
          <member>
            <ParameterValue>https://httpd-old-settings-17w6y79y4ppel.s3.amazonaws.com/releases/RBJFRKXUHTD/env</ParameterValue>
            <ParameterKey>Environment</ParameterKey>
          </member>
          <member>
            <ParameterValue/>
            <ParameterKey>WebPort80Certificate</ParameterKey>
          </member>
          <member>
            <ParameterValue>arn:aws:kms:us-east-1:132866487567:key/d9f38426-9017-4931-84f8-604ad1524920</ParameterValue>
            <ParameterKey>Key</ParameterKey>
          </member>
          <member>
            <ParameterValue>256</ParameterValue>
            <ParameterKey>WebMemory</ParameterKey>
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
            <ParameterValue>37636</ParameterValue>
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
            <ParameterValue>RBJFRKXUHTD</ParameterValue>
            <ParameterKey>Release</ParameterKey>
          </member>
          <member>
            <ParameterValue>No</ParameterValue>
            <ParameterKey>WebPort80Secure</ParameterKey>
          </member>
          <member>
            <ParameterValue>200</ParameterValue>
            <ParameterKey>Cpu</ParameterKey>
          </member>
          <member>
            <ParameterValue>subnet-13de3139,subnet-b5578fc3,subnet-21c13379</ParameterValue>
            <ParameterKey>Subnets</ParameterKey>
          </member>
          <member>
            <ParameterValue>20160330143438-command-exec-form</ParameterValue>
            <ParameterKey>Version</ParameterKey>
          </member>
        </Parameters>
        <DisableRollback>false</DisableRollback>
        <Capabilities>
          <member>CAPABILITY_IAM</member>
        </Capabilities>
        <Outputs>
          <member>
            <OutputValue>httpd-old-132500142.us-east-1.elb.amazonaws.com</OutputValue>
            <OutputKey>BalancerWebHost</OutputKey>
          </member>
          <member>
            <OutputValue>httpd-old-Kinesis-1E7IWRINRFHLF</OutputValue>
            <OutputKey>Kinesis</OutputKey>
          </member>
          <member>
            <OutputValue>httpd-old-LogGroup-P27NBY2OI3CP</OutputValue>
            <OutputKey>LogGroup</OutputKey>
          </member>
          <member>
            <OutputValue>132866487567</OutputValue>
            <OutputKey>RegistryId</OutputKey>
          </member>
          <member>
            <OutputValue>httpd-old-wcuacldvzi</OutputValue>
            <OutputKey>RegistryRepository</OutputKey>
          </member>
          <member>
            <OutputValue>httpd-old-settings-17w6y79y4ppel</OutputValue>
            <OutputKey>Settings</OutputKey>
          </member>
          <member>
            <OutputValue>80</OutputValue>
            <OutputKey>WebPort80Balancer</OutputKey>
          </member>
          <member>
            <OutputValue>httpd-old</OutputValue>
            <OutputKey>WebPort80BalancerName</OutputKey>
          </member>
        </Outputs>
      </member>
    </Stacks>
  </DescribeStacksResult>
  <ResponseMetadata>
    <RequestId>e41b6410-f776-11e5-9d00-3de14f66444d</RequestId>
  </ResponseMetadata>
</DescribeStacksResponse>`},
}
