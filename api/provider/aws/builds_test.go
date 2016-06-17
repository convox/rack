package aws_test

import (
	"os"
	"testing"
	"time"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/structs"

	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("RACK", "convox")
	os.Setenv("DYNAMO_BUILDS", "convox-builds")
	os.Setenv("DYNAMO_RELEASES", "convox-releases")
}

func TestBuildGet(t *testing.T) {
	aws := StubAwsProvider(
		describeStacksCycle,

		build1GetItemCycle,
		build1GetObjectCycle,
	)
	defer aws.Close()

	defer func() {
		//TODO: remove: as we arent updating all tests we need to set current provider back to a
		//clean default one (I miss rspec before)
		provider.CurrentProvider = new(provider.TestProviderRunner)
	}()

	b, err := provider.BuildGet("httpd", "BHINCLZYYVN")

	assert.Nil(t, err)
	assert.EqualValues(t, &structs.Build{
		Id:       "BHINCLZYYVN",
		App:      "httpd",
		Logs:     "RUNNING: docker pull httpd",
		Manifest: "web:\n  image: httpd\n  ports:\n  - 80:80\n",
		Release:  "RVFETUHHKKD",
		Status:   "complete",
		Started:  time.Unix(1459780456, 178278576).UTC(),
		Ended:    time.Unix(1459780542, 440881687).UTC(),
	}, b)
}

func TestBuildDelete(t *testing.T) {
	aws := StubAwsProvider(
		describeStacksCycle,

		build2GetItemCycle,
		build2GetObjectCycle,

		describeStacksCycle,
		releasesBuild2QueryCycle,

		releasesBuild2BatchWriteItemCycle,
		build2DeleteItemCycle,

		build2BatchDeleteImageCycle,
	)
	defer aws.Close()

	defer func() {
		//TODO: remove: as we arent updating all tests we need to set current provider back to a
		//clean default one (I miss rspec before)
		provider.CurrentProvider = new(provider.TestProviderRunner)
	}()

	b, err := provider.BuildDelete("httpd", "BNOARQMVHUO")

	assert.Nil(t, err)
	assert.EqualValues(t, &structs.Build{
		Id:       "BNOARQMVHUO",
		App:      "httpd",
		Logs:     "RUNNING: docker pull httpd",
		Manifest: "web:\n  image: httpd\n  ports:\n  - 80:80\n",
		Release:  "RFVZFLKVTYO",
		Status:   "complete",
		Started:  time.Unix(1459709087, 472025215).UTC(),
		Ended:    time.Unix(1459709198, 984281955).UTC(),
	}, b)
}

func TestBuildDeleteActive(t *testing.T) {
	aws := StubAwsProvider(
		describeStacksCycle,

		build1GetItemCycle,
		build1GetObjectCycle,

		describeStacksCycle,
		releasesBuild1QueryCycle,
	)
	defer aws.Close()

	defer func() {
		//TODO: remove: as we arent updating all tests we need to set current provider back to a
		//clean default one (I miss rspec before)
		provider.CurrentProvider = new(provider.TestProviderRunner)
	}()

	_, err := provider.BuildDelete("httpd", "BHINCLZYYVN")

	assert.Equal(t, err.Error(), "cant delete build contained in active release")
}

func TestBuildList(t *testing.T) {
	aws := StubAwsProvider(
		describeStacksCycle,
		buildsQueryCycle,

		build1GetObjectCycle,
		build2GetObjectCycle,
	)
	defer aws.Close()

	defer func() {
		//TODO: remove: as we arent updating all tests we need to set current provider back to a
		//clean default one (I miss rspec before)
		provider.CurrentProvider = new(provider.TestProviderRunner)
	}()

	b, err := provider.BuildList("httpd", 20)

	assert.Nil(t, err)
	assert.EqualValues(t, structs.Builds{
		structs.Build{
			Id:       "BHINCLZYYVN",
			App:      "httpd",
			Logs:     "RUNNING: docker pull httpd",
			Manifest: "web:\n  image: httpd\n  ports:\n  - 80:80\n",
			Release:  "RVFETUHHKKD",
			Status:   "complete",
			Started:  time.Unix(1459780456, 178278576).UTC(),
			Ended:    time.Unix(1459780542, 440881687).UTC(),
		},
		structs.Build{
			Id:       "BNOARQMVHUO",
			App:      "httpd",
			Logs:     "RUNNING: docker pull httpd",
			Manifest: "web:\n  image: httpd\n  ports:\n  - 80:80\n",
			Release:  "RFVZFLKVTYO",
			Status:   "complete",
			Started:  time.Unix(1459709087, 472025215).UTC(),
			Ended:    time.Unix(1459709198, 984281955).UTC(),
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

var buildsQueryCycle = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.Query",
		Body:       `{"IndexName":"app.created","KeyConditions":{"app":{"AttributeValueList":[{"S":"httpd"}],"ComparisonOperator":"EQ"}},"Limit":20,"ScanIndexForward":false,"TableName":"convox-builds"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{"Count":2,"Items":[{"id":{"S":"BHINCLZYYVN"},"manifest":{"S":"web:\n  image: httpd\n  ports:\n  - 80:80\n"},"release":{"S":"RVFETUHHKKD"},"ended":{"S":"20160404.143542.440881687"},"app":{"S":"httpd"},"created":{"S":"20160404.143416.178278576"},"status":{"S":"complete"}},{"id":{"S":"BNOARQMVHUO"},"manifest":{"S":"web:\n  image: httpd\n  ports:\n  - 80:80\n"},"release":{"S":"RFVZFLKVTYO"},"ended":{"S":"20160403.184638.984281955"},"app":{"S":"httpd"},"created":{"S":"20160403.184447.472025215"},"status":{"S":"complete"}}],"ScannedCount":2}`,
	},
}

var build1GetItemCycle = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.GetItem",
		Body:       `{"ConsistentRead":true,"Key":{"id":{"S":"BHINCLZYYVN"}},"TableName":"convox-builds"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{"Item":{"id":{"S":"BHINCLZYYVN"},"manifest":{"S":"web:\n  image: httpd\n  ports:\n  - 80:80\n"},"ended":{"S":"20160404.143542.440881687"},"release":{"S":"RVFETUHHKKD"},"app":{"S":"httpd"},"created":{"S":"20160404.143416.178278576"},"status":{"S":"complete"}}}`,
	},
}

