package aws_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/pkg/test/awsutil"
	"github.com/convox/rack/provider/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	awssdk "github.com/aws/aws-sdk-go/aws"
	mockaws "github.com/convox/rack/pkg/mock/aws"
)

func TestSystemGet(t *testing.T) {
	provider := StubAwsProvider(
		cycleSystemDescribeStacks,
		cycleListRackStackResources,
		cycleDescribeAutoscalingGroups,
		cycleECSListServices,
		cycleECSDescribeServices,
	)
	defer provider.Close()

	s, err := provider.SystemGet()

	assert.NoError(t, err)
	assert.EqualValues(t, &structs.System{
		Count:      3,
		Name:       "convox",
		Provider:   "aws",
		Region:     "us-test-1",
		Status:     "running",
		Type:       "t2.small",
		Version:    "dev",
		Outputs:    map[string]string{},
		Parameters: map[string]string{"Autoscale": "No", "SubnetPrivate2CIDR": "10.0.6.0/24", "Subnet0CIDR": "10.0.1.0/24", "Encryption": "Yes", "Development": "Yes", "Private": "No", "InstanceUpdateBatchSize": "1", "InstanceRunCommand": "", "ExistingVpc": "", "PrivateApi": "No", "ContainerDisk": "10", "Ami": "", "VolumeSize": "50", "Tenancy": "default", "Version": "dev", "VPCCIDR": "10.0.0.0/16", "Subnet2CIDR": "10.0.3.0/24", "InstanceType": "t2.small", "Password": "****", "Key": "convox-keypair-4415", "ApiCpu": "128", "SwapSize": "5", "ApiMemory": "128", "SubnetPrivate0CIDR": "10.0.4.0/24", "InstanceCount": "3", "InstanceBootCommand": "", "Internal": "No", "Subnet1CIDR": "10.0.2.0/24", "ClientId": "nmert38iwdsrj362jdf", "SubnetPrivate1CIDR": "10.0.5.0/24"},
	}, s)
}

func TestSystemGetConverging(t *testing.T) {
	provider := StubAwsProvider(
		cycleSystemDescribeStacks,
		cycleListRackStackResources,
		cycleDescribeAutoscalingGroupsInstanceTerminating,
	)
	defer provider.Close()

	s, err := provider.SystemGet()

	assert.NoError(t, err)
	assert.EqualValues(t, &structs.System{
		Count:      3,
		Name:       "convox",
		Provider:   "aws",
		Region:     "us-test-1",
		Status:     "converging",
		Type:       "t2.small",
		Version:    "dev",
		Outputs:    map[string]string{},
		Parameters: map[string]string{"Autoscale": "No", "SubnetPrivate2CIDR": "10.0.6.0/24", "Subnet0CIDR": "10.0.1.0/24", "Encryption": "Yes", "Development": "Yes", "Private": "No", "InstanceUpdateBatchSize": "1", "InstanceRunCommand": "", "ExistingVpc": "", "PrivateApi": "No", "ContainerDisk": "10", "Ami": "", "VolumeSize": "50", "Tenancy": "default", "Version": "dev", "VPCCIDR": "10.0.0.0/16", "Subnet2CIDR": "10.0.3.0/24", "InstanceType": "t2.small", "Password": "****", "Key": "convox-keypair-4415", "ApiCpu": "128", "SwapSize": "5", "ApiMemory": "128", "SubnetPrivate0CIDR": "10.0.4.0/24", "InstanceCount": "3", "InstanceBootCommand": "", "Internal": "No", "Subnet1CIDR": "10.0.2.0/24", "ClientId": "nmert38iwdsrj362jdf", "SubnetPrivate1CIDR": "10.0.5.0/24"},
	}, s)
}

func TestSystemGetBadStack(t *testing.T) {
	provider := StubAwsProvider(
		cycleDescribeStacksNotFound("convox"),
	)
	defer provider.Close()

	r, err := provider.SystemGet()

	assert.Nil(t, r)
	assert.EqualError(t, err, "convox not found")
}

