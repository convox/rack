package aws_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/provider/aws"
	"github.com/stretchr/testify/assert"
)

func TestFormationList(t *testing.T) {
	provider := StubAwsProvider(
		cycleFormationDescribeStacks,
		cycleFormationDescribeStacks,
		cycleFormationDescribeStacks,
		cycleReleaseGetItem,
		cycleReleaseDescribeStackResources,
		cycleReleaseEnvironmentGet,
	)
	defer provider.Close()

	r, err := provider.FormationList("httpd")

	assert.NoError(t, err)
	assert.EqualValues(t, structs.Formation{
		structs.ProcessFormation{
			Balancer: "httpd-web-7E5UPCM-1241527783.us-east-1.elb.amazonaws.com",
			Name:     "web",
			Count:    1,
			Memory:   256,
			CPU:      256,
			Ports:    []int{80},
		},
	}, r)
}

func TestFormationListBadApp(t *testing.T) {
	provider := StubAwsProvider(
		cycleDescribeStacksNotFound("convox-httpe"),
	)
	defer provider.Close()

	r, err := provider.FormationList("httpe")

	assert.Nil(t, r)
	assert.True(t, aws.ErrorNotFound(err))
	assert.Equal(t, "httpe not found", err.Error())
}

func TestFormationListEmptyRelease(t *testing.T) {
	provider := StubAwsProvider(
		cycleFormationDescribeStacksEmptyRelease,
		cycleFormationDescribeStacksEmptyRelease,
	)
	defer provider.Close()

	r, err := provider.FormationList("httpd")

	assert.Equal(t, structs.Formation{}, r)
	assert.NoError(t, err)
}

func TestFormationListBadRelease(t *testing.T) {
	provider := StubAwsProvider(
		cycleFormationDescribeStacks,
		cycleFormationDescribeStacks,
		cycleFormationDescribeStacks,
		cycleReleaseGetItemNotFound,
	)
	defer provider.Close()

	r, err := provider.FormationList("httpd")

	assert.Nil(t, r)
	assert.True(t, aws.ErrorNotFound(err))
	assert.Equal(t, "no such release: RVFETUHHKKD", err.Error())
}

func TestFormationListBadManifest(t *testing.T) {
	provider := StubAwsProvider(
		cycleFormationDescribeStacks,
		cycleFormationDescribeStacks,
		cycleFormationDescribeStacks,
		cycleReleaseGetItemBadManifest,
		cycleReleaseDescribeStackResources,
		cycleReleaseEnvironmentGet,
	)
	defer provider.Close()

	r, err := provider.FormationList("httpd")

	assert.Nil(t, r)
	assert.Equal(t, fmt.Errorf("could not parse manifest for release: RVFETUHHKKD"), err)
}

func TestFormationListBadFormation(t *testing.T) {
	provider := StubAwsProvider(
		cycleFormationDescribeStacksBadFormation,
		cycleFormationDescribeStacksBadFormation,
		cycleFormationDescribeStacksBadFormation,
		cycleReleaseGetItem,
		cycleReleaseDescribeStackResources,
		cycleReleaseEnvironmentGet,
	)
	defer provider.Close()

	r, err := provider.FormationList("httpd")

	assert.Nil(t, r)
	assert.Equal(t, fmt.Errorf("web cpu not numeric"), err)
}

func TestFormationGet(t *testing.T) {
	provider := StubAwsProvider(
		cycleFormationDescribeStacks,
		cycleFormationDescribeStacks,
		cycleFormationDescribeStacks,
		cycleReleaseGetItem,
		cycleReleaseDescribeStackResources,
		cycleReleaseEnvironmentGet,
	)
	defer provider.Close()

	r, err := provider.FormationGet("httpd", "web")

	assert.NoError(t, err)
	assert.EqualValues(t, &structs.ProcessFormation{
		Balancer: "httpd-web-7E5UPCM-1241527783.us-east-1.elb.amazonaws.com",
		Name:     "web",
		Count:    1,
		Memory:   256,
		CPU:      256,
		Ports:    []int{80},
	}, r)
}

