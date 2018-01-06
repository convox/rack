package aws_test

import (
	"testing"

	"github.com/convox/rack/structs"
	"github.com/convox/rack/test/awsutil"
	"github.com/stretchr/testify/assert"
)

func TestResourceWebhookURL(t *testing.T) {
	provider := StubAwsProvider(
		cycleDescribeStacksNotFound("convox-mywebhook"),
		cycleResourceCreateWebhook,
		cycleResourceCreateNotificationPublish,
	)
	defer provider.Close()

	params := map[string]string{
		"url": "https://www.example.com",
	}

	url := "http://notifications.example.org/sns?endpoint=https%3A%2F%2Fwww.example.com"
	s, err := provider.ResourceCreate("mywebhook", "webhook", structs.ResourceCreateOptions{Parameters: params})

	if assert.NoError(t, err) {
		assert.Equal(t, url, s.Parameters["Url"])
	}
}

func TestResourceList(t *testing.T) {
	provider := StubAwsProvider(
		cycleServiceDescribeStacksList,
		cycleResourceDescribeStacks,
		cycleAppDescribeStacks,
	)
	defer provider.Close()

	expected := structs.Resources{
		structs.Resource{
			Name:       "syslog",
			Status:     "running",
			Type:       "",
			Parameters: map[string]string{},
		},
	}

	s, err := provider.ResourceList()

	if assert.Nil(t, err) {
		assert.EqualValues(t, expected, s)
	}
}

func TestResourceGet(t *testing.T) {
	provider := StubAwsProvider(
		cycleResourceDescribeStacks,
		cycleResourceDescribeStacks,
		cycleAppDescribeStacks,
	)
	defer provider.Close()

	expected := &structs.Resource{
		Name:       "syslog",
		Status:     "running",
		Parameters: map[string]string{},
		Apps:       structs.Apps{structs.App{Generation: "1", Name: "httpd", Release: "RVFETUHHKKD", Status: "running", Outputs: map[string]string{"Kinesis": "convox-httpd-Kinesis-1MAP0GJ6RITJF", "LogGroup": "convox-httpd-LogGroup-L4V203L35WRM", "RegistryId": "132866487567", "RegistryRepository": "convox-httpd-hqvvfosgxt", "Settings": "convox-httpd-settings-139bidzalmbtu", "WebPort80Balancer": "80", "WebPort80BalancerName": "httpd-web-7E5UPCM", "BalancerWebHost": "httpd-web-7E5UPCM-1241527783.us-east-1.elb.amazonaws.com"}, Parameters: map[string]string{"WebCpu": "256", "WebMemory": "256", "WebPort80Secure": "No", "Environment": "https://convox-httpd-settings-139bidzalmbtu.s3.amazonaws.com/releases/RVFETUHHKKD/env", "WebPort80Certificate": "", "WebPort80ProxyProtocol": "No", "Key": "arn:aws:kms:us-east-1:132866487567:key/d9f38426-9017-4931-84f8-604ad1524920", "Repository": "", "WebPort80Host": "56694", "VPC": "vpc-f8006b9c", "WebDesiredCount": "1", "Cluster": "convox-Cluster-1E4XJ0PQWNAYS", "Release": "RVFETUHHKKD", "Private": "Yes", "WebPort80Balancer": "80", "SubnetsPrivate": "subnet-d4e85cfe,subnet-103d5a66,subnet-57952a0f", "Subnets": "subnet-13de3139,subnet-b5578fc3,subnet-21c13379", "Version": "20160330143438-command-exec-form"}, Tags: map[string]string{"Rack": "convox", "Name": "httpd", "Type": "app", "System": "convox"}}},
	}

	s, err := provider.ResourceGet("syslog")

	if assert.NoError(t, err) {
		assert.EqualValues(t, expected, s)
	}
}