func TestSystemMetrics(t *testing.T) {
	testProvider(func(p *aws.Provider) {
		m := &mockaws.CloudWatchAPI{}

		p.AsgSpot = "asg-spot"
		p.AsgStandard = "asg-standard"
		p.Cluster = "cluster1"

		metrics := []struct {
			Name       string
			Namespace  string
			Dimensions []string
		}{
			{Namespace: "AWS/ECS", Name: "CPUReservation", Dimensions: []string{"ClusterName:cluster1"}},
			{Namespace: "AWS/ECS", Name: "MemoryReservation", Dimensions: []string{"ClusterName:cluster1"}},
			{Namespace: "AWS/ECS", Name: "CPUUtilization", Dimensions: []string{"ClusterName:cluster1"}},
			{Namespace: "AWS/ECS", Name: "MemoryUtilization", Dimensions: []string{"ClusterName:cluster1"}},
			{Namespace: "AWS/EC2", Name: "CPUUtilization", Dimensions: []string{"AutoScalingGroupName:asg-spot"}},
			{Namespace: "AWS/EC2", Name: "CPUUtilization", Dimensions: []string{"AutoScalingGroupName:asg-standard"}},
		}

		for _, metric := range metrics {
			input := &cloudwatch.GetMetricStatisticsInput{
				EndTime:    awssdk.Time(time.Date(2018, 10, 1, 3, 4, 5, 0, time.UTC)),
				MetricName: awssdk.String(metric.Name),
				Namespace:  awssdk.String(metric.Namespace),
				Period:     awssdk.Int64(300),
				StartTime:  awssdk.Time(time.Date(2018, 9, 1, 2, 3, 4, 0, time.UTC)),
				Statistics: []*string{awssdk.String("Average"), awssdk.String("Minimum"), awssdk.String("Maximum")},
			}

			for _, d := range metric.Dimensions {
				parts := strings.Split(d, ":")
				input.Dimensions = append(input.Dimensions, &cloudwatch.Dimension{Name: awssdk.String(parts[0]), Value: awssdk.String(parts[1])})
			}

			output := &cloudwatch.GetMetricStatisticsOutput{
				Datapoints: []*cloudwatch.Datapoint{
					{
						Timestamp: awssdk.Time(time.Date(2018, 9, 1, 2, 3, 4, 0, time.UTC)),
						Average:   awssdk.Float64(2.12345),
						Minimum:   awssdk.Float64(1.12345),
						Maximum:   awssdk.Float64(3.12345),
					},
				},
			}

			m.On("GetMetricStatistics", input).Return(output, nil)
		}

		p.CloudWatch = m

		opts := structs.MetricsOptions{
			Start:  options.Time(time.Date(2018, 9, 1, 2, 3, 4, 0, time.UTC)),
			End:    options.Time(time.Date(2018, 10, 1, 3, 4, 5, 0, time.UTC)),
			Period: options.Int64(300),
		}

		m1, err := p.SystemMetrics(opts)
		require.NoError(t, err)

		m2 := structs.Metrics{
			{Name: "cluster:cpu:reservation", Values: structs.MetricValues{{Time: time.Date(2018, 9, 1, 2, 3, 4, 0, time.UTC), Average: 2.12, Maximum: 3.12, Minimum: 1.12}}},
			{Name: "cluster:cpu:utilization", Values: structs.MetricValues{{Time: time.Date(2018, 9, 1, 2, 3, 4, 0, time.UTC), Average: 2.12, Maximum: 3.12, Minimum: 1.12}}},
			{Name: "cluster:mem:reservation", Values: structs.MetricValues{{Time: time.Date(2018, 9, 1, 2, 3, 4, 0, time.UTC), Average: 2.12, Maximum: 3.12, Minimum: 1.12}}},
			{Name: "cluster:mem:utilization", Values: structs.MetricValues{{Time: time.Date(2018, 9, 1, 2, 3, 4, 0, time.UTC), Average: 2.12, Maximum: 3.12, Minimum: 1.12}}},
			{Name: "instances:standard:cpu", Values: structs.MetricValues{{Time: time.Date(2018, 9, 1, 2, 3, 4, 0, time.UTC), Average: 2.12, Maximum: 3.12, Minimum: 1.12}}},
			{Name: "instances:spot:cpu", Values: structs.MetricValues{{Time: time.Date(2018, 9, 1, 2, 3, 4, 0, time.UTC), Average: 2.12, Maximum: 3.12, Minimum: 1.12}}},
		}

		require.Equal(t, m2, m1)
	})
}

