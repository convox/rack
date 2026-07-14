package aws_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/pkg/test/awsutil"
	"github.com/convox/rack/provider/aws"
	"github.com/stretchr/testify/assert"
)

func stubAwsProviderCounting(t *testing.T, calls *int32, cycles ...awsutil.Cycle) *aws.Provider {
	handler := awsutil.NewHandler(cycles)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(calls, 1)
		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(s.Close)

	t.Setenv("AWS_ACCESS_KEY_ID", "test-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")

	return &aws.Provider{
		Region:         "us-test-1",
		Endpoint:       s.URL,
		Rack:           "convox",
		DynamoReleases: "convox-releases",
		SettingsBucket: "convox-settings",
		SkipCache:      true,
	}
}

func cycleDescribeStacksVersioned(version, amiParams string) awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/",
			Body:       `Action=DescribeStacks&StackName=convox&Version=2010-05-15`,
		},
		Response: awsutil.Response{
			StatusCode: 200,
			Body: fmt.Sprintf(`<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
				<DescribeStacksResult>
					<Stacks>
						<member>
							<StackId>arn:aws:cloudformation:us-east-1:778743527532:stack/convox/eb743e00-7d8e-11e5-8280-50ba0727c06e</StackId>
							<StackName>convox</StackName>
							<StackStatus>UPDATE_IN_PROGRESS</StackStatus>
							<CreationTime>2015-10-28T16:14:09.590Z</CreationTime>
							<Parameters>
								<member><ParameterKey>Version</ParameterKey><ParameterValue>%s</ParameterValue></member>
								<member><ParameterKey>InstanceCount</ParameterKey><ParameterValue>3</ParameterValue></member>
								<member><ParameterKey>InstanceType</ParameterKey><ParameterValue>t2.small</ParameterValue></member>
								%s
							</Parameters>
						</member>
					</Stacks>
				</DescribeStacksResult>
			</DescribeStacksResponse>`, version, amiParams),
		},
	}
}

const retiredAmiStackParams = `<member><ParameterKey>DefaultAmi</ParameterKey><ParameterValue>/aws/service/ecs/optimized-ami/amazon-linux-2/recommended/image_id</ParameterValue></member>
	<member><ParameterKey>DefaultAmiArm</ParameterKey><ParameterValue>/aws/service/ecs/optimized-ami/amazon-linux-2/arm64/recommended/image_id</ParameterValue></member>`

const migratedAmiStackParams = `<member><ParameterKey>DefaultAmi</ParameterKey><ParameterValue>/aws/service/ecs/optimized-ami/amazon-linux-2023/recommended/image_id</ParameterValue></member>
	<member><ParameterKey>DefaultAmiArm</ParameterKey><ParameterValue>/aws/service/ecs/optimized-ami/amazon-linux-2023/arm64/recommended/image_id</ParameterValue></member>`

const pinnedRetiredAmiStackParams = `<member><ParameterKey>Ami</ParameterKey><ParameterValue>ami-0123456789abcdef0</ParameterValue></member>
	<member><ParameterKey>DefaultAmi</ParameterKey><ParameterValue>/aws/service/ecs/optimized-ami/amazon-linux-2/recommended/image_id</ParameterValue></member>
	<member><ParameterKey>DefaultAmiArm</ParameterKey><ParameterValue>/aws/service/ecs/optimized-ami/amazon-linux-2/arm64/recommended/image_id</ParameterValue></member>`

func TestSystemUpdateSameVersionRunsAmiMigration(t *testing.T) {
	var calls int32

	provider := stubAwsProviderCounting(t, &calls,
		cycleDescribeStacksVersioned("20260530110127", retiredAmiStackParams),
		cycleDescribeStacksVersioned("20260530110127", retiredAmiStackParams),
		cycleSameVersionReleasePutItem,
		cycleDescribeStacksVersioned("20260530110127", retiredAmiStackParams),
		cycleSystemListStackResources,
		cycleSystemTemplatePut,
		cycleSameVersionUpdateStack,
		cycleSameVersionNotificationPublish,
	)

	err := provider.SystemUpdate(structs.SystemUpdateOptions{
		Version: options.String("20260530110127"),
	})

	assert.NoError(t, err)
	assert.Equal(t, int32(8), atomic.LoadInt32(&calls))
}