func TestFormationGetBadApp(t *testing.T) {
	provider := StubAwsProvider(
		cycleDescribeStacksNotFound("convox-httpe"),
	)
	defer provider.Close()

	r, err := provider.FormationGet("httpe", "web")

	assert.Nil(t, r)
	assert.True(t, aws.ErrorNotFound(err))
	assert.Equal(t, "httpe not found", err.Error())
}

func TestFormationGetEmptyRelease(t *testing.T) {
	provider := StubAwsProvider(
		cycleFormationDescribeStacks,
		cycleFormationDescribeStacksEmptyRelease,
	)
	defer provider.Close()

	r, err := provider.FormationGet("httpd", "web")

	assert.Nil(t, r)
	assert.Equal(t, fmt.Errorf("no release for app: httpd"), err)
}

func TestFormationGetBadRelease(t *testing.T) {
	provider := StubAwsProvider(
		cycleFormationDescribeStacks,
		cycleFormationDescribeStacks,
		cycleFormationDescribeStacks,
		cycleReleaseGetItemNotFound,
	)
	defer provider.Close()

	r, err := provider.FormationGet("httpd", "web")

	assert.Nil(t, r)
	assert.True(t, aws.ErrorNotFound(err))
	assert.Equal(t, "no such release: RVFETUHHKKD", err.Error())
}

func TestFormationGetBadManifest(t *testing.T) {
	provider := StubAwsProvider(
		cycleFormationDescribeStacks,
		cycleFormationDescribeStacks,
		cycleFormationDescribeStacks,
		cycleReleaseGetItemBadManifest,
		cycleReleaseDescribeStackResources,
		cycleReleaseEnvironmentGet,
	)
	defer provider.Close()

	r, err := provider.FormationGet("httpd", "web")

	assert.Nil(t, r)
	assert.Equal(t, fmt.Errorf("could not parse manifest for release: RVFETUHHKKD"), err)
}

func TestFormationGetUnknownProcess(t *testing.T) {
	provider := StubAwsProvider(
		cycleFormationDescribeStacks,
		cycleFormationDescribeStacks,
		cycleFormationDescribeStacks,
		cycleReleaseGetItem,
		cycleReleaseDescribeStackResources,
		cycleReleaseEnvironmentGet,
	)
	defer provider.Close()

	r, err := provider.FormationGet("httpd", "unknown")

	assert.Nil(t, r)
	assert.Equal(t, fmt.Errorf("no such process: unknown"), err)
}

func TestFormationSave(t *testing.T) {
	provider := StubAwsProvider(
		cycleFormationDescribeStacks,
		cycleCapacityListContainerInstances,
		cycleCapacityDescribeContainerInstances,
		cycleCapacityListServices,
		cycleCapacityDescribeServices,
		cycleCapacityDescribeTaskDefinition2,
		cycleCapacityDescribeTaskDefinition1,
		cycleCapacityDescribeTaskDefinition1,
		cycleFormationDescribeStack,
		cycleFormationUpdateStack,
		cycleNotificationPublish,
	)
	defer provider.Close()

	pf := &structs.ProcessFormation{
		Name:   "web",
		Count:  1,
		Memory: 512,
		CPU:    256,
	}

	err := provider.FormationSave("httpd", pf)

	assert.NoError(t, err)
}

func TestFormationSaveBadApp(t *testing.T) {
	provider := StubAwsProvider(
		cycleDescribeStacksNotFound("convox-httpe"),
	)
	defer provider.Close()

	pf := &structs.ProcessFormation{
		Name:   "web",
		Count:  1,
		Memory: 512,
		CPU:    256,
	}

	err := provider.FormationSave("httpe", pf)

	assert.True(t, aws.ErrorNotFound(err))
	assert.Equal(t, "httpe not found", err.Error())
}