func TestSystemReleases(t *testing.T) {
	provider := StubAwsProvider(
		cycleSystemReleaseList,
	)
	defer provider.Close()

	r, err := provider.SystemReleases()

	assert.NoError(t, err)

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

func TestSystemUpdate(t *testing.T) {
	provider := StubAwsProvider(
		cycleSystemReleasePutItem,
		cycleSystemDescribeStacks,
		cycleSystemListStackResources,
		cycleSystemTemplatePut,
		cycleSystemUpdateStack,
		cycleSystemUpdateNotificationPublish,
	)
	defer provider.Close()

	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	os.Chdir("../..")

	err := provider.SystemUpdate(structs.SystemUpdateOptions{
		Count:   options.Int(5),
		Type:    options.String("t2.small"),
		Version: options.String("dev"),
	})

	assert.NoError(t, err)
}

func TestSystemUpdateNewParameter(t *testing.T) {
	provider := StubAwsProvider(
		cycleSystemReleasePutItem,
		cycleSystemDescribeStacksMissingParameters,
		cycleSystemListStackResources,
		cycleSystemTemplatePut,
		cycleSystemUpdateStackNewParameter,
		cycleSystemUpdateNotificationPublish,
	)
	defer provider.Close()

	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	os.Chdir("../..")

	err := provider.SystemUpdate(structs.SystemUpdateOptions{
		Count:   options.Int(5),
		Type:    options.String("t2.small"),
		Version: options.String("dev"),
	})

	assert.NoError(t, err)
}

func TestSystemProcessesList(t *testing.T) {
	provider := StubAwsProvider(
		cycleSystemListStackResources,
		cycleSystemDescribeStacks,
		cycleSystemListTasks,
		cycleSystemDescribeTasks,
		cycleSystemDescribeTaskDefinition,
		cycleSystemDescribeContainerInstances,
		cycleSystemDescribeRackInstances,
		cycleSystemDescribeTaskDefinition2,
		cycleSystemDescribeContainerInstances,
	)
	defer provider.Close()

	_, err := provider.SystemProcesses(structs.SystemProcessesOptions{})

	assert.NoError(t, err)
}

func TestSystemProcessesListAll(t *testing.T) {
	provider := StubAwsProvider(
		cycleSystemListTasksAll,
		cycleSystemDescribeTasksAll,
		cycleSystemDescribeTaskDefinition,
		cycleSystemDescribeContainerInstances,
		cycleSystemDescribeRackInstances,
		cycleSystemDescribeTaskDefinition2,
		cycleSystemDescribeContainerInstances,
	)
	defer provider.Close()

	_, err := provider.SystemProcesses(structs.SystemProcessesOptions{
		All: options.Bool(true),
	})

	assert.NoError(t, err)
}

var cycleSystemDescribeStacks = awsutil.Cycle{
	awsutil.Request{"POST", "/", "", `Action=DescribeStacks&StackName=convox&Version=2010-05-15`},
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
	awsutil.Request{"POST", "/", "", `Action=DescribeStacks&StackName=convox&Version=2010-05-15`},
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

var cycleSystemReleasePutItem = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.PutItem",
		Body:       `{"Item":{"app":{"S":"convox"},"created":{"S":"00010101.000000.000000000"},"id":{"S":"dev"}},"TableName":"convox-releases"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{}`,
	},
}

var cycleSystemUpdateNotificationPublish = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=Publish&Message=%7B%22action%22%3A%22rack%3Aupdate%22%2C%22data%22%3A%7B%22count%22%3A%225%22%2C%22rack%22%3A%22convox%22%2C%22type%22%3A%22t2.small%22%2C%22version%22%3A%22dev%22%7D%2C%22status%22%3A%22success%22%2C%22timestamp%22%3A%220001-01-01T00%3A00%3A00Z%22%7D&Subject=rack%3Aupdate&TargetArn=&Version=2010-03-31`,
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
		Body:       `Action=UpdateStack&Capabilities.member.1=CAPABILITY_IAM&NotificationARNs.member.1=&Parameters.member.1.ParameterKey=Ami&Parameters.member.1.UsePreviousValue=true&Parameters.member.10.ParameterKey=InstanceRunCommand&Parameters.member.10.UsePreviousValue=true&Parameters.member.11.ParameterKey=InstanceType&Parameters.member.11.ParameterValue=t2.small&Parameters.member.12.ParameterKey=InstanceUpdateBatchSize&Parameters.member.12.UsePreviousValue=true&Parameters.member.13.ParameterKey=Internal&Parameters.member.13.UsePreviousValue=true&Parameters.member.14.ParameterKey=Key&Parameters.member.14.UsePreviousValue=true&Parameters.member.15.ParameterKey=Password&Parameters.member.15.UsePreviousValue=true&Parameters.member.16.ParameterKey=Private&Parameters.member.16.UsePreviousValue=true&Parameters.member.17.ParameterKey=PrivateApi&Parameters.member.17.UsePreviousValue=true&Parameters.member.18.ParameterKey=Subnet0CIDR&Parameters.member.18.UsePreviousValue=true&Parameters.member.19.ParameterKey=Subnet1CIDR&Parameters.member.19.UsePreviousValue=true&Parameters.member.2.ParameterKey=ApiMemory&Parameters.member.2.UsePreviousValue=true&Parameters.member.20.ParameterKey=Subnet2CIDR&Parameters.member.20.UsePreviousValue=true&Parameters.member.21.ParameterKey=SubnetPrivate0CIDR&Parameters.member.21.UsePreviousValue=true&Parameters.member.22.ParameterKey=SubnetPrivate1CIDR&Parameters.member.22.UsePreviousValue=true&Parameters.member.23.ParameterKey=SubnetPrivate2CIDR&Parameters.member.23.UsePreviousValue=true&Parameters.member.24.ParameterKey=SwapSize&Parameters.member.24.UsePreviousValue=true&Parameters.member.25.ParameterKey=Tenancy&Parameters.member.25.UsePreviousValue=true&Parameters.member.26.ParameterKey=VPCCIDR&Parameters.member.26.UsePreviousValue=true&Parameters.member.27.ParameterKey=Version&Parameters.member.27.UsePreviousValue=true&Parameters.member.28.ParameterKey=VolumeSize&Parameters.member.28.UsePreviousValue=true&Parameters.member.3.ParameterKey=Autoscale&Parameters.member.3.UsePreviousValue=true&Parameters.member.4.ParameterKey=ClientId&Parameters.member.4.UsePreviousValue=true&Parameters.member.5.ParameterKey=Development&Parameters.member.5.UsePreviousValue=true&Parameters.member.6.ParameterKey=Encryption&Parameters.member.6.UsePreviousValue=true&Parameters.member.7.ParameterKey=ExistingVpc&Parameters.member.7.UsePreviousValue=true&Parameters.member.8.ParameterKey=InstanceBootCommand&Parameters.member.8.UsePreviousValue=true&Parameters.member.9.ParameterKey=InstanceCount&Parameters.member.9.ParameterValue=5&StackName=convox&Tags.member.1.Key=System&Tags.member.1.Value=convox&Tags.member.2.Key=Type&Tags.member.2.Value=rack&TemplateURL=https%3A%2F%2Fs3.us-test-1.amazonaws.com%2Fconvox-settings%2Ftest-key&Version=2010-05-15`,
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
		Body:       `Action=UpdateStack&Capabilities.member.1=CAPABILITY_IAM&NotificationARNs.member.1=&Parameters.member.1.ParameterKey=Ami&Parameters.member.1.UsePreviousValue=true&Parameters.member.2.ParameterKey=InstanceCount&Parameters.member.2.ParameterValue=5&Parameters.member.3.ParameterKey=InstanceType&Parameters.member.3.ParameterValue=t2.small&StackName=convox&Tags.member.1.Key=System&Tags.member.1.Value=convox&Tags.member.2.Key=Type&Tags.member.2.Value=rack&TemplateURL=https%3A%2F%2Fs3.us-test-1.amazonaws.com%2Fconvox-settings%2Ftest-key&Version=2010-05-15`,
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

var cycleListRackStackResources = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "",
		Body:       `Action=ListStackResources&StackName=convox&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<ListStackResourcesResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
  <ListStackResourcesResult>
    <StackResourceSummaries>
    <member>
      <PhysicalResourceId>convox-Instances-1UEIK1IO8W9K3</PhysicalResourceId>
      <ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
      <StackId>arn:aws:cloudformation:us-east-1:990037048036:stack/convox/b8423690-917d-1fe6-8737-50dseaf92cd2</StackId>
      <StackName>convox</StackName>
      <LogicalResourceId>Instances</LogicalResourceId>
      <Timestamp>2016-10-22T02:53:23.817Z</Timestamp>
      <ResourceType>AWS::AutoScaling::AutoScalingGroup</ResourceType>
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