func TestSystemUpdateSameVersionNoopWhenMigrated(t *testing.T) {
	var calls int32

	provider := stubAwsProviderCounting(t, &calls,
		cycleDescribeStacksVersioned("20260530110127", migratedAmiStackParams),
		cycleDescribeStacksVersioned("20260530110127", migratedAmiStackParams),
	)

	err := provider.SystemUpdate(structs.SystemUpdateOptions{
		Version: options.String("20260530110127"),
	})

	assert.NoError(t, err)
	assert.Equal(t, int32(2), atomic.LoadInt32(&calls))
}

func TestSystemUpdateSameVersionPinnedAmiNoUpdatesTolerated(t *testing.T) {
	var calls int32

	provider := stubAwsProviderCounting(t, &calls,
		cycleDescribeStacksVersioned("20260530110127", pinnedRetiredAmiStackParams),
		cycleDescribeStacksVersioned("20260530110127", pinnedRetiredAmiStackParams),
		cycleSameVersionReleasePutItem,
		cycleDescribeStacksVersioned("20260530110127", pinnedRetiredAmiStackParams),
		cycleSystemListStackResources,
		cycleSystemTemplatePut,
		cyclePinnedAmiUpdateStackNoUpdates,
	)

	err := provider.SystemUpdate(structs.SystemUpdateOptions{
		Version: options.String("20260530110127"),
	})

	assert.NoError(t, err)
	assert.Equal(t, int32(7), atomic.LoadInt32(&calls))
}

func TestSystemUpdateSameVersionRealUpdateErrorPropagates(t *testing.T) {
	var calls int32

	provider := stubAwsProviderCounting(t, &calls,
		cycleDescribeStacksVersioned("20260530110127", retiredAmiStackParams),
		cycleDescribeStacksVersioned("20260530110127", retiredAmiStackParams),
		cycleSameVersionReleasePutItem,
		cycleDescribeStacksVersioned("20260530110127", retiredAmiStackParams),
		cycleSystemListStackResources,
		cycleSystemTemplatePut,
		cycleSameVersionUpdateStackInProgress,
	)

	err := provider.SystemUpdate(structs.SystemUpdateOptions{
		Version: options.String("20260530110127"),
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "UPDATE_IN_PROGRESS")
	assert.Equal(t, int32(7), atomic.LoadInt32(&calls))
}

var cycleSameVersionReleasePutItem = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.PutItem",
		Body:       `{"Item":{"app":{"S":"convox"},"created":{"S":"00010101.000000.000000000"},"id":{"S":"20260530110127"}},"TableName":"convox-releases"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{}`,
	},
}