var cycleResourceDescribeStacks = awsutil.Cycle{
	awsutil.Request{"POST", "/", "", `Action=DescribeStacks&StackName=convox-syslog&Version=2010-05-15`},
	awsutil.Response{
		200,
		`<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
			<DescribeStacksResult>
				<Stacks>
					<member>
						<Outputs>
							<member>
								<OutputKey>Url</OutputKey>
								<OutputValue>tcp+tls://logs1.example.com:11235</OutputValue>
							</member>
							<member>
								<OutputKey>HttpdLink</OutputKey>
								<OutputValue>convox-httpd-LogGroup-12345678</OutputValue>
							</member>
						</Outputs>
						<Capabilities>
							<member>CAPABILITY_IAM</member>
						</Capabilities>
						<CreationTime>2015-10-28T16:14:09.590Z</CreationTime>
						<NotificationARNs/>
						<StackId>arn:aws:cloudformation:us-east-1:778743527532:stack/convox/eb743e00-7d8e-11e5-8280-50ba0727c06e</StackId>
						<StackName>syslog</StackName>
						<StackStatus>UPDATE_COMPLETE</StackStatus>
						<DisableRollback>false</DisableRollback>
						<Tags>
							<member>
								<Value>resource</Value>
								<Key>Type</Key>
							</member>
							<member>
								<Value>syslog</Value>
								<Key>Name</Key>
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
						<LastUpdatedTime>2016-08-27T16:29:05.963Z</LastUpdatedTime>
						<Parameters>
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

var cycleServiceDescribeStacksList = awsutil.Cycle{
	awsutil.Request{"POST", "/", "", `Action=DescribeStacks&Version=2010-05-15`},
	awsutil.Response{
		200,
		`<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
			<DescribeStacksResult>
				<Stacks>
					<member>
						<Outputs>
							<member>
								<OutputKey>Url</OutputKey>
								<OutputValue>tcp+tls://logs1.example.com:11235</OutputValue>
							</member>
							<member>
								<OutputKey>HttpdLink</OutputKey>
								<OutputValue>convox-httpd-LogGroup-12345678</OutputValue>
							</member>
							<member>
								<OutputKey>Httpd2Link</OutputKey>
								<OutputValue></OutputValue>
							</member>
						</Outputs>
						<Capabilities>
							<member>CAPABILITY_IAM</member>
						</Capabilities>
						<CreationTime>2015-10-28T16:14:09.590Z</CreationTime>
						<NotificationARNs/>
						<StackId>arn:aws:cloudformation:us-east-1:778743527532:stack/convox/eb743e00-7d8e-11e5-8280-50ba0727c06e</StackId>
						<StackName>syslog</StackName>
						<StackStatus>UPDATE_COMPLETE</StackStatus>
						<DisableRollback>false</DisableRollback>
						<Tags>
							<member>
								<Value>service</Value>
								<Key>Type</Key>
							</member>
							<member>
								<Value>syslog</Value>
								<Key>Name</Key>
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
						<LastUpdatedTime>2016-08-27T16:29:05.963Z</LastUpdatedTime>
						<Parameters>
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

var cycleResourceCreateWebhook = awsutil.Cycle{
	awsutil.Request{"POST", "/", "", `Action=CreateStack&Capabilities.member.1=CAPABILITY_IAM&Parameters.member.1.ParameterKey=CustomTopic&Parameters.member.1.ParameterValue=&Parameters.member.2.ParameterKey=NotificationTopic&Parameters.member.2.ParameterValue=&Parameters.member.3.ParameterKey=Url&Parameters.member.3.ParameterValue=http%3A%2F%2Fnotifications.example.org%2Fsns%3Fendpoint%3Dhttps%253A%252F%252Fwww.example.com&StackName=convox-mywebhook&Tags.member.1.Key=Name&Tags.member.1.Value=mywebhook&Tags.member.2.Key=Rack&Tags.member.2.Value=convox&Tags.member.3.Key=Resource&Tags.member.3.Value=webhook&Tags.member.4.Key=System&Tags.member.4.Value=convox&Tags.member.5.Key=Type&Tags.member.5.Value=resource&TemplateBody=%0A%7B%0A++%22AWSTemplateFormatVersion%22+%3A+%222010-09-09%22%2C%0A++%22Parameters%22%3A+%7B%0A++++%22Url%22%3A+%7B%0A++++++%22Type%22+%3A+%22String%22%2C%0A++++++%22Description%22+%3A+%22Webhook+URL%2C+e.g.+%27https%3A%2F%2Fgrid.convox.com%2Frack-hook%2F1234%27%22%0A++++%7D%2C%0A++++%22CustomTopic%22%3A+%7B%0A++++++%22Type%22+%3A+%22String%22%2C%0A++++++%22Description%22+%3A+%22%22%0A++++%7D%2C%0A++++%22NotificationTopic%22%3A+%7B%0A++++++%22Type%22+%3A+%22String%22%2C%0A++++++%22Description%22+%3A+%22%22%0A++++%7D%0A++%7D%2C%0A++%22Resources%22%3A+%7B%0A++++%22Notifications%22%3A+%7B%0A++++++%22Type%22+%3A+%22Custom%3A%3ASNSSubscription%22%2C%0A++++++%22Version%22%3A+%221.0%22%2C%0A++++++%22Properties%22%3A+%7B%0A++++++++%22ServiceToken%22%3A+%7B+%22Ref%22%3A+%22CustomTopic%22+%7D%2C%0A++++++++%22TopicArn%22+%3A+%7B+%22Ref%22%3A+%22NotificationTopic%22+%7D%2C%0A++++++++%22Protocol%22+%3A+%22http%22%2C%0A++++++++%22Endpoint%22+%3A+%7B+%22Ref%22%3A+%22Url%22+%7D%0A++++++%7D%0A++++%7D%0A++%7D%2C%0A++%22Outputs%22%3A+%7B%0A++++%22Url%22%3A+%7B%0A++++++%22Value%22%3A+%7B+%22Ref%22%3A+%22Url%22+%7D%0A++++%7D%0A++%7D%0A%7D%0A&Version=2010-05-15`},
	awsutil.Response{
		200,
		`<CreateStackResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
                                <CreateStackResult>
                                        <StackId>arn:aws:cloudformation:us-east-1:990037048036:stack/` + "mywebhook" + `/cd77a770-7059-11e6-9f55-50fa5f2588d2</StackId>
                                </CreateStackResult>
                                <ResponseMetadata>
                                        <RequestId>cd6fdfe7-7059-11e6-af63-31e395e4ce23</RequestId>
                                </ResponseMetadata>
                        </CreateStackResponse>`,
	},
}

var cycleResourceCreateNotificationPublish = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=Publish&Message=%7B%22action%22%3A%22resource%3Acreate%22%2C%22status%22%3A%22success%22%2C%22data%22%3A%7B%22name%22%3A%22mywebhook%22%2C%22rack%22%3A%22convox%22%2C%22type%22%3A%22webhook%22%7D%2C%22timestamp%22%3A%220001-01-01T00%3A00%3A00Z%22%7D&Subject=resource%3Acreate&TargetArn=&Version=2010-03-31`,
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