var cycleDescribeRackStackResources = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "",
		Body:       `Action=DescribeStackResources&StackName=convox&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<DescribeStackResourcesResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
  <DescribeStackResourcesResult>
    <StackResources>
    <member>
      <PhysicalResourceId>convox-Instances-1UEIK1IO8W9K3</PhysicalResourceId>
      <ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
      <StackId>arn:aws:cloudformation:us-east-1:990037048036:stack/convox/b8423690-917d-1fe6-8737-50dseaf92cd2</StackId>
      <StackName>convox</StackName>
      <LogicalResourceId>Instances</LogicalResourceId>
      <Timestamp>2016-10-22T02:53:23.817Z</Timestamp>
      <ResourceType>AWS::AutoScaling::AutoScalingGroup</ResourceType>
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

var cycleDescribeAutoscalingGroups = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "",
		Body:       `Action=DescribeAutoScalingGroups&AutoScalingGroupNames.member.1=convox-Instances-1UEIK1IO8W9K3&Version=2011-01-01`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<DescribeAutoScalingGroupsResponse xmlns="http://autoscaling.amazonaws.com/doc/2011-01-01/">
    <DescribeAutoScalingGroupsResult>
        <AutoScalingGroups>
            <member>
                <Instances>
                    <member>
                        <LaunchConfigurationName>convox-LaunchConfiguration-58846ZXWGVYY</LaunchConfigurationName>
                        <LifecycleState>InService</LifecycleState>
                        <InstanceId>i-02fbf6732eac0d195</InstanceId>
                        <HealthStatus>Healthy</HealthStatus>
                        <ProtectedFromScaleIn>false</ProtectedFromScaleIn>
                        <AvailabilityZone>us-east-1b</AvailabilityZone>
                    </member>
                    <member>
                        <LaunchConfigurationName>convox-LaunchConfiguration-58846ZXWGVYY</LaunchConfigurationName>
                        <LifecycleState>InService</LifecycleState>
                        <InstanceId>i-047a745d1d8016000</InstanceId>
                        <HealthStatus>Healthy</HealthStatus>
                        <ProtectedFromScaleIn>false</ProtectedFromScaleIn>
                        <AvailabilityZone>us-east-1a</AvailabilityZone>
                    </member>
                    <member>
                        <LaunchConfigurationName>convox-LaunchConfiguration-58846ZXWGVYY</LaunchConfigurationName>
                        <LifecycleState>InService</LifecycleState>
                        <InstanceId>i-0b0fa380591282dd0</InstanceId>
                        <HealthStatus>Healthy</HealthStatus>
                        <ProtectedFromScaleIn>false</ProtectedFromScaleIn>
                        <AvailabilityZone>us-east-1b</AvailabilityZone>
                    </member>
                    <member>
                        <LaunchConfigurationName>convox-LaunchConfiguration-58846ZXWGVYY</LaunchConfigurationName>
                        <LifecycleState>InService</LifecycleState>
                        <InstanceId>i-0b56f635928702c76</InstanceId>
                        <HealthStatus>Healthy</HealthStatus>
                        <ProtectedFromScaleIn>false</ProtectedFromScaleIn>
                        <AvailabilityZone>us-east-1d</AvailabilityZone>
                    </member>
                </Instances>
            </member>
        </AutoScalingGroups>
    </DescribeAutoScalingGroupsResult>
    <ResponseMetadata>
        <RequestId>62487f00-9807-11e6-a11e-1336240b2ac0</RequestId>
    </ResponseMetadata>
</DescribeAutoScalingGroupsResponse>`,
	},
}

