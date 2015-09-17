package test

import "github.com/convox/rack/api/awsutil"

func CreateAppStackCycle(appName string) awsutil.Cycle {
	return awsutil.Cycle{
		awsutil.Request{"/", "", createAppUrlBody()},
		awsutil.Response{200, ""},
	}
}

func CreateAppStackExistsCycle(appName string) awsutil.Cycle {
	return awsutil.Cycle{
		awsutil.Request{"/", "", createAppUrlBody()},
		awsutil.Response{
			400,
			`<ErrorResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
  <Error>
    <Type>Sender</Type>
    <Code>AlreadyExistsException</Code>
    <Message>Stack with id ` + appName + ` already exists</Message>
  </Error>
  <RequestId>bc91dc86-5803-11e5-a24f-85fde26a90fa</RequestId>
</ErrorResponse>`,
		},
	}
}

// returns the stack you asked for
func DescribeAppStackCycle(stackName string) awsutil.Cycle {
	return awsutil.Cycle{
		awsutil.Request{"/", "", `Action=DescribeStacks&StackName=` + stackName + `&Version=2010-05-15`},
		awsutil.Response{200,
			` <DescribeStacksResult><Stacks>` + appStackXML(stackName) + `</Stacks></DescribeStacksResult>`},
	}
}

// no filter - returns convox stack and an app
func DescribeStackCycleWithoutQuery(appName string) awsutil.Cycle {
	xml := appStackXML(appName) + convoxStackXML("convox")

	return awsutil.Cycle{
		awsutil.Request{"/", "", `Action=DescribeStacks&Version=2010-05-15`},
		awsutil.Response{200, ` <DescribeStacksResult><Stacks>` + xml + `</Stacks></DescribeStacksResult>`},
	}
}

// returns convox stack
func DescribeConvoxStackCycle(stackName string) awsutil.Cycle {
	return awsutil.Cycle{
		awsutil.Request{"/", "", `Action=DescribeStacks&StackName=` + stackName + `&Version=2010-05-15`},
		awsutil.Response{200,
			` <DescribeStacksResult><Stacks>` + convoxStackXML(stackName) + `</Stacks></DescribeStacksResult>`},
	}
}

func DeleteStackCycle(stackName string) awsutil.Cycle {
	return awsutil.Cycle{
		awsutil.Request{"/", "", `Action=DeleteStack&StackName=` + stackName + `&Version=2010-05-15`},
		awsutil.Response{200, ""},
	}
}

// search for stack, return missing
func DescribeStackNotFound(stackName string) awsutil.Cycle {
	return awsutil.Cycle{
		awsutil.Request{"/", "", `Action=DescribeStacks&StackName=` + stackName + `&Version=2010-05-15`},
		awsutil.Response{
			400,
			`<ErrorResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
  <Error>
    <Type>Sender</Type>
    <Code>ValidationError</Code>
    <Message>Stack with id ` + stackName + ` does not exist</Message>
  </Error>
  <RequestId>bc91dc86-5803-11e5-a24f-85fde26a90fa</RequestId>
</ErrorResponse>`,
		},
	}
}

func convoxStackXML(stackName string) string {
	return `
      <member>
        <Tags/>
        <StackId>arn:aws:cloudformation:us-east-1:938166070011:stack/` + stackName + `/9a10bbe0-51d5-11e5-b85a-5001dc3ed8d2</StackId>
        <StackStatus>CREATE_COMPLETE</StackStatus>
        <StackName>` + stackName + `</StackName>
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
`

}

func appStackXML(appName string) string {

	return `
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
            <OutputValue>apache-app-2311461.test.elb.amazonaws.com</OutputValue>
            <OutputKey>BalancerHost</OutputKey>
          </member>
          <member>
            <OutputValue>apache-app-Kinesis-6OTFWDVFK9BB</OutputValue>
            <OutputKey>Kinesis</OutputKey>
          </member>
          <member>
            <OutputValue>80</OutputValue>
            <OutputKey>MainPort80Balancer</OutputKey>
          </member>
          <member>
            <OutputValue>apache-app-settings-2gkjc9lf123nm</OutputValue>
            <OutputKey>Settings</OutputKey>
          </member>
        </Outputs>
      </member>`

}

// NOTE: app stack paramter serialization does not guarantee order,
// 			 so even the same source object is not guaranteed to serialize
//       correctly for comparison.
func createAppUrlBody() string {
	return `/^Action=CreateStack/`
}
