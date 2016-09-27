package aws_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/provider/aws"
	"github.com/stretchr/testify/assert"
)

func TestSystemGet(t *testing.T) {
	provider := StubAwsProvider(
		cycleSystemDescribeStacks,
	)
	defer provider.Close()

	s, err := provider.SystemGet()

	assert.Nil(t, err)
	assert.EqualValues(t, &structs.System{
		Count:   3,
		Name:    "convox",
		Region:  "us-test-1",
		Status:  "running",
		Type:    "t2.small",
		Version: "dev",
	}, s)
}

func TestSystemGetBadStack(t *testing.T) {
	provider := StubAwsProvider(
		cycleDescribeStacksNotFound("convox"),
	)
	defer provider.Close()

	r, err := provider.SystemGet()

	assert.Nil(t, r)
	assert.True(t, aws.ErrorNotFound(err))
	assert.Equal(t, "convox not found", err.Error())
}

func TestSystemReleases(t *testing.T) {
	provider := StubAwsProvider(
		cycleSystemReleaseList,
	)
	defer provider.Close()

	r, err := provider.SystemReleases()

	assert.Nil(t, err)

	assert.EqualValues(t, structs.Releases{
		structs.Release{
			Id:      "test1",
			App:     "convox",
			Created: time.Unix(1459780542, 627770380).UTC(),
		},
		structs.Release{
			Id:      "test2",
			App:     "convox",
			Created: time.Unix(1459709199, 166694813).UTC(),
		},
	}, r)
}

func TestSystemSave(t *testing.T) {
	provider := StubAwsProvider(
		cycleSystemDescribeStacks,
		cycleReleasePutItem,
		cycleSystemUpdateNotificationPublish,
		cycleSystemDescribeStacks,
		cycleSystemUpdateStack,
	)
	defer provider.Close()

	err := provider.SystemSave(structs.System{
		Count:   5,
		Type:    "t2.small",
		Version: "20160820033210",
	})

	assert.Nil(t, err)
}

func TestSystemSaveNewParameter(t *testing.T) {
	provider := StubAwsProvider(
		cycleSystemDescribeStacks,
		cycleReleasePutItem,
		cycleSystemUpdateNotificationPublish,
		cycleSystemDescribeStacksMissingParameters,
		cycleSystemUpdateStackNewParameter,
	)
	defer provider.Close()

	err := provider.SystemSave(structs.System{
		Count:   5,
		Type:    "t2.small",
		Version: "20160820033210",
	})

	assert.Nil(t, err)
}

func TestSystemSaveWrongType(t *testing.T) {
	sys := structs.System{
		Name:    "name",
		Version: "version",
		Type:    "wrongtype",
	}

	provider := &aws.AWSProvider{}

	err := provider.SystemSave(sys)

	assert.Equal(t, err, fmt.Errorf("invalid instance type: wrongtype"))
}

