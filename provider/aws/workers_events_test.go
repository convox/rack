package aws

import (
	"errors"
	"testing"

	"github.com/convox/rack/pkg/test/awsutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var cycleRackDescribeStacks = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "POST",
		RequestURI: "/",
		Body:       `Action=DescribeStacks&StackName=convox&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
			<DescribeStacksResult>
				<Stacks>
					<member>
						<StackName>convox</StackName>
						<StackStatus>UPDATE_COMPLETE</StackStatus>
					</member>
				</Stacks>
			</DescribeStacksResult>
		</DescribeStacksResponse>`,
	},
}

var cycleRackListStackResourcesNoLogGroup = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "POST",
		RequestURI: "/",
		Body:       `Action=ListStackResources&StackName=convox&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `<ListStackResourcesResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
			<ListStackResourcesResult>
				<StackResourceSummaries>
					<member>
						<LogicalResourceId>Cluster</LogicalResourceId>
						<PhysicalResourceId>convox-Cluster-QLZBS9RCPJ3T</PhysicalResourceId>
						<ResourceType>AWS::ECS::Cluster</ResourceType>
						<ResourceStatus>CREATE_COMPLETE</ResourceStatus>
					</member>
				</StackResourceSummaries>
			</ListStackResourcesResult>
		</ListStackResourcesResponse>`,
	},
}

var cycleAppListStackResourcesNoLogGroup = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "POST",
		RequestURI: "/",
		Body:       `Action=ListStackResources&StackName=convox-httpd&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `<ListStackResourcesResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
			<ListStackResourcesResult>
				<StackResourceSummaries>
					<member>
						<LogicalResourceId>ServiceWeb</LogicalResourceId>
						<PhysicalResourceId>arn:aws:ecs:us-test-1:901416387788:service/convox-httpd-ServiceWeb</PhysicalResourceId>
						<ResourceType>AWS::ECS::Service</ResourceType>
						<ResourceStatus>CREATE_COMPLETE</ResourceStatus>
					</member>
				</StackResourceSummaries>
			</ListStackResourcesResult>
		</ListStackResourcesResponse>`,
	},
}

var cycleRackListStackResourcesLogGroup = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "POST",
		RequestURI: "/",
		Body:       `Action=ListStackResources&StackName=convox&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `<ListStackResourcesResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
			<ListStackResourcesResult>
				<StackResourceSummaries>
					<member>
						<LogicalResourceId>LogGroup</LogicalResourceId>
						<PhysicalResourceId>convox-LogGroup-1LO7CKS3HFDN9</PhysicalResourceId>
						<ResourceType>AWS::Logs::LogGroup</ResourceType>
						<ResourceStatus>CREATE_COMPLETE</ResourceStatus>
					</member>
				</StackResourceSummaries>
			</ListStackResourcesResult>
		</ListStackResourcesResponse>`,
	},
}

// A rack with LogDriver=Syslog has no LogGroup, and the not-found fallback used to
// re-enter with the same stack name until the monitor's goroutine stack blew up.
func TestGetStackLogGroupRackWithoutLogGroup(t *testing.T) {
	p, bodies := stubProvider(t, cycleRackDescribeStacks, cycleRackListStackResourcesNoLogGroup)

	group, err := p.getStackLogGroup("convox")

	require.Error(t, err)
	assert.True(t, errors.Is(err, errNoLogGroup))
	assert.Empty(t, group)
	assert.Len(t, *bodies, 2)
}

func TestGetStackLogGroupFallsBackToRack(t *testing.T) {
	p, bodies := stubProvider(t,
		cycleBuildStuckDescribeStacks,
		cycleAppListStackResourcesNoLogGroup,
		cycleRackDescribeStacks,
		cycleRackListStackResourcesLogGroup,
	)

	group, err := p.getStackLogGroup("convox-httpd")

	require.NoError(t, err)
	assert.Equal(t, "convox-LogGroup-1LO7CKS3HFDN9", group)
	assert.Len(t, *bodies, 4)
}