var cycleSameVersionUpdateStack = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=UpdateStack&Capabilities.member.1=CAPABILITY_IAM&NotificationARNs.member.1=&Parameters.member.1.ParameterKey=AvailabilityZones&Parameters.member.1.ParameterValue=&Parameters.member.2.ParameterKey=DefaultAmi&Parameters.member.2.ParameterValue=%2Faws%2Fservice%2Fecs%2Foptimized-ami%2Famazon-linux-2023%2Frecommended%2Fimage_id&Parameters.member.3.ParameterKey=DefaultAmiArm&Parameters.member.3.ParameterValue=%2Faws%2Fservice%2Fecs%2Foptimized-ami%2Famazon-linux-2023%2Farm64%2Frecommended%2Fimage_id&Parameters.member.4.ParameterKey=InstanceCount&Parameters.member.4.UsePreviousValue=true&Parameters.member.5.ParameterKey=InstanceType&Parameters.member.5.UsePreviousValue=true&Parameters.member.6.ParameterKey=Version&Parameters.member.6.ParameterValue=20260530110127&StackName=convox&Tags.member.1.Key=System&Tags.member.1.Value=convox&Tags.member.2.Key=Type&Tags.member.2.Value=rack&TemplateURL=https%3A%2F%2Fs3.us-test-1.amazonaws.com%2Fconvox-settings%2Ftest-key&Version=2010-05-15`,
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

var cycleSameVersionUpdateStackInProgress = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=UpdateStack&Capabilities.member.1=CAPABILITY_IAM&NotificationARNs.member.1=&Parameters.member.1.ParameterKey=AvailabilityZones&Parameters.member.1.ParameterValue=&Parameters.member.2.ParameterKey=DefaultAmi&Parameters.member.2.ParameterValue=%2Faws%2Fservice%2Fecs%2Foptimized-ami%2Famazon-linux-2023%2Frecommended%2Fimage_id&Parameters.member.3.ParameterKey=DefaultAmiArm&Parameters.member.3.ParameterValue=%2Faws%2Fservice%2Fecs%2Foptimized-ami%2Famazon-linux-2023%2Farm64%2Frecommended%2Fimage_id&Parameters.member.4.ParameterKey=InstanceCount&Parameters.member.4.UsePreviousValue=true&Parameters.member.5.ParameterKey=InstanceType&Parameters.member.5.UsePreviousValue=true&Parameters.member.6.ParameterKey=Version&Parameters.member.6.ParameterValue=20260530110127&StackName=convox&Tags.member.1.Key=System&Tags.member.1.Value=convox&Tags.member.2.Key=Type&Tags.member.2.Value=rack&TemplateURL=https%3A%2F%2Fs3.us-test-1.amazonaws.com%2Fconvox-settings%2Ftest-key&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 400,
		Body: `
			<ErrorResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
				<Error>
					<Type>Sender</Type>
					<Code>ValidationError</Code>
					<Message>Stack:arn:aws:cloudformation:us-east-1:778743527532:stack/convox/eb743e00 is in UPDATE_IN_PROGRESS state and can not be updated.</Message>
				</Error>
				<RequestId>b9b4b068-3a41-11e5-94eb-example</RequestId>
			</ErrorResponse>
		`,
	},
}

var cyclePinnedAmiUpdateStackNoUpdates = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=UpdateStack&Capabilities.member.1=CAPABILITY_IAM&NotificationARNs.member.1=&Parameters.member.1.ParameterKey=Ami&Parameters.member.1.UsePreviousValue=true&Parameters.member.2.ParameterKey=AvailabilityZones&Parameters.member.2.ParameterValue=&Parameters.member.3.ParameterKey=DefaultAmi&Parameters.member.3.ParameterValue=%2Faws%2Fservice%2Fecs%2Foptimized-ami%2Famazon-linux-2023%2Frecommended%2Fimage_id&Parameters.member.4.ParameterKey=DefaultAmiArm&Parameters.member.4.ParameterValue=%2Faws%2Fservice%2Fecs%2Foptimized-ami%2Famazon-linux-2023%2Farm64%2Frecommended%2Fimage_id&Parameters.member.5.ParameterKey=InstanceCount&Parameters.member.5.UsePreviousValue=true&Parameters.member.6.ParameterKey=InstanceType&Parameters.member.6.UsePreviousValue=true&Parameters.member.7.ParameterKey=Version&Parameters.member.7.ParameterValue=20260530110127&StackName=convox&Tags.member.1.Key=System&Tags.member.1.Value=convox&Tags.member.2.Key=Type&Tags.member.2.Value=rack&TemplateURL=https%3A%2F%2Fs3.us-test-1.amazonaws.com%2Fconvox-settings%2Ftest-key&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 400,
		Body: `
			<ErrorResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
				<Error>
					<Type>Sender</Type>
					<Code>ValidationError</Code>
					<Message>No updates are to be performed.</Message>
				</Error>
				<RequestId>b9b4b068-3a41-11e5-94eb-example</RequestId>
			</ErrorResponse>
		`,
	},
}

var cycleSameVersionNotificationPublish = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=Publish&Message=%7B%22action%22%3A%22rack%3Aupdate%22%2C%22data%22%3A%7B%22rack%22%3A%22convox%22%2C%22version%22%3A%2220260530110127%22%7D%2C%22status%22%3A%22success%22%2C%22timestamp%22%3A%220001-01-01T00%3A00%3A00Z%22%7D&Subject=rack%3Aupdate&TargetArn=&Version=2010-03-31`,
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

func TestSystemUpdateMigratesRetiredAmiPaths(t *testing.T) {
	provider := StubAwsProvider(
		cycleSystemDescribeStacksRetiredAmi,
		cycleSystemUpdateStackMigratedAmi,
		cycleSystemUpdateParamsNotificationPublish,
	)
	defer provider.Close()

	err := provider.SystemUpdate(structs.SystemUpdateOptions{
		Parameters: map[string]string{"ApiCpu": "256"},
	})

	assert.NoError(t, err)
}