var cycleSystemDescribeStacks = awsutil.Cycle{
	awsutil.Request{"/", "", `Action=DescribeStacks&StackName=convox&Version=2010-05-15`},
	awsutil.Response{
		200,
		`<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
			<DescribeStacksResult>
				<Stacks>
					<member>
						<Outputs>
						</Outputs>
						<Capabilities>
							<member>CAPABILITY_IAM</member>
						</Capabilities>
						<CreationTime>2015-10-28T16:14:09.590Z</CreationTime>
						<NotificationARNs/>
						<StackId>arn:aws:cloudformation:us-east-1:778743527532:stack/convox/eb743e00-7d8e-11e5-8280-50ba0727c06e</StackId>
						<StackName>convox</StackName>
						<StackStatus>UPDATE_COMPLETE</StackStatus>
						<DisableRollback>false</DisableRollback>
						<Tags/>
						<LastUpdatedTime>2016-08-27T16:29:05.963Z</LastUpdatedTime>
						<Parameters>
							<member>
								<ParameterKey>Tenancy</ParameterKey>
								<ParameterValue>default</ParameterValue>
							</member>
							<member>
								<ParameterKey>Internal</ParameterKey>
								<ParameterValue>No</ParameterValue>
							</member>
							<member>
								<ParameterKey>ApiCpu</ParameterKey>
								<ParameterValue>128</ParameterValue>
							</member>
							<member>
								<ParameterKey>PrivateApi</ParameterKey>
								<ParameterValue>No</ParameterValue>
							</member>
							<member>
								<ParameterKey>ContainerDisk</ParameterKey>
								<ParameterValue>10</ParameterValue>
							</member>
							<member>
								<ParameterKey>SwapSize</ParameterKey>
								<ParameterValue>5</ParameterValue>
							</member>
							<member>
								<ParameterKey>Encryption</ParameterKey>
								<ParameterValue>Yes</ParameterValue>
							</member>
							<member>
								<ParameterKey>Subnet1CIDR</ParameterKey>
								<ParameterValue>10.0.2.0/24</ParameterValue>
							</member>
							<member>
								<ParameterKey>Autoscale</ParameterKey>
								<ParameterValue>No</ParameterValue>
							</member>
							<member>
								<ParameterKey>Version</ParameterKey>
								<ParameterValue>dev</ParameterValue>
							</member>
							<member>
								<ParameterKey>VPCCIDR</ParameterKey>
								<ParameterValue>10.0.0.0/16</ParameterValue>
							</member>
							<member>
								<ParameterKey>Development</ParameterKey>
								<ParameterValue>Yes</ParameterValue>
							</member>
							<member>
								<ParameterKey>ClientId</ParameterKey>
								<ParameterValue>nmert38iwdsrj362jdf</ParameterValue>
							</member>
							<member>
								<ParameterKey>Private</ParameterKey>
								<ParameterValue>No</ParameterValue>
							</member>
							<member>
								<ParameterKey>Subnet2CIDR</ParameterKey>
								<ParameterValue>10.0.3.0/24</ParameterValue>
							</member>
							<member>
								<ParameterKey>Ami</ParameterKey>
								<ParameterValue/>
							</member>
							<member>
								<ParameterKey>InstanceType</ParameterKey>
								<ParameterValue>t2.small</ParameterValue>
							</member>
							<member>
								<ParameterKey>SubnetPrivate1CIDR</ParameterKey>
								<ParameterValue>10.0.5.0/24</ParameterValue>
							</member>
							<member>
								<ParameterKey>VolumeSize</ParameterKey>
								<ParameterValue>50</ParameterValue>
							</member>
							<member>
								<ParameterKey>Password</ParameterKey>
								<ParameterValue>****</ParameterValue>
							</member>
							<member>
								<ParameterKey>ApiMemory</ParameterKey>
								<ParameterValue>128</ParameterValue>
							</member>
							<member>
								<ParameterKey>InstanceUpdateBatchSize</ParameterKey>
								<ParameterValue>1</ParameterValue>
							</member>
							<member>
								<ParameterKey>SubnetPrivate0CIDR</ParameterKey>
								<ParameterValue>10.0.4.0/24</ParameterValue>
							</member>
							<member>
								<ParameterKey>InstanceRunCommand</ParameterKey>
								<ParameterValue/>
							</member>
							<member>
								<ParameterKey>InstanceCount</ParameterKey>
								<ParameterValue>3</ParameterValue>
							</member>
							<member>
								<ParameterKey>SubnetPrivate2CIDR</ParameterKey>
								<ParameterValue>10.0.6.0/24</ParameterValue>
							</member>
							<member>
								<ParameterKey>Subnet0CIDR</ParameterKey>
								<ParameterValue>10.0.1.0/24</ParameterValue>
							</member>
							<member>
								<ParameterKey>ExistingVpc</ParameterKey>
								<ParameterValue/>
							</member>
							<member>
								<ParameterKey>InstanceBootCommand</ParameterKey>
								<ParameterValue/>
							</member>
							<member>
								<ParameterKey>Key</ParameterKey>
								<ParameterValue>convox-keypair-4415</ParameterValue>
							</member>
						</Parameters>
					</member>
				</Stacks>
			</DescribeStacksResult>
			<ResponseMetadata>
				<RequestId>9715cab7-6c75-11e6-837d-ebe72becd936</RequestId>
			</ResponseMetadata>
		</DescribeStacksResponse>`,
	},
}

