package aws_test

import (
	"testing"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func TestServiceWebhookURL(t *testing.T) {
	provider := StubAwsProvider(
		test.DescribeStackNotFound("convox-mywebhook"),
		test.DescribeStackNotFound("mywebhook"),
		cycleServiceCreateWebhook,
	)
	defer provider.Close()

	params := map[string]string{
		"url": "https://www.example.com",
	}

	url := "http://notifications.example.org/sns?endpoint=https%3A%2F%2Fwww.example.com"
	s, err := provider.ServiceCreate("mywebhook", "webhook", params)

	if assert.Nil(t, err) {
		assert.Equal(t, url, s.Parameters["Url"])
	}
}

func TestServiceGet(t *testing.T) {
	provider := StubAwsProvider(
		test.DescribeStackNotFound("convox-syslog"),
		cycleServiceDescribeStacks1,
		cycleAppDescribeStacks,
	)
	defer provider.Close()

	expected := &structs.Service{
		Name:   "syslog",
		Stack:  "syslog",
		Status: "running",
		Apps: structs.Apps{
			structs.App{
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
			},
		},
		Tags: map[string]string{
			"Type":   "service",
			"Name":   "syslog",
			"System": "convox",
			"Rack":   "convox",
		},
		Outputs: map[string]string{
			"Url":       "tcp+tls://logs1.example.com:11235",
			"HttpdLink": "convox-httpd-LogGroup-12345678",
		},
		Exports:    map[string]string{},
		Parameters: map[string]string{},
	}

	s, err := provider.ServiceGet("syslog")

	if assert.Nil(t, err) {
		assert.EqualValues(t, expected, s)
	}
}

var cycleServiceDescribeStacks1 = awsutil.Cycle{
	awsutil.Request{"/", "", `Action=DescribeStacks&StackName=syslog&Version=2010-05-15`},
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

var cycleServiceDescribeStacks2 = awsutil.Cycle{
	awsutil.Request{"/", "", `Action=DescribeStacks&StackName=syslog&Version=2010-05-15`},
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

var cycleServiceCreateWebhook = awsutil.Cycle{
	awsutil.Request{"/", "", `Action=CreateStack&Capabilities.member.1=CAPABILITY_IAM&Parameters.member.1.ParameterKey=CustomTopic&Parameters.member.1.ParameterValue=&Parameters.member.2.ParameterKey=NotificationTopic&Parameters.member.2.ParameterValue=&Parameters.member.3.ParameterKey=Url&Parameters.member.3.ParameterValue=http%3A%2F%2Fnotifications.example.org%2Fsns%3Fendpoint%3Dhttps%253A%252F%252Fwww.example.com&StackName=convox-mywebhook&Tags.member.1.Key=Name&Tags.member.1.Value=mywebhook&Tags.member.2.Key=Rack&Tags.member.2.Value=convox&Tags.member.3.Key=Service&Tags.member.3.Value=webhook&Tags.member.4.Key=System&Tags.member.4.Value=convox&Tags.member.5.Key=Type&Tags.member.5.Value=service&TemplateBody=%0A%7B%0A++%22AWSTemplateFormatVersion%22+%3A+%222010-09-09%22%2C%0A++%22Parameters%22%3A+%7B%0A++++%22Url%22%3A+%7B%0A++++++%22Type%22+%3A+%22String%22%2C%0A++++++%22Description%22+%3A+%22Webhook+URL%2C+e.g.+%27https%3A%2F%2Fgrid.convox.com%2Frack-hook%2F1234%27%22%0A++++%7D%2C%0A++++%22CustomTopic%22%3A+%7B%0A++++++%22Type%22+%3A+%22String%22%2C%0A++++++%22Description%22+%3A+%22%22%0A++++%7D%2C%0A++++%22NotificationTopic%22%3A+%7B%0A++++++%22Type%22+%3A+%22String%22%2C%0A++++++%22Description%22+%3A+%22%22%0A++++%7D%0A++%7D%2C%0A++%22Resources%22%3A+%7B%0A++++%22Notifications%22%3A+%7B%0A++++++%22Type%22+%3A+%22Custom%3A%3ASNSSubscription%22%2C%0A++++++%22Version%22%3A+%221.0%22%2C%0A++++++%22Properties%22%3A+%7B%0A++++++++%22ServiceToken%22%3A+%7B+%22Ref%22%3A+%22CustomTopic%22+%7D%2C%0A++++++++%22TopicArn%22+%3A+%7B+%22Ref%22%3A+%22NotificationTopic%22+%7D%2C%0A++++++++%22Protocol%22+%3A+%22http%22%2C%0A++++++++%22Endpoint%22+%3A+%7B+%22Ref%22%3A+%22Url%22+%7D%0A++++++%7D%0A++++%7D%0A++%7D%2C%0A++%22Outputs%22%3A+%7B%0A++++%22Url%22%3A+%7B%0A++++++%22Value%22%3A+%7B+%22Ref%22%3A+%22Url%22+%7D%0A++++%7D%0A++%7D%0A%7D%0A&Version=2010-05-15`},
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