var cycleECSListServices = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.ListServices",
		Body:       `{"cluster": "cluster-test", "maxResults": 10}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{"serviceArns": [
        "arn:aws:ecs:us-east-1:111111111111:service/rack-RackMonitor-1JI86RBJGU0M2",
        "arn:aws:ecs:us-east-1:111111111111:service/rack-RackWeb-1W12WRB8CUUW4",
        "arn:aws:ecs:us-east-1:111111111111:service/rack-httpd-ServiceWeb-1PZ7WERU0UPVN"
    ]
}`,
	},
}

var cycleECSDescribeServices = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.DescribeServices",
		Body: `{"cluster": "cluster-test", "services": [
    "arn:aws:ecs:us-east-1:111111111111:service/rack-RackMonitor-1JI86RBJGU0M2",
    "arn:aws:ecs:us-east-1:111111111111:service/rack-RackWeb-1W12WRB8CUUW4",
    "arn:aws:ecs:us-east-1:111111111111:service/rack-httpd-ServiceWeb-1PZ7WERU0UPVN"
  ]}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
    "failures": [],
    "services": [{
        "clusterArn": "arn:aws:ecs:us-east-1:111111111111:cluster/rack-Cluster-DFMP83Z2KMOB",
        "createdAt": 1.479163546803E9,
        "deploymentConfiguration": {
            "maximumPercent": 200,
            "minimumHealthyPercent": 100
        },
        "deployments": [{
            "createdAt": 1.48095261184E9,
            "desiredCount": 1,
            "id": "ecs-svc/9223370555902163967",
            "pendingCount": 0,
            "runningCount": 1,
            "status": "PRIMARY",
            "taskDefinition": "arn:aws:ecs:us-east-1:111111111111:task-definition/rack-monitor:104",
            "updatedAt": 1.48095261184E9
        }],
        "desiredCount": 1,
        "events": [
          {
            "createdAt": 1.480959425817E9,
            "id": "11111111-c7f4-1111-8faa-1111111f8daa",
            "message": "(service rack-RackMonitor-1JI86R98ZU0M2) has reached a steady state."
        }],
        "loadBalancers": [],
        "pendingCount": 0,
        "runningCount": 1,
        "serviceArn": "arn:aws:ecs:us-east-1:111111111111:service/rack-RackMonitor-1JI86R98ZU0M2",
        "serviceName": "rack-RackMonitor-1JI86R98ZU0M2",
        "status": "ACTIVE",
        "taskDefinition": "arn:aws:ecs:us-east-1:111111111111:task-definition/rack-monitor:104"
    }, {
        "clusterArn": "arn:aws:ecs:us-east-1:111111111111:cluster/rack-Cluster-DFMP83Z2KMOB",
        "createdAt": 1.479163547885E9,
        "deploymentConfiguration": {
            "maximumPercent": 200,
            "minimumHealthyPercent": 100
        },
        "deployments": [{
            "createdAt": 1.480952617042E9,
            "desiredCount": 2,
            "id": "ecs-svc/9223370555902158765",
            "pendingCount": 0,
            "runningCount": 2,
            "status": "PRIMARY",
            "taskDefinition": "arn:aws:ecs:us-east-1:111111111111:task-definition/rack-web:101",
            "updatedAt": 1.480952617042E9
        }],
        "desiredCount": 2,
        "events": [{
            "createdAt": 1.480729663517E9,
            "id": "4a111112-ba11-111b-9136-19111116a0e3",
            "message": "(service rack-RackWeb-1W1WQRB8CUUW4) has started 1 tasks: (task foo-bar-49c2f91081)."
        }],
        "loadBalancers": [{
            "containerName": "web",
            "containerPort": 3000,
            "loadBalancerName": "rack"
        }],
        "pendingCount": 0,
        "roleArn": "arn:aws:iam::111111111111:role/convox/rack-ServiceRole-1UHAEF0KU6PQP",
        "runningCount": 2,
        "serviceArn": "arn:aws:ecs:us-east-1:111111111111:service/rack-RackWeb-1W1WQRB8CUUW4",
        "serviceName": "rack-RackWeb-1W1WQRB8CUUW4",
        "status": "ACTIVE",
        "taskDefinition": "arn:aws:ecs:us-east-1:111111111111:task-definition/rack-web:101"
    }]
}`,
	},
}