func TestFormationSaveBadCluster(t *testing.T) {
	provider := StubAwsProvider(
		cycleFormationDescribeStacks,
		cycleCapacityListContainerInstancesBadCluster,
	)
	defer provider.Close()

	pf := &structs.ProcessFormation{
		Name:   "web",
		Count:  1,
		Memory: 512,
		CPU:    256,
	}

	err := provider.FormationSave("httpd", pf)

	assert.True(t, aws.ErrorNotFound(err))
	assert.Equal(t, "cluster not found: cluster-test", err.Error())
}

func TestFormationSaveBadCount(t *testing.T) {
	provider := StubAwsProvider(
		cycleFormationDescribeStacks,
		cycleCapacityListContainerInstances,
		cycleCapacityDescribeContainerInstances,
		cycleCapacityListServices,
		cycleCapacityDescribeServices,
		cycleCapacityDescribeTaskDefinition2,
		cycleCapacityDescribeTaskDefinition1,
		cycleCapacityDescribeTaskDefinition1,
	)
	defer provider.Close()

	pf := &structs.ProcessFormation{
		Name:  "web",
		Count: -2,
	}

	err := provider.FormationSave("httpd", pf)

	assert.Equal(t, fmt.Errorf("requested count -2 must be -1 or greater"), err)
}

func TestFormationSaveCpuTooSmall(t *testing.T) {
	provider := StubAwsProvider(
		cycleFormationDescribeStacks,
		cycleCapacityListContainerInstances,
		cycleCapacityDescribeContainerInstances,
		cycleCapacityListServices,
		cycleCapacityDescribeServices,
		cycleCapacityDescribeTaskDefinition2,
		cycleCapacityDescribeTaskDefinition1,
		cycleCapacityDescribeTaskDefinition1,
	)
	defer provider.Close()

	pf := &structs.ProcessFormation{
		Name: "web",
		CPU:  -1,
	}

	err := provider.FormationSave("httpd", pf)

	assert.Equal(t, fmt.Errorf("requested cpu -1 must be 0 or greater"), err)
}

func TestFormationSaveCpuTooLarge(t *testing.T) {
	provider := StubAwsProvider(
		cycleFormationDescribeStacks,
		cycleCapacityListContainerInstances,
		cycleCapacityDescribeContainerInstances,
		cycleCapacityListServices,
		cycleCapacityDescribeServices,
		cycleCapacityDescribeTaskDefinition2,
		cycleCapacityDescribeTaskDefinition1,
		cycleCapacityDescribeTaskDefinition1,
	)
	defer provider.Close()

	pf := &structs.ProcessFormation{
		Name: "web",
		CPU:  10000,
	}

	err := provider.FormationSave("httpd", pf)

	assert.Equal(t, fmt.Errorf("requested cpu 10000 greater than instance size 1024"), err)
}

func TestFormationSaveMemoryTooLarge(t *testing.T) {
	provider := StubAwsProvider(
		cycleFormationDescribeStacks,
		cycleCapacityListContainerInstances,
		cycleCapacityDescribeContainerInstances,
		cycleCapacityListServices,
		cycleCapacityDescribeServices,
		cycleCapacityDescribeTaskDefinition2,
		cycleCapacityDescribeTaskDefinition1,
		cycleCapacityDescribeTaskDefinition1,
	)
	defer provider.Close()

	pf := &structs.ProcessFormation{
		Name:   "web",
		Memory: 20000,
	}

	err := provider.FormationSave("httpd", pf)

	assert.Equal(t, fmt.Errorf("requested memory 20000 greater than instance size 2004"), err)
}

var cycleFormationDescribeStack = awsutil.Cycle{
	awsutil.Request{"POST", "/", "", `Action=DescribeStacks&StackName=convox-httpd&Version=2010-05-15`},
	awsutil.Response{
		200,
		`<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
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
		</DescribeStacksResponse>`,
	},
}

var cycleFormationDescribeStacks = awsutil.Cycle{
	awsutil.Request{"POST", "/", "", `Action=DescribeStacks&StackName=convox-httpd&Version=2010-05-15`},
	awsutil.Response{
		200,
		`<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
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
		</DescribeStacksResponse>`,
	},
}