var cycleSystemDescribeStacksMissingParameters = awsutil.Cycle{
	awsutil.Request{"/", "", `Action=DescribeStacks&StackName=convox&Version=2010-05-15`},
	awsutil.Response{
		200,
		`<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
			<DescribeStacksResult>
				<Stacks>
					<member>
						<Outputs>
						</Outputs>
						<Capabilities>
							<member>CAPABILITY_IAM</member>
						</Capabilities>
						<CreationTime>2015-10-28T16:14:09.590Z</CreationTime>
						<NotificationARNs/>
						<StackId>arn:aws:cloudformation:us-east-1:778743527532:stack/convox/eb743e00-7d8e-11e5-8280-50ba0727c06e</StackId>
						<StackName>convox</StackName>
						<StackStatus>UPDATE_COMPLETE</StackStatus>
						<DisableRollback>false</DisableRollback>
						<Tags/>
						<LastUpdatedTime>2016-08-27T16:29:05.963Z</LastUpdatedTime>
						<Parameters>
							<member>
								<ParameterKey>Ami</ParameterKey>
								<ParameterValue/>
							</member>
						</Parameters>
					</member>
				</Stacks>
			</DescribeStacksResult>
			<ResponseMetadata>
				<RequestId>9715cab7-6c75-11e6-837d-ebe72becd936</RequestId>
			</ResponseMetadata>
		</DescribeStacksResponse>`,
	},
}

var cycleSystemReleaseList = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.Query",
		Body:       `{"IndexName":"app.created","KeyConditions":{"app":{"AttributeValueList":[{"S":"convox"}],"ComparisonOperator":"EQ"}},"Limit":20,"ScanIndexForward":false,"TableName":"convox-releases"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{"Count":2,"Items":[{"id":{"S":"test1"},"app":{"S":"convox"},"created":{"S":"20160404.143542.627770380"}},{"id":{"S":"test2"},"app":{"S":"convox"},"created":{"S":"20160403.184639.166694813"}}],"ScannedCount":2}`,
	},
}

var cycleReleasePutItem = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.PutItem",
		Body:       `{"Item":{"app":{"S":"convox"},"created":{"S":"00010101.000000.000000000"},"id":{"S":"20160820033210"}},"TableName":"convox-releases"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{}`,
	},
}

var cycleSystemUpdateNotificationPublish = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=Publish&Message=%7B%22action%22%3A%22rack%3Aupdate%22%2C%22status%22%3A%22success%22%2C%22data%22%3A%7B%22count%22%3A%225%22%2C%22version%22%3A%2220160820033210%22%7D%2C%22timestamp%22%3A%220001-01-01T00%3A00%3A00Z%22%7D&Subject=rack%3Aupdate&TargetArn=&Version=2010-03-31`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<PublishResponse xmlns="http://sns.amazonaws.com/doc/2010-03-31/">
				<PublishResult>
					<MessageId>94f20ce6-13c5-43a0-9a9e-ca52d816e90b</MessageId>
				</PublishResult>
				<ResponseMetadata>
					<RequestId>f187a3c1-376f-11df-8963-01868b7c937a</RequestId>
				</ResponseMetadata>
			</PublishResponse>
		`,
	},
}

