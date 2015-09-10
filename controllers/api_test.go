package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/awsutil"
	"github.com/convox/kernel/controllers"
)

/*
func TestNoAuth(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://convox/system", nil)
	w := httptest.NewRecorder()
	controllers.SingleRequest(w, req)

	if w.Code != 301 {
		t.Errorf("expected status code of %d, got %d", 301, w.Code)
		return
	}
}
*/

func stubSystem() (s *httptest.Server) {
	s = httptest.NewServer(awsutil.NewHandler([]awsutil.Cycle{
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "",
				Body:       `Action=DescribeStacks&StackName=convox-test&Version=2010-05-15`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body: ` <DescribeStacksResult>
    <Stacks>
      <member>
        <Tags/>
        <StackId>arn:aws:cloudformation:us-east-1:938166070011:stack/convox/9a10bbe0-51d5-11e5-b85a-5001dc3ed8d2</StackId>
        <StackStatus>CREATE_COMPLETE</StackStatus>
        <StackName>convox</StackName>
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
            <ParameterValue>chris@convox.com</ParameterValue>
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
		},
	}))

	aws.DefaultConfig.Region = "test"
	aws.DefaultConfig.Endpoint = s.URL
	os.Setenv("RACK", "convox-test")
	return
}

func TestBasicAuth(t *testing.T) {
	defer func(p string) {
		os.Setenv("PASSWORD", p)
	}(os.Getenv("PASSWORD"))

	server := stubSystem()
	defer server.Close()

	os.Setenv("PASSWORD", "keymaster")
	req, _ := http.NewRequest("GET", "http://convox/system", nil)
	w := httptest.NewRecorder()
	controllers.SingleRequest(w, req)

	if w.Code != 401 {
		t.Errorf("expected status code of %d, got %d", 401, w.Code)
		return
	}

	w = httptest.NewRecorder()
	req.SetBasicAuth("", "keymaster")
	controllers.SingleRequest(w, req)

	if w.Code != 200 {
		t.Errorf("expected status code of %d, got %d", 200, w.Code)
		return
	}
}
