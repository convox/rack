package controllers_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/convox/kernel/awsutil"
	"github.com/convox/kernel/controllers"
)

func TestAppList(t *testing.T) {
	handler := awsutil.NewHandler([]awsutil.Cycle{
		DescribeStacksCycle("bar"),
	})
	s := httptest.NewServer(handler)
	aws.DefaultConfig.Endpoint = s.URL
	defer s.Close()

	body := assert.HTTPBody(controllers.HandlerFunc, "GET", "http://convox/apps", nil)

	var resp []map[string]string
	err := json.Unmarshal([]byte(body), &resp)

	if assert.Nil(t, err) {
		assert.Equal(t, "bar", resp[0]["name"])
	}
}

func TestAppShowFound(t *testing.T) {
	handler := awsutil.NewHandler([]awsutil.Cycle{
		DescribeStackCycle("bar"),
	})
	s := httptest.NewServer(handler)
	aws.DefaultConfig.Endpoint = s.URL
	defer s.Close()

	router := controllers.NewRouter()
	body := assert.HTTPBody(router.ServeHTTP, "GET", "http://convox/apps/bar", nil)

	t.Log(body)

	//assert.Nil(t, err)
}

func TestAppShowWithNoApp(t *testing.T) {

}

func TestAppCreate(t *testing.T) {

}

func TestAppCreateWithAlreadyExists(t *testing.T) {

}

func TestAppDelete(t *testing.T) {

}

func TestAppDeleteWithNoApp(t *testing.T) {

}

func TestAppLogs(t *testing.T) {

}

