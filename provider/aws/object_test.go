package aws_test

import "github.com/convox/rack/test/awsutil"

var cycleObjectDescribeStackResources = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "",
		Body:       `Action=DescribeStackResources&StackName=convox-httpd&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<DescribeStackResourcesResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
  <DescribeStackResourcesResult>
    <StackResources>
    <member>
      <PhysicalResourceId>convox-httpd-settings-139bidzalmbtu</PhysicalResourceId>
      <ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
      <LogicalResourceId>Settings</LogicalResourceId>
      <Timestamp>2016-10-22T02:53:23.817Z</Timestamp>
      <ResourceType>AWS::Logs::LogGroup</ResourceType>
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