var cycleDescribeAutoscalingGroupsInstanceTerminating = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "",
		Body:       `Action=DescribeAutoScalingGroups&AutoScalingGroupNames.member.1=convox-Instances-1UEIK1IO8W9K3&Version=2011-01-01`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<DescribeAutoScalingGroupsResponse xmlns="http://autoscaling.amazonaws.com/doc/2011-01-01/">
    <DescribeAutoScalingGroupsResult>
        <AutoScalingGroups>
            <member>
                <Instances>
                    <member>
                        <LaunchConfigurationName>convox-LaunchConfiguration-58846ZXWGVYY</LaunchConfigurationName>
                        <LifecycleState>InService</LifecycleState>
                        <InstanceId>i-02fbf6732eac0d195</InstanceId>
                        <HealthStatus>Healthy</HealthStatus>
                        <ProtectedFromScaleIn>false</ProtectedFromScaleIn>
                        <AvailabilityZone>us-east-1b</AvailabilityZone>
                    </member>
                    <member>
                        <LaunchConfigurationName>convox-LaunchConfiguration-58846ZXWGVYY</LaunchConfigurationName>
                        <LifecycleState>Terminating:Wait</LifecycleState>
                        <InstanceId>i-047a745d1d8016000</InstanceId>
                        <HealthStatus>Healthy</HealthStatus>
                        <ProtectedFromScaleIn>false</ProtectedFromScaleIn>
                        <AvailabilityZone>us-east-1a</AvailabilityZone>
                    </member>
                    <member>
                        <LaunchConfigurationName>convox-LaunchConfiguration-58846ZXWGVYY</LaunchConfigurationName>
                        <LifecycleState>InService</LifecycleState>
                        <InstanceId>i-0b0fa380591282dd0</InstanceId>
                        <HealthStatus>Healthy</HealthStatus>
                        <ProtectedFromScaleIn>false</ProtectedFromScaleIn>
                        <AvailabilityZone>us-east-1b</AvailabilityZone>
                    </member>
                    <member>
                        <LaunchConfigurationName>convox-LaunchConfiguration-58846ZXWGVYY</LaunchConfigurationName>
                        <LifecycleState>InService</LifecycleState>
                        <InstanceId>i-0b56f635928702c76</InstanceId>
                        <HealthStatus>Healthy</HealthStatus>
                        <ProtectedFromScaleIn>false</ProtectedFromScaleIn>
                        <AvailabilityZone>us-east-1d</AvailabilityZone>
                    </member>
                </Instances>
            </member>
        </AutoScalingGroups>
    </DescribeAutoScalingGroupsResult>
    <ResponseMetadata>
        <RequestId>62487f00-9807-11e6-a11e-1336240b2ac0</RequestId>
    </ResponseMetadata>