func DescribeStacksCycle(appName string) awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/",
			Operation:  "",
			Body:       `Action=DescribeStacks&Version=2010-05-15`,
		},
		Response: awsutil.Response{
			StatusCode: 200,
			Body: ` <DescribeStacksResult>
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
        </Tags>
        <StackId>arn:aws:cloudformation:us-east-1:938166070011:stack/` + appName + `/9a10bbe0-51d5-11e5-b85a-5001dc3ed8d2</StackId>
        <StackStatus>CREATE_COMPLETE</StackStatus>
        <StackName>` + appName + `</StackName>
        <NotificationARNs/>
        <CreationTime>2015-09-03T00:49:16.068Z</CreationTime>
        <Parameters>
          <member>
            <ParameterValue>https://apache-app2-settings-1vudpykaywx8o.s3.amazonaws.com/releases/RCSUVJNDLDK/env</ParameterValue>
            <ParameterKey>Environment</ParameterKey>
          </member>
          <member>
            <ParameterValue>arn:aws:kms:us-east-1:938166070011:key/e4c9e19c-7410-4e0f-88bf-ac7ac085625d</ParameterValue>
            <ParameterKey>Key</ParameterKey>
          </member>
          <member>
            <ParameterValue>256</ParameterValue>
            <ParameterKey>MainMemory</ParameterKey>
          </member>
          <member>
            <ParameterValue/>
            <ParameterKey>Repository</ParameterKey>
          </member>
          <member>
            <ParameterValue>vpc-e853928c</ParameterValue>
            <ParameterKey>VPC</ParameterKey>
          </member>
          <member>
            <ParameterValue/>
            <ParameterKey>MainService</ParameterKey>
          </member>
          <member>
            <ParameterValue>convo-Clust-GEFJGLHH7O0V</ParameterValue>
            <ParameterKey>Cluster</ParameterKey>
          </member>
          <member>
            <ParameterValue>RCSUVJNDLDK</ParameterValue>
            <ParameterKey>Release</ParameterKey>
          </member>
          <member>
            <ParameterValue>80</ParameterValue>
            <ParameterKey>MainPort80Balancer</ParameterKey>
          </member>
          <member>
            <ParameterValue>200</ParameterValue>
            <ParameterKey>Cpu</ParameterKey>
          </member>
          <member>
            <ParameterValue>1</ParameterValue>
            <ParameterKey>MainDesiredCount</ParameterKey>
          </member>
          <member>
            <ParameterValue>subnet-2f5e0804,subnet-74a4aa03,subnet-f0c3e3a9</ParameterValue>
            <ParameterKey>Subnets</ParameterKey>
          </member>
          <member>
            <ParameterValue/>
            <ParameterKey>MainCommand</ParameterKey>
          </member>
          <member>
            <ParameterValue>33787</ParameterValue>
            <ParameterKey>MainPort80Host</ParameterKey>
          </member>
          <member>
            <ParameterValue>latest</ParameterValue>
            <ParameterKey>Version</ParameterKey>
          </member>
          <member>
          <ParameterValue>convox-720091589.us-east-1.elb.amazonaws.com:5000/apache-app2-main:BDDTZLECEZN</ParameterValue>
          <ParameterKey>MainImage</ParameterKey>
          </member>
        </Parameters>
        <Capabilities>
          <member>CAPABILITY_IAM</member>
        </Capabilities>
        <DisableRollback>false</DisableRollback>
        <Outputs>
          <member>
            <OutputValue>convox-1842138601.us-east-1.elb.amazonaws.com</OutputValue>
            <OutputKey>Dashboard</OutputKey>
          </member>
          <member>
            <OutputValue>convox-Kinesis-1BGCFIB6PK55Y</OutputValue>
            <OutputKey>Kinesis</OutputKey>
          </member>
        </Outputs>
      </member>
      <member>
        <Tags/>
        <StackId>arn:aws:cloudformation:us-east-1:938166070011:stack/foo/9a10bbe0-51d5-11e5-b85a-5001dc3ed8d2</StackId>
        <StackStatus>CREATE_COMPLETE</StackStatus>
        <StackName>foo</StackName>
        <NotificationARNs/>
        <CreationTime>2015-09-03T00:49:16.068Z</CreationTime>
        <Parameters>
          <member>
            <ParameterValue>3</ParameterValue>
            <ParameterKey>InstanceCount</ParameterKey>
          </member>
          <member>
            <ParameterValue/>
            <ParameterKey>RegistryHost</ParameterKey>
          </member>
          <member>
            <ParameterValue/>
            <ParameterKey>Key</ParameterKey>
          </member>
          <member>
            <ParameterValue/>
            <ParameterKey>Ami</ParameterKey>
          </member>
          <member>
            <ParameterValue>LmAlykMYpjFVKopVgibGfxjVnNCZVi</ParameterValue>
            <ParameterKey>Password</ParameterKey>
          </member>
          <member>
            <ParameterValue/>
            <ParameterKey>RegistryPort</ParameterKey>
          </member>
          <member>
            <ParameterValue>No</ParameterValue>
            <ParameterKey>Development</ParameterKey>
          </member>
          <member>
            <ParameterValue>latest</ParameterValue>
            <ParameterKey>Version</ParameterKey>
          </member>
          <member>
            <ParameterValue>test@convox.com</ParameterValue>
            <ParameterKey>ClientId</ParameterKey>
          </member>
          <member>
            <ParameterValue/>
            <ParameterKey>Certificate</ParameterKey>
          </member>
          <member>
            <ParameterValue>default</ParameterValue>
            <ParameterKey>Tenancy</ParameterKey>
          </member>
          <member>
            <ParameterValue>t2.small</ParameterValue>
            <ParameterKey>InstanceType</ParameterKey>
          </member>
        </Parameters>
        <Capabilities>
          <member>CAPABILITY_IAM</member>
        </Capabilities>
        <DisableRollback>false</DisableRollback>
        <Outputs>
          <member>
            <OutputValue>convox-1842138601.us-east-1.elb.amazonaws.com</OutputValue>
            <OutputKey>Dashboard</OutputKey>
          </member>
          <member>
            <OutputValue>convox-Kinesis-1BGCFIB6PK55Y</OutputValue>
            <OutputKey>Kinesis</OutputKey>
          </member>
        </Outputs>
      </member>
    </Stacks>
  </DescribeStacksResult>`,
		},
	}

}