var build1GetObjectCycle = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/convox-httpd-settings-139bidzalmbtu/builds/BHINCLZYYVN.log",
		Operation:  "",
		Body:       ``,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `RUNNING: docker pull httpd`,
	},
}

var build2BatchDeleteImageCycle = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerRegistry_V20150921.BatchDeleteImage",
		Body:       `{"imageIds":[{"imageTag":"web.BNOARQMVHUO"}],"registryId":"132866487567","repositoryName":"convox-httpd-hqvvfosgxt"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{"failures":[],"imageIds":[{"imageDigest":"sha256:77f27a1381e53241cd230ca1abf74e33ece2715a51e89ba8bdf8908b9a75aa3d","imageTag":"web.BNOARQMVHUO"}]}`,
	},
}

var build2GetItemCycle = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.GetItem",
		Body:       `{"ConsistentRead":true,"Key":{"id":{"S":"BNOARQMVHUO"}},"TableName":"convox-builds"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{"Item":{"id":{"S":"BNOARQMVHUO"},"manifest":{"S":"web:\n  image: httpd\n  ports:\n  - 80:80\n"},"ended":{"S":"20160403.184638.984281955"},"release":{"S":"RFVZFLKVTYO"},"app":{"S":"httpd"},"created":{"S":"20160403.184447.472025215"},"status":{"S":"complete"}}}`,
	},
}

var build2GetObjectCycle = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/convox-httpd-settings-139bidzalmbtu/builds/BNOARQMVHUO.log",
		Operation:  "",
		Body:       ``,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `RUNNING: docker pull httpd`,
	},
}

var build2DeleteItemCycle = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.DeleteItem",
		Body:       `{"Key":{"id":{"S":"BNOARQMVHUO"}},"TableName":"convox-builds"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{}`,
	},
}

var releasesQueryCycle = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.Query",
		Body:       `{"IndexName":"app.created","KeyConditions":{"app":{"AttributeValueList":[{"S":"httpd"}],"ComparisonOperator":"EQ"}},"Limit":20,"ScanIndexForward":false,"TableName":"convox-releases"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{"Count":2,"Items":[{"id":{"S":"RVFETUHHKKD"},"build":{"S":"BHINCLZYYVN"},"app":{"S":"httpd"},"manifest":{"S":"web:\n  image: httpd\n  ports:\n  - 80:80\n"},"env":{"S":"foo=bar"},"created":{"S":"20160404.143542.627770380"}},{"id":{"S":"RFVZFLKVTYO"},"build":{"S":"BNOARQMVHUO"},"app":{"S":"httpd"},"manifest":{"S":"web:\n  image: httpd\n  ports:\n  - 80:80\n"},"env":{"S":"foo=bar"},"created":{"S":"20160403.184639.166694813"}}],"ScannedCount":2}`,
	},
}

var releasesBuild1QueryCycle = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.Query",
		Body:       `{"ExpressionAttributeValues":{":app":{"S":"httpd"},":build":{"S":"BHINCLZYYVN"}},"FilterExpression":"build = :build","IndexName":"app.created","KeyConditionExpression":"app = :app","TableName":"convox-releases"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{"Count":1,"Items":[{"id":{"S":"RVFETUHHKKD"},"build":{"S":"BHINCLZYYVN"},"app":{"S":"httpd"},"manifest":{"S":"web:\n  image: httpd\n  ports:\n  - 80:80\n"},"env":{"S":"foo=bar"},"created":{"S":"20160404.143542.627770380"}}],"ScannedCount":2}`,
	},
}

var releasesBuild2QueryCycle = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.Query",
		Body:       `{"ExpressionAttributeValues":{":app":{"S":"httpd"},":build":{"S":"BNOARQMVHUO"}},"FilterExpression":"build = :build","IndexName":"app.created","KeyConditionExpression":"app = :app","TableName":"convox-releases"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{"Count":1,"Items":[{"id":{"S":"RFVZFLKVTYO"},"build":{"S":"BNOARQMVHUO"},"app":{"S":"httpd"},"manifest":{"S":"web:\n  image: httpd\n  ports:\n  - 80:80\n"},"env":{"S":"foo=bar"},"created":{"S":"20160403.184639.166694813"}}],"ScannedCount":2}`,
	},
}

var releasesBuild2BatchWriteItemCycle = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.BatchWriteItem",
		Body:       `{"RequestItems":{"convox-releases":[{"DeleteRequest":{"Key":{"id":{"S":"RFVZFLKVTYO"}}}}]}}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{"UnprocessedItems":{}}`,
	},
}