</DescribeAutoScalingGroupsResponse>		`,
	},
}

func cycleDescribeStacksNotFound(name string) awsutil.Cycle {
	return awsutil.Cycle{
		awsutil.Request{"POST", "/", "", `Action=DescribeStacks&StackName=` + name + `&Version=2010-05-15`},
		awsutil.Response{
			400,
			`<ErrorResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
				<Error>
					<Type>Sender</Type>
					<Code>ValidationError</Code>
					<Message>Stack with id ` + name + ` does not exist</Message>
				</Error>
				<RequestId>bc91dc86-5803-11e5-a24f-85fde26a90fa</RequestId>
			</ErrorResponse>`,
		},
	}
}

var cycleSystemTemplatePut = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "PUT",
		RequestURI: "/convox-httpd-settings-139bidzalmbtu/releases/R23456/env",
		Body:       "ignore",
	},
	Response: awsutil.Response{
		StatusCode: 200,
	},
}

var cycleSystemListStackResources = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "",
		Body:       `Action=ListStackResources&StackName=convox&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<ListStackResourcesResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
				<ListStackResourcesResult>
					<StackResourceSummaries>
						<member>
							<PhysicalResourceId>arn:aws:ecs:us-east-1:778743527532:service/convox-myapp-ServiceDatabase-1I2PTXAZ5ECRD</PhysicalResourceId>
							<ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
							<StackId>arn:aws:cloudformation:us-east-1:778743527532:stack/convox-myapp/5c05e0c0-6e10-11e6-8a4e-50fae98a10d2</StackId>
							<StackName>convox-myapp</StackName>
							<LogicalResourceId>ServiceDatabase</LogicalResourceId>
							<Timestamp>2016-09-10T04:35:11.280Z</Timestamp>
							<ResourceType>AWS::ECS::Service</ResourceType>
						</member>
						<member>
							<PhysicalResourceId>arn:kms::::::</PhysicalResourceId>
							<ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
							<StackId>arn:aws:cloudformation:us-east-1:990037048036:stack/convox/b8423690-917d-1fe6-8737-50dseaf92cd2</StackId>
							<StackName>convox</StackName>
							<LogicalResourceId>EncryptionKey</LogicalResourceId>
							<Timestamp>2016-10-22T02:53:23.817Z</Timestamp>
							<ResourceType>AWS::AutoScaling::AutoScalingGroup</ResourceType>
						</member>
						<member>
							<PhysicalResourceId>settings-bucket</PhysicalResourceId>
							<ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
							<StackId>arn:aws:cloudformation:us-east-1:990037048036:stack/convox/b8423690-917d-1fe6-8737-50dseaf92cd2</StackId>
							<StackName>convox</StackName>
							<LogicalResourceId>Settings</LogicalResourceId>
							<Timestamp>2016-10-22T02:53:23.817Z</Timestamp>
							<ResourceType>AWS::S3::Bucket</ResourceType>
						</member>
					</StackResourceSummaries>
				</ListStackResourcesResult>
				<ResponseMetadata>
					<RequestId>8be86de9-7760-11e6-b2f2-6b253bb2c005</RequestId>
				</ResponseMetadata>
			</ListStackResourcesResponse>
		`,
	},
}

var cycleSystemDescribeStackResources = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "",
		Body:       `Action=DescribeStackResources&StackName=convox&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<DescribeStackResourcesResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
				<DescribeStackResourcesResult>
					<StackResources>
						<member>
							<PhysicalResourceId>arn:aws:ecs:us-east-1:778743527532:service/convox-myapp-ServiceDatabase-1I2PTXAZ5ECRD</PhysicalResourceId>
							<ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
							<StackId>arn:aws:cloudformation:us-east-1:778743527532:stack/convox-myapp/5c05e0c0-6e10-11e6-8a4e-50fae98a10d2</StackId>
							<StackName>convox-myapp</StackName>
							<LogicalResourceId>ServiceDatabase</LogicalResourceId>
							<Timestamp>2016-09-10T04:35:11.280Z</Timestamp>
							<ResourceType>AWS::ECS::Service</ResourceType>
						</member>
						<member>
							<PhysicalResourceId>arn:kms::::::</PhysicalResourceId>
							<ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
							<StackId>arn:aws:cloudformation:us-east-1:990037048036:stack/convox/b8423690-917d-1fe6-8737-50dseaf92cd2</StackId>
							<StackName>convox</StackName>
							<LogicalResourceId>EncryptionKey</LogicalResourceId>
							<Timestamp>2016-10-22T02:53:23.817Z</Timestamp>
							<ResourceType>AWS::AutoScaling::AutoScalingGroup</ResourceType>
						</member>
					</StackResources>
				</DescribeStackResourcesResult>
				<ResponseMetadata>
					<RequestId>8be86de9-7760-11e6-b2f2-6b253bb2c005</RequestId>
				</ResponseMetadata>
			</DescribeStackResourcesResponse>
		`,
	},
}

var cycleSystemListTasks = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.ListTasks",
		Body: `{
			  "cluster": "cluster-test",
				  "serviceName": "arn:aws:ecs:us-east-1:778743527532:service/convox-myapp-ServiceDatabase-1I2PTXAZ5ECRD"
				}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskArns": [
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0846",
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845"
			]
		}`,
	},
}

var cycleSystemListTasksAll = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.ListTasks",
		Body: `{
			  "cluster": "cluster-test"
				}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskArns": [
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845"
			]
		}`,
	},
}

var cycleSystemDescribeTasks = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTasks",
		Body: `{
			"cluster": "cluster-test",
			"tasks": [
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0846",
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845"
			]
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"failures": [],
			"tasks": [
				{
					"taskArn": "arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845",
					"overrides": {
						"containerOverrides": [
							{
								"command": ["sh", "-c", "foo"]
							}
						]
					},
					"lastStatus": "RUNNING",
					"taskDefinitionArn": "arn:aws:ecs:us-east-1:778743527532:task-definition/convox:34",
					"containerInstanceArn": "arn:aws:ecs:us-east-1:778743527532:container-instance/e126c67d-fa95-4b09-8b4a-3723932cd2aa",
					"containers": [
						{
							"name": "web",
							"containerArn": "arn:aws:ecs:us-east-1:778743527532:container/3ab3b8c5-aa5c-4b54-89f8-5f1193aff5f9"
						}
					]
				}
			]
		}`,
	},
}

var cycleSystemDescribeTasksAll = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTasks",
		Body: `{
			"cluster": "cluster-test",
			"tasks": [
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845"
			]
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"failures": [],
			"tasks": [
				{
					"taskArn": "arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845",
					"overrides": {
						"containerOverrides": [
							{
								"command": ["sh", "-c", "foo"]
							}
						]
					},
					"lastStatus": "RUNNING",
					"taskDefinitionArn": "arn:aws:ecs:us-east-1:778743527532:task-definition/convox:34",
					"containerInstanceArn": "arn:aws:ecs:us-east-1:778743527532:container-instance/e126c67d-fa95-4b09-8b4a-3723932cd2aa",
					"containers": [
						{
							"name": "web",
							"containerArn": "arn:aws:ecs:us-east-1:778743527532:container/3ab3b8c5-aa5c-4b54-89f8-5f1193aff5f9"
						}
					]
				}
			]
		}`,
	},
}

var cycleSystemDescribeTaskDefinition = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "POST",
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition",
		Body: `{
			  "taskDefinition": "arn:aws:ecs:us-east-1:778743527532:task-definition/convox:34"
			}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskDefinition": {
				"status": "ACTIVE",
				"family": "convox-myapp-web",
				"requiresAttributes": [
					{
						"name": "com.amazonaws.ecs.capability.ecr-auth"
					}
				],
				"volumes": [],
				"taskDefinitionArn": "arn:aws:ecs:us-east-1:778743527532:task-definition/convox-myapp-web:34",
				"containerDefinitions": [
					{
						"environment": [
							{
								"name": "RELEASE",
								"value": "R1234"
							}
						],
						"name": "web",
						"mountPoints": [],
						"image": "778743527532.dkr.ecr.us-east-1.amazonaws.com/convox-myapp-nkdecwppkq:web.BMPBJLITPZT",
						"cpu": 0,
						"portMappings": [],
						"memory": 256,
						"privileged": false,
						"essential": true,
						"volumesFrom": []
					}
				],
				"revision": 34
			}
		}`,
	},
}

var cycleSystemDescribeContainerInstances = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.DescribeContainerInstances",
		Body: `{
			"cluster": "cluster-test",
			"containerInstances": [
				"arn:aws:ecs:us-east-1:778743527532:container-instance/e126c67d-fa95-4b09-8b4a-3723932cd2aa"
			]
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"failures": [],
			"containerInstances": [
				{
					"ec2InstanceId": "i-5bc45dc2"
				}
			]
		}`,
	},
}

var cycleSystemDescribeRackInstances = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "",
		Body:       `Action=DescribeInstances&Filter.1.Name=tag%3ARack&Filter.1.Value.1=convox&Version=2016-11-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<?xml version="1.0" encoding="UTF-8"?>
			<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
				<reservationSet>
					<item>
						<reservationId>r-003ed1d7</reservationId>
						<ownerId>778743527532</ownerId>
						<groupSet/>
						<instancesSet>
							<item>
								<instanceId>i-5bc45dc2</instanceId>
								<privateIpAddress>10.0.1.244</privateIpAddress>
							</item>
						</instancesSet>
					</item>
				</reservationSet>
			</DescribeInstancesResponse>
		}`,
	},
}

var cycleSystemDescribeTaskDefinition2 = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition",
		Body: `{
			"taskDefinition": "arn:aws:ecs:us-east-1:778743527532:task-definition/convox-myapp-web:34"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskDefinition": {
				"status": "ACTIVE",
				"family": "convox-myapp-web",
				"requiresAttributes": [
					{
						"name": "com.amazonaws.ecs.capability.ecr-auth"
					}
				],
				"volumes": [],
				"taskDefinitionArn": "arn:aws:ecs:us-east-1:778743527532:task-definition/convox-myapp-web:34",
				"containerDefinitions": [
					{
						"environment": [
							{
								"name": "RELEASE",
								"value": "R1234"
							}
						],
						"name": "web",
						"mountPoints": [],
						"image": "778743527532.dkr.ecr.us-east-1.amazonaws.com/convox-myapp-nkdecwppkq:web.BMPBJLITPZT",
						"cpu": 0,
						"portMappings": [],
						"memory": 256,
						"privileged": false,
						"essential": true,
						"volumesFrom": []
					}
				],
				"revision": 34
			}
		}`,
	},
}

var cycleSystemDockerListContainers2 = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "GET",
		RequestURI: "/containers/json?all=1&filters=%7B%22label%22%3A%5B%22com.amazonaws.ecs.task-arn%3Darn%3Aaws%3Aecs%3Aus-east-1%3A778743527532%3Atask%2F50b8de99-f94f-4ecd-a98f-5850760f0845%22%5D%7D",
		Operation:  "",
		Body:       ``,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `[
			{
				"Id": "8dfafdbc3a40",
				"Names":["/boring_feynman"],
				"Image": "ubuntu:latest",
				"ImageID": "d74508fb6632491cea586a1fd7d748dfc5274cd6fdfedee309ecdcbc2bf5cb82",
				"Command": "echo 1",
				"Created": 1367854155,
				"State": "Exited",
				"Status": "Exit 0",
				"Ports": [{"PrivatePort": 2222, "PublicPort": 3333, "Type": "tcp"}]
			}
		]`,
	},
}

var cycleSystemListTasksByStack = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.ListTasks",
		Body: `{
			"cluster": "cluster-test",
			"serviceName": "service-web"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskArns": [
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0846"
			]
		}`,
	},
}