var cycleSystemUpdateStack = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=UpdateStack&Capabilities.member.1=CAPABILITY_IAM&Parameters.member.1.ParameterKey=Ami&Parameters.member.1.UsePreviousValue=true&Parameters.member.10.ParameterKey=InstanceBootCommand&Parameters.member.10.UsePreviousValue=true&Parameters.member.11.ParameterKey=InstanceCount&Parameters.member.11.ParameterValue=5&Parameters.member.12.ParameterKey=InstanceRunCommand&Parameters.member.12.UsePreviousValue=true&Parameters.member.13.ParameterKey=InstanceType&Parameters.member.13.ParameterValue=t2.small&Parameters.member.14.ParameterKey=InstanceUpdateBatchSize&Parameters.member.14.UsePreviousValue=true&Parameters.member.15.ParameterKey=Internal&Parameters.member.15.UsePreviousValue=true&Parameters.member.16.ParameterKey=Key&Parameters.member.16.UsePreviousValue=true&Parameters.member.17.ParameterKey=Password&Parameters.member.17.UsePreviousValue=true&Parameters.member.18.ParameterKey=Private&Parameters.member.18.UsePreviousValue=true&Parameters.member.19.ParameterKey=PrivateApi&Parameters.member.19.UsePreviousValue=true&Parameters.member.2.ParameterKey=ApiCpu&Parameters.member.2.UsePreviousValue=true&Parameters.member.20.ParameterKey=Subnet0CIDR&Parameters.member.20.UsePreviousValue=true&Parameters.member.21.ParameterKey=Subnet1CIDR&Parameters.member.21.UsePreviousValue=true&Parameters.member.22.ParameterKey=Subnet2CIDR&Parameters.member.22.UsePreviousValue=true&Parameters.member.23.ParameterKey=SubnetPrivate0CIDR&Parameters.member.23.UsePreviousValue=true&Parameters.member.24.ParameterKey=SubnetPrivate1CIDR&Parameters.member.24.UsePreviousValue=true&Parameters.member.25.ParameterKey=SubnetPrivate2CIDR&Parameters.member.25.UsePreviousValue=true&Parameters.member.26.ParameterKey=SwapSize&Parameters.member.26.UsePreviousValue=true&Parameters.member.27.ParameterKey=Tenancy&Parameters.member.27.UsePreviousValue=true&Parameters.member.28.ParameterKey=VPCCIDR&Parameters.member.28.UsePreviousValue=true&Parameters.member.29.ParameterKey=Version&Parameters.member.29.ParameterValue=20160820033210&Parameters.member.3.ParameterKey=ApiMemory&Parameters.member.3.UsePreviousValue=true&Parameters.member.30.ParameterKey=VolumeSize&Parameters.member.30.UsePreviousValue=true&Parameters.member.4.ParameterKey=Autoscale&Parameters.member.4.UsePreviousValue=true&Parameters.member.5.ParameterKey=ClientId&Parameters.member.5.UsePreviousValue=true&Parameters.member.6.ParameterKey=ContainerDisk&Parameters.member.6.UsePreviousValue=true&Parameters.member.7.ParameterKey=Development&Parameters.member.7.UsePreviousValue=true&Parameters.member.8.ParameterKey=Encryption&Parameters.member.8.UsePreviousValue=true&Parameters.member.9.ParameterKey=ExistingVpc&Parameters.member.9.UsePreviousValue=true&StackName=convox&TemplateURL=https%3A%2F%2Fconvox.s3.amazonaws.com%2Frelease%2F20160820033210%2Fformation.json&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<UpdateStackResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
				<UpdateStackResult>
					<StackId>arn:aws:cloudformation:us-east-1:901416387788:stack/convox/9a10bbe0-51d5-11e5-b85a-5001dc3ed8d2</StackId>
				</UpdateStackResult>
				<ResponseMetadata>
					<RequestId>b9b4b068-3a41-11e5-94eb-example</RequestId>
				</ResponseMetadata>
			</UpdateStackResponse>
		`,
	},
}

var cycleSystemUpdateStackNewParameter = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=UpdateStack&Capabilities.member.1=CAPABILITY_IAM&Parameters.member.1.ParameterKey=Ami&Parameters.member.1.UsePreviousValue=true&Parameters.member.2.ParameterKey=InstanceCount&Parameters.member.2.ParameterValue=5&Parameters.member.3.ParameterKey=InstanceType&Parameters.member.3.ParameterValue=t2.small&Parameters.member.4.ParameterKey=Version&Parameters.member.4.ParameterValue=20160820033210&StackName=convox&TemplateURL=https%3A%2F%2Fconvox.s3.amazonaws.com%2Frelease%2F20160820033210%2Fformation.json&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<UpdateStackResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
				<UpdateStackResult>
					<StackId>arn:aws:cloudformation:us-east-1:901416387788:stack/convox/9a10bbe0-51d5-11e5-b85a-5001dc3ed8d2</StackId>
				</UpdateStackResult>
				<ResponseMetadata>
					<RequestId>b9b4b068-3a41-11e5-94eb-example</RequestId>
				</ResponseMetadata>
			</UpdateStackResponse>
		`,
	},
}