func TestSystemUpdateExplicitAmiParamWinsOverMigration(t *testing.T) {
	provider := StubAwsProvider(
		cycleSystemDescribeStacksRetiredAmi,
		cycleSystemUpdateStackExplicitAmi,
		cycleSystemUpdateParamsNotificationPublish,
	)
	defer provider.Close()

	err := provider.SystemUpdate(structs.SystemUpdateOptions{
		Parameters: map[string]string{"DefaultAmi": "/custom/ami/path"},
	})

	assert.NoError(t, err)
}

var cycleSystemDescribeStacksRetiredAmi = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=DescribeStacks&StackName=convox&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
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
							<member>
								<ParameterKey>ApiCpu</ParameterKey>
								<ParameterValue>128</ParameterValue>
							</member>
							<member>
								<ParameterKey>DefaultAmi</ParameterKey>
								<ParameterValue>/aws/service/ecs/optimized-ami/amazon-linux-2/recommended/image_id</ParameterValue>
							</member>
							<member>
								<ParameterKey>DefaultAmiArm</ParameterKey>
								<ParameterValue>/aws/service/ecs/optimized-ami/amazon-linux-2/arm64/recommended/image_id</ParameterValue>
							</member>
							<member>
								<ParameterKey>InstanceType</ParameterKey>
								<ParameterValue>t2.small</ParameterValue>
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

var cycleSystemUpdateStackMigratedAmi = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=UpdateStack&Capabilities.member.1=CAPABILITY_IAM&NotificationARNs.member.1=&Parameters.member.1.ParameterKey=Ami&Parameters.member.1.UsePreviousValue=true&Parameters.member.2.ParameterKey=ApiCpu&Parameters.member.2.ParameterValue=256&Parameters.member.3.ParameterKey=DefaultAmi&Parameters.member.3.ParameterValue=%2Faws%2Fservice%2Fecs%2Foptimized-ami%2Famazon-linux-2023%2Frecommended%2Fimage_id&Parameters.member.4.ParameterKey=DefaultAmiArm&Parameters.member.4.ParameterValue=%2Faws%2Fservice%2Fecs%2Foptimized-ami%2Famazon-linux-2023%2Farm64%2Frecommended%2Fimage_id&Parameters.member.5.ParameterKey=InstanceType&Parameters.member.5.UsePreviousValue=true&StackName=convox&Tags.member.1.Key=System&Tags.member.1.Value=convox&Tags.member.2.Key=Type&Tags.member.2.Value=rack&UsePreviousTemplate=true&Version=2010-05-15`,
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

var cycleSystemUpdateStackExplicitAmi = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=UpdateStack&Capabilities.member.1=CAPABILITY_IAM&NotificationARNs.member.1=&Parameters.member.1.ParameterKey=Ami&Parameters.member.1.UsePreviousValue=true&Parameters.member.2.ParameterKey=ApiCpu&Parameters.member.2.UsePreviousValue=true&Parameters.member.3.ParameterKey=DefaultAmi&Parameters.member.3.ParameterValue=%2Fcustom%2Fami%2Fpath&Parameters.member.4.ParameterKey=DefaultAmiArm&Parameters.member.4.ParameterValue=%2Faws%2Fservice%2Fecs%2Foptimized-ami%2Famazon-linux-2023%2Farm64%2Frecommended%2Fimage_id&Parameters.member.5.ParameterKey=InstanceType&Parameters.member.5.UsePreviousValue=true&StackName=convox&Tags.member.1.Key=System&Tags.member.1.Value=convox&Tags.member.2.Key=Type&Tags.member.2.Value=rack&UsePreviousTemplate=true&Version=2010-05-15`,
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

var cycleSystemUpdateParamsNotificationPublish = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=Publish&Message=%7B%22action%22%3A%22rack%3Aupdate%22%2C%22data%22%3A%7B%22rack%22%3A%22convox%22%7D%2C%22status%22%3A%22success%22%2C%22timestamp%22%3A%220001-01-01T00%3A00%3A00Z%22%7D&Subject=rack%3Aupdate&TargetArn=&Version=2010-03-31`,
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
