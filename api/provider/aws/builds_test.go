package aws_test

import (
	"os"
	"testing"
	"time"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/structs"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("RACK", "convox")
	os.Setenv("DYNAMO_BUILDS", "convox-builds")
}

func TestBuildGet(t *testing.T) {
	aws := StubAwsProvider(
		describeStacksCycle,
		getItemCycle,
		getObjectCycle,
	)
	defer aws.Close()

	defer func() {
		//TODO: remove: as we arent updating all tests we need to set current provider back to a
		//clean default one (I miss rspec before)
		provider.CurrentProvider = new(provider.TestProviderRunner)
	}()

	b, err := provider.BuildGet("httpd", "BVZSXXWEIBT")

	assert.Nil(t, err)
	assert.EqualValues(t, &structs.Build{
		Id:       "BVZSXXWEIBT",
		App:      "httpd",
		Logs:     "RUNNING: docker pull httpd",
		Manifest: "web:\n  image: httpd\n  ports:\n  - 80:80\n",
		Release:  "RLLOVNNXWKR",
		Status:   "complete",
		Started:  time.Unix(1459444265, 29372915).UTC(),
		Ended:    time.Unix(1459444334, 284503073).UTC(),
	}, b)
}

func TestBuildList(t *testing.T) {
	aws := StubAwsProvider(
		describeStacksCycle,
		queryCycle,
		getObjectCycle,
	)
	defer aws.Close()

	defer func() {
		//TODO: remove: as we arent updating all tests we need to set current provider back to a
		//clean default one (I miss rspec before)
		provider.CurrentProvider = new(provider.TestProviderRunner)
	}()

	b, err := provider.BuildList("httpd")

	assert.Nil(t, err)
	assert.EqualValues(t, structs.Builds{
		structs.Build{
			Id:       "BVZSXXWEIBT",
			App:      "httpd",
			Logs:     "RUNNING: docker pull httpd",
			Manifest: "web:\n  image: httpd\n  ports:\n  - 80:80\n",
			Release:  "RLLOVNNXWKR",
			Status:   "complete",
			Started:  time.Unix(1459444265, 29372915).UTC(),
			Ended:    time.Unix(1459444334, 284503073).UTC(),
		},
	}, b)
}

var describeStacksCycle = awsutil.Cycle{
	awsutil.Request{"/", "", `Action=DescribeStacks&StackName=convox-httpd&Version=2010-05-15`},
	awsutil.Response{200, `<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
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
            <ParameterValue>https://convox-httpd-settings-139bidzalmbtu.s3.amazonaws.com/releases/RLLOVNNXWKR/env</ParameterValue>
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
            <ParameterValue>RLLOVNNXWKR</ParameterValue>
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
</DescribeStacksResponse>`},
}

var getItemCycle = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.GetItem",
		Body:       `{"ConsistentRead":true,"Key":{"id":{"S":"BVZSXXWEIBT"}},"TableName":"convox-builds"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{"Item":{"id":{"S":"BVZSXXWEIBT"},"manifest":{"S":"web:\n  image: httpd\n  ports:\n  - 80:80\n"},"ended":{"S":"20160331.171214.284503073"},"release":{"S":"RLLOVNNXWKR"},"app":{"S":"httpd"},"created":{"S":"20160331.171105.029372915"},"status":{"S":"complete"}}}`,
	},
}

var getObjectCycle = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/convox-httpd-settings-139bidzalmbtu/builds/BVZSXXWEIBT.log",
		Operation:  "",
		Body:       ``,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `RUNNING: docker pull httpd`,
	},
}

var queryCycle = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.Query",
		Body:       `{"IndexName":"app.created","KeyConditions":{"app":{"AttributeValueList":[{"S":"httpd"}],"ComparisonOperator":"EQ"}},"Limit":20,"ScanIndexForward":false,"TableName":"convox-builds"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{"Count":1,"Items":[{"id":{"S":"BVZSXXWEIBT"},"manifest":{"S":"web:\n  image: httpd\n  ports:\n  - 80:80\n"},"ended":{"S":"20160331.171214.284503073"},"release":{"S":"RLLOVNNXWKR"},"app":{"S":"httpd"},"created":{"S":"20160331.171105.029372915"},"status":{"S":"complete"}}],"ScannedCount":1}`,
	},
}
