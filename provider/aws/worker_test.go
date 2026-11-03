package aws

import (
	"os"
	"testing"

	"github.com/convox/rack/pkg/test/awsutil"
	"github.com/stretchr/testify/require"
)

func TestTry(t *testing.T) {
	t.Skip()

	p := &Provider{
		Region:              os.Getenv("AWS_REGION"),
		Private:             false,
		WhiteListSpecified:  true,
		ApiBalancerSecurity: "sg-0042ec8049da65c75",
		Rack:                "v2-rack219",
	}

	err := p.SyncInstancesIpInSecurityGroup()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSyncInstancesIpMoreUserDefined(t *testing.T) {
	p, bodies := stubProvider(t,
		awsutil.Cycle{
			Request:  awsutil.Request{RequestURI: "/", Body: "/Action=DescribeInstances/"},
			Response: awsutil.Response{StatusCode: 200, Body: `<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/"><reservationSet><item><instancesSet><item><instanceId>i-1</instanceId><ipAddress>1.2.3.4</ipAddress></item></instancesSet></item></reservationSet></DescribeInstancesResponse>`},
		},
		awsutil.Cycle{
			Request:  awsutil.Request{RequestURI: "/", Body: "/Action=DescribeSecurityGroups/"},
			Response: awsutil.Response{StatusCode: 200, Body: `<DescribeSecurityGroupsResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/"><securityGroupInfo><item><groupId>sg-1</groupId><ipPermissions><item><ipProtocol>tcp</ipProtocol><fromPort>443</fromPort><toPort>443</toPort><ipRanges><item><cidrIp>1.2.3.4/32</cidrIp></item><item><cidrIp>5.6.7.8/32</cidrIp></item></ipRanges></item></ipPermissions></item></securityGroupInfo></DescribeSecurityGroupsResponse>`},
		},
	)

	p.ApiBalancerSecurity = "sg-1"
	p.InstancesIpToIncludInWhiteListing = "Both"
	p.WhiteListSpecified = true

	require.NotPanics(t, func() {
		require.NoError(t, p.SyncInstancesIpInSecurityGroup())
	})
	require.Len(t, *bodies, 2)
}
