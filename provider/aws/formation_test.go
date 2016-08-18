package aws_test

import (
	"testing"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func TestFormationList(t *testing.T) {
	aws, provider := StubAwsProvider(
		describeStacksCycle,

		describeStacksCycle,
		formationGetItemCycle,
	)
	defer aws.Close()

	r, err := provider.FormationList("httpd")

	assert.Nil(t, err)
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

func TestFormationGet(t *testing.T) {
	aws, provider := StubAwsProvider(
		describeStacksCycle,

		describeStacksCycle,
		formationGetItemCycle,
	)
	defer aws.Close()

	r, err := provider.FormationGet("httpd", "web")

	assert.Nil(t, err)
	assert.EqualValues(t, &structs.ProcessFormation{
		Balancer: "httpd-web-7E5UPCM-1241527783.us-east-1.elb.amazonaws.com",
		Name:     "web",
		Count:    1,
		Memory:   256,
		CPU:      256,
		Ports:    []int{80},
	}, r)
}

func TestFormationSave(t *testing.T) {
	aws, provider := StubAwsProvider(
		describeStacksCycle,

		test.ListContainerInstancesCycle(""),
		test.DescribeContainerInstancesCycle(""),
		test.ListServicesCycle(""),
		test.DescribeServicesCycle(""),
		test.DescribeTaskDefinition1Cycle(""),

		test.DescribeAppStackCycle("convox-httpd"),

		formationGetItemCycle,
		formationPublishNotificationCycle,
		formationUpdateStackCycle,
	)
	defer aws.Close()

	pf := &structs.ProcessFormation{
		Balancer: "httpd-web-7E5UPCM-1241527783.us-east-1.elb.amazonaws.com",
		Name:     "web",
		Count:    1,
		Memory:   512,
		CPU:      256,
		Ports:    []int{80},
	}

	err := provider.FormationSave("httpd", pf)

	assert.Nil(t, err)
}

var formationGetItemCycle = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.GetItem",
		Body:       `{"ConsistentRead":true,"Key":{"id":{"S":"RVFETUHHKKD"}},"TableName":"convox-releases"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{"Item":{"id":{"S":"RVFETUHHKKD"},"build":{"S":"BHINCLZYYVN"},"app":{"S":"httpd"},"manifest":{"S":"web:\n  image: httpd\n  ports:\n  - 80:80\n"},"env":{"S":"foo=bar"},"created":{"S":"20160404.143542.627770380"}}}`,
	},
}

var formationPublishNotificationCycle = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=Publish&Message=%7B%22action%22%3A%22release%3Ascale%22%2C%22status%22%3A%22success%22%2C%22data%22%3A%7B%22app%22%3A%22httpd%22%2C%22id%22%3A%22RVFETUHHKKD%22%7D%2C%22timestamp%22%3A%220001-01-01T00%3A00%3A00Z%22%7D&Subject=release%3Ascale&TargetArn=&Version=2010-03-31`,
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

var formationUpdateStackCycle = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=UpdateStack&Capabilities.member.1=CAPABILITY_IAM&Parameters.member.1.ParameterKey=Cluster&Parameters.member.1.ParameterValue=convox-Cluster-1E4XJ0PQWNAYS&Parameters.member.10.ParameterKey=Version&Parameters.member.10.ParameterValue=20160330143438-command-exec-form&Parameters.member.11.ParameterKey=WebCpu&Parameters.member.11.ParameterValue=256&Parameters.member.12.ParameterKey=WebDesiredCount&Parameters.member.12.ParameterValue=1&Parameters.member.13.ParameterKey=WebMemory&Parameters.member.13.ParameterValue=512&Parameters.member.14.ParameterKey=WebPort80Balancer&Parameters.member.14.ParameterValue=80&Parameters.member.15.ParameterKey=WebPort80Certificate&Parameters.member.15.ParameterValue=&Parameters.member.16.ParameterKey=WebPort80Host&Parameters.member.16.ParameterValue=56694&Parameters.member.17.ParameterKey=WebPort80ProxyProtocol&Parameters.member.17.ParameterValue=No&Parameters.member.18.ParameterKey=WebPort80Secure&Parameters.member.18.ParameterValue=No&Parameters.member.2.ParameterKey=Environment&Parameters.member.2.ParameterValue=https%3A%2F%2Fconvox-httpd-settings-139bidzalmbtu.s3.amazonaws.com%2Freleases%2FRVFETUHHKKD%2Fenv&Parameters.member.3.ParameterKey=Key&Parameters.member.3.ParameterValue=arn%3Aaws%3Akms%3Aus-east-1%3A132866487567%3Akey%2Fd9f38426-9017-4931-84f8-604ad1524920&Parameters.member.4.ParameterKey=Private&Parameters.member.4.ParameterValue=Yes&Parameters.member.5.ParameterKey=Release&Parameters.member.5.ParameterValue=RVFETUHHKKD&Parameters.member.6.ParameterKey=Repository&Parameters.member.6.ParameterValue=&Parameters.member.7.ParameterKey=Subnets&Parameters.member.7.ParameterValue=subnet-13de3139%2Csubnet-b5578fc3%2Csubnet-21c13379&Parameters.member.8.ParameterKey=SubnetsPrivate&Parameters.member.8.ParameterValue=subnet-d4e85cfe%2Csubnet-103d5a66%2Csubnet-57952a0f&Parameters.member.9.ParameterKey=VPC&Parameters.member.9.ParameterValue=vpc-f8006b9c&StackName=convox-httpd&UsePreviousTemplate=true&Version=2010-05-15`,
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