var cycleFormationDescribeStacksBadFormation = awsutil.Cycle{
	awsutil.Request{"POST", "/", "", `Action=DescribeStacks&StackName=convox-httpd&Version=2010-05-15`},
	awsutil.Response{
		200,
		`<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
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
								<ParameterValue>1,foo,bar</ParameterValue>
								<ParameterKey>WebFormation</ParameterKey>
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
		</DescribeStacksResponse>`,
	},
}

var cycleFormationDescribeStacksEmptyRelease = awsutil.Cycle{
	awsutil.Request{"POST", "/", "", `Action=DescribeStacks&StackName=convox-httpd&Version=2010-05-15`},
	awsutil.Response{
		200,
		`<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
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
								<ParameterValue></ParameterValue>
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
		</DescribeStacksResponse>`,
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

var cycleNotificationPublish = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=Publish&Message=%7B%22action%22%3A%22release%3Ascale%22%2C%22status%22%3A%22success%22%2C%22data%22%3A%7B%22app%22%3A%22httpd%22%2C%22id%22%3A%22RVFETUHHKKD%22%2C%22rack%22%3A%22convox%22%7D%2C%22timestamp%22%3A%220001-01-01T00%3A00%3A00Z%22%7D&Subject=release%3Ascale&TargetArn=&Version=2010-03-31`,
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

var cycleFormationUpdateStack = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=UpdateStack&Capabilities.member.1=CAPABILITY_IAM&NotificationARNs.member.1=&Parameters.member.1.ParameterKey=Cluster&Parameters.member.1.UsePreviousValue=true&Parameters.member.10.ParameterKey=Version&Parameters.member.10.UsePreviousValue=true&Parameters.member.11.ParameterKey=WebCpu&Parameters.member.11.ParameterValue=256&Parameters.member.12.ParameterKey=WebDesiredCount&Parameters.member.12.ParameterValue=1&Parameters.member.13.ParameterKey=WebMemory&Parameters.member.13.ParameterValue=512&Parameters.member.14.ParameterKey=WebPort80Balancer&Parameters.member.14.UsePreviousValue=true&Parameters.member.15.ParameterKey=WebPort80Certificate&Parameters.member.15.UsePreviousValue=true&Parameters.member.16.ParameterKey=WebPort80Host&Parameters.member.16.UsePreviousValue=true&Parameters.member.17.ParameterKey=WebPort80ProxyProtocol&Parameters.member.17.UsePreviousValue=true&Parameters.member.18.ParameterKey=WebPort80Secure&Parameters.member.18.UsePreviousValue=true&Parameters.member.2.ParameterKey=Environment&Parameters.member.2.UsePreviousValue=true&Parameters.member.3.ParameterKey=Key&Parameters.member.3.UsePreviousValue=true&Parameters.member.4.ParameterKey=Private&Parameters.member.4.UsePreviousValue=true&Parameters.member.5.ParameterKey=Release&Parameters.member.5.UsePreviousValue=true&Parameters.member.6.ParameterKey=Repository&Parameters.member.6.UsePreviousValue=true&Parameters.member.7.ParameterKey=Subnets&Parameters.member.7.UsePreviousValue=true&Parameters.member.8.ParameterKey=SubnetsPrivate&Parameters.member.8.UsePreviousValue=true&Parameters.member.9.ParameterKey=VPC&Parameters.member.9.UsePreviousValue=true&StackName=convox-httpd&UsePreviousTemplate=true&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<UpdateStackResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
				<UpdateStackResult>
					<StackId>arn:aws:cloudformation:us-east-1:901416387788:stack/convox-httpd/9a10bbe0-51d5-11e5-b85a-5001dc3ed8d2</StackId>
				</UpdateStackResult>
				<ResponseMetadata>
					<RequestId>b9b4b068-3a41-11e5-94eb-example</RequestId>
				</ResponseMetadata>
			</UpdateStackResponse>
		`,
	},
}
