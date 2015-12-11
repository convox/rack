package test

import (
	"net/http/httptest"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/convox/rack/api/awsutil"
)

func init() {
	region := "test"
	defaults.DefaultConfig.Region = &region
}

/*
Create a test server that mocks an AWS request/response cycle,
suitable for a single test

Example:
		s := stubAws(DescribeStackCycleWithoutQuery("bar"))
		defer s.Close()
*/
func StubAws(cycles ...awsutil.Cycle) (s *httptest.Server) {
	handler := awsutil.NewHandler(cycles)
	s = httptest.NewServer(handler)
	defaults.DefaultConfig.Endpoint = &s.URL
	return s
}

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
			` <DescribeStacksResult><Stacks>` + appStackXML(stackName, "CREATE_COMPLETE") + `</Stacks></DescribeStacksResult>`},
	}
}

// returns the stack you asked for with a status
func DescribeAppStatusStackCycle(stackName string, status string) awsutil.Cycle {
	return awsutil.Cycle{
		awsutil.Request{"/", "", `Action=DescribeStacks&StackName=` + stackName + `&Version=2010-05-15`},
		awsutil.Response{200,
			` <DescribeStacksResult><Stacks>` + appStackXML(stackName, status) + `</Stacks></DescribeStacksResult>`},
	}
}

func DescribeContainerInstancesCycle(clusterName string) awsutil.Cycle {
	return awsutil.Cycle{
		awsutil.Request{"/", "AmazonEC2ContainerServiceV20141113.DescribeContainerInstances",
			`{"cluster":"` + clusterName + `",
				"containerInstances": [
					"arn:aws:ecs:us-east-1:938166070011:container-instance/0ac4bb1c-be98-4202-a9c1-03153e91c05e",
					"arn:aws:ecs:us-east-1:938166070011:container-instance/38a59629-6f5d-4d02-8733-fdb49500ae45",
					"arn:aws:ecs:us-east-1:938166070011:container-instance/e7c311ae-968f-4125-8886-f9b724860d4c"
				]
		}`},
		awsutil.Response{200, describeContainerInstancesResponse()},
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

func DescribeInstancesCycle() awsutil.Cycle {
	return awsutil.Cycle{
		awsutil.Request{"/", "", `Action=DescribeInstances&Filter.1.Name=instance-id&Filter.1.Value.1=i-4a5513f4&Filter.1.Value.2=i-3963798e&Filter.1.Value.3=i-c6a72b76&Version=2015-10-01`},
		awsutil.Response{200, describeInstancesResponse()},
	}
}

// no filter - returns convox stack and an app
func DescribeStackCycleWithoutQuery(appName string) awsutil.Cycle {
	xml := appStackXML(appName, "CREATE_COMPLETE") + convoxStackXML("convox")

	return awsutil.Cycle{
		awsutil.Request{"/", "", `Action=DescribeStacks&Version=2010-05-15`},
		awsutil.Response{200, ` <DescribeStacksResult><Stacks>` + xml + `</Stacks></DescribeStacksResult>`},
	}
}

func DeleteInstanceCycle(instance string) awsutil.Cycle {
	return awsutil.Cycle{
		awsutil.Request{"/", "", `Action=TerminateInstances&InstanceId.1=` + instance + `&Version=2015-10-01`},
		awsutil.Response{200, ""},
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

func ListContainerInstancesCycle(clusterName string) awsutil.Cycle {
	return awsutil.Cycle{
		awsutil.Request{"/", "AmazonEC2ContainerServiceV20141113.ListContainerInstances",
			`{"cluster":"` + clusterName + `"}`},
		awsutil.Response{200,
			`{"containerInstanceArns":["arn:aws:ecs:us-east-1:938166070011:container-instance/0ac4bb1c-be98-4202-a9c1-03153e91c05e","arn:aws:ecs:us-east-1:938166070011:container-instance/38a59629-6f5d-4d02-8733-fdb49500ae45","arn:aws:ecs:us-east-1:938166070011:container-instance/e7c311ae-968f-4125-8886-f9b724860d4c"]}`},
	}
}

func ListTasksCycle(clusterName string) awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/",
			Operation:  "AmazonEC2ContainerServiceV20141113.ListTasks",
			Body:       `{"cluster":"` + clusterName + `"}`,
		},
		Response: awsutil.Response{
			StatusCode: 200,
			Body:       `{"taskArns":["arn:aws:ecs:us-east-1:901416387788:task/320a8b6a-c243-47d3-a1d1-6db5dfcb3f58"]}`,
		},
	}
}

func DescribeTasksCycle(clusterName string) awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/",
			Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTasks",
			Body:       `{"cluster":"` + clusterName + `","tasks":["arn:aws:ecs:us-east-1:901416387788:task/320a8b6a-c243-47d3-a1d1-6db5dfcb3f58"]}`,
		},
		Response: awsutil.Response{
			StatusCode: 200,
			Body:       `{"tasks":[{"containerInstanceArn":"arn:aws:ecs:us-east-1:901416387788:container-instance/0ac4bb1c-be98-4202-a9c1-03153e91c05e","containers":[{"containerArn":"arn:aws:ecs:us-east-1:901416387788:container/821cc6e1-b120-422c-9092-4932cce0897b","name":"worker"}], "taskArn":"arn:aws:ecs:us-east-1:901416387788:task/320a8b6a-c243-47d3-a1d1-6db5dfcb3f58","taskDefinitionArn":"arn:aws:ecs:us-east-1:901416387788:task-definition/myapp-staging-worker:3","lastStatus":"RUNNING"}]}`,
		},
	}
}

func DescribeTaskDefinitionCycle(clusterName string) awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/",
			Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition",
			Body:       `{"taskDefinition":"arn:aws:ecs:us-east-1:901416387788:task-definition/myapp-staging-worker:3"}`,
		},
		Response: awsutil.Response{
			StatusCode: 200,
			Body:       `{"taskDefinition":{"volumes":[{"host":{"sourcePath":"/var/run/docker.sock"},"name":"myapp-staging-0-0"}],"containerDefinitions":[{"name":"worker","cpu":200,"memory":256,"image":"test-image","environment":[{"name":"PROCESS","value":"worker"}],"mountPoints":[{"sourceVolume":"worker-0-0","readOnly":false,"containerPath":"/var/run/docker.sock"}]}],"family":"myapp-staging-worker"}}`,
		},
	}
}

func ListServicesCycle(clusterName string) awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/",
			Operation:  "AmazonEC2ContainerServiceV20141113.ListServices",
			Body:       `{"cluster":"` + clusterName + `"}`,
		},
		Response: awsutil.Response{
			StatusCode: 200,
			Body:       `{"serviceArns":["arn:aws:ecs:us-west-2:901416387788:service/myapp-staging-worker-SCELGCIYSKF"]}`,
		},
	}
}

func DescribeServicesCycle(clusterName string) awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/",
			Operation:  "AmazonEC2ContainerServiceV20141113.DescribeServices",
			Body:       `{"cluster":"` + clusterName + `", "services":["arn:aws:ecs:us-west-2:901416387788:service/myapp-staging-worker-SCELGCIYSKF"]}`,
		},
		Response: awsutil.Response{
			StatusCode: 200,
			Body: `
{
    "services": [
        {
            "status": "ACTIVE", 
            "taskDefinition": "arn:aws:ecs:us-west-2:901416387788:task-definition/httpd-web:36", 
            "pendingCount": 0, 
            "loadBalancers": [
                {
                    "containerName": "web", 
                    "containerPort": 80, 
                    "loadBalancerName": "httpd"
                }
            ], 
            "roleArn": "arn:aws:iam::901416387788:role/httpd-ServiceRole-1HNRHXNKGNLT9", 
            "desiredCount": 2, 
            "serviceName": "httpd-web-SCELGCIYSKF", 
            "clusterArn": "arn:aws:ecs:us-west-2:901416387788:cluster/convox-Cluster-1NCWX9EC0JOV4", 
            "serviceArn": "arn:aws:ecs:us-west-2:901416387788:service/httpd-web-SCELGCIYSKF", 
            "deployments": [
                {
                    "status": "PRIMARY", 
                    "pendingCount": 0, 
                    "createdAt": 1449559137.768, 
                    "desiredCount": 2, 
                    "taskDefinition": "arn:aws:ecs:us-west-2:901416387788:task-definition/httpd-web:36", 
                    "updatedAt": 1449559137.768, 
                    "id": "ecs-svc/9223370587295638039", 
                    "runningCount": 1
                }, 
                {
                    "status": "ACTIVE", 
                    "pendingCount": 0, 
                    "createdAt": 1449511658.683, 
                    "desiredCount": 2, 
                    "taskDefinition": "arn:aws:ecs:us-west-2:901416387788:task-definition/httpd-web:33", 
                    "updatedAt": 1449511869.412, 
                    "id": "ecs-svc/9223370587343117124", 
                    "runningCount": 1
                }
            ], 
            "events": [
                {
                    "message": "(service httpd-web-SCELGCIYSKF) was unable to place a task because no container instance met all of its requirements. The closest matching (container-instance b1a73168-f8a6-4ed9-b69e-94adc7a0f1e0) has insufficient memory available. For more information, see the Troubleshooting section of the Amazon ECS Developer Guide.", 
                    "id": "3890020b-7e55-4d25-9694-ba823cc34822", 
                    "createdAt": 1449760390.037
                },
                {
                    "message": "(service httpd-web-SCELGCIYSKF) has started 1 tasks: (task f120ddee-5aa5-434e-b765-30503080078b).", 
                    "id": "d84b8245-9653-453f-a449-27d7c7cfdc0a", 
                    "createdAt": 1449003339.092
                }
            ], 
            "runningCount": 2
        }
    ], 
    "failures": []
}`,
		},
	}
}

func DescribeContainerInstancesFilteredCycle(clusterName string) awsutil.Cycle {
	return awsutil.Cycle{
		awsutil.Request{"/", "AmazonEC2ContainerServiceV20141113.DescribeContainerInstances",
			`{"cluster":"` + clusterName + `",
        "containerInstances": [
          "arn:aws:ecs:us-east-1:901416387788:container-instance/0ac4bb1c-be98-4202-a9c1-03153e91c05e"
        ]
    }`},
		awsutil.Response{200, `{"containerInstances":[
  {"agentConnected":true,"containerInstanceArn":"arn:aws:ecs:us-east-1:938166070011:container-instance/0ac4bb1c-be98-4202-a9c1-03153e91c05e","ec2InstanceId":"i-4a5513f4","pendingTasksCount":0,"registeredResources":[{"doubleValue":0.0,"integerValue":1024,"longValue":0,"name":"CPU","type":"INTEGER"},{"doubleValue":0.0,"integerValue":2004,"longValue":0,"name":"MEMORY","type":"INTEGER"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS","stringSetValue":["22","2376","2375","51678"],"type":"STRINGSET"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS_UDP","stringSetValue":[],"type":"STRINGSET"}],"remainingResources":[{"doubleValue":0.0,"integerValue":1024,"longValue":0,"name":"CPU","type":"INTEGER"},{"doubleValue":0.0,"integerValue":2004,"longValue":0,"name":"MEMORY","type":"INTEGER"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS","stringSetValue":["22","2376","2375","51678"],"type":"STRINGSET"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS_UDP","stringSetValue":[],"type":"STRINGSET"}],"runningTasksCount":0,"status":"ACTIVE","versionInfo":{"agentHash":"4ab1051","agentVersion":"1.4.0","dockerVersion":"DockerVersion: 1.7.1"}}
],"failures":[]}`},
	}
}

func DescribeInstancesFilteredCycle() awsutil.Cycle {
	return awsutil.Cycle{
		awsutil.Request{"/", "", `Action=DescribeInstances&Filter.1.Name=instance-id&Filter.1.Value.1=i-4a5513f4&Version=2015-10-01`},
		awsutil.Response{200, `<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2015-10-01/">
    <requestId>f215b40f-5a0c-4fe6-9624-657cd1f4ef6b</requestId>
    <reservationSet>
        <item>
            <reservationId>r-8d7e2072</reservationId>
            <ownerId>938166070011</ownerId>
            <groupSet/>
            <instancesSet>
                <item>
                    <instanceId>i-c6a72b76</instanceId>
                    <imageId>ami-c5fa5aae</imageId>
                    <instanceState>
                        <code>16</code>
                        <name>running</name>
                    </instanceState>
                    <privateDnsName>ip-10-0-3-248.ec2.internal</privateDnsName>
                    <dnsName/>
                    <reason/>
                    <amiLaunchIndex>0</amiLaunchIndex>
                    <productCodes/>
                    <instanceType>t2.small</instanceType>
                    <launchTime>2015-11-19T02:59:53.000Z</launchTime>
                    <placement>
                        <availabilityZone>us-east-1c</availabilityZone>
                        <groupName/>
                        <tenancy>default</tenancy>
                    </placement>
                    <monitoring>
                        <state>enabled</state>
                    </monitoring>
                    <subnetId>subnet-21bab178</subnetId>
                    <vpcId>vpc-e948f08d</vpcId>
                    <privateIpAddress>10.0.3.248</privateIpAddress>
                    <ipAddress>52.71.252.224</ipAddress>
                    <sourceDestCheck>true</sourceDestCheck>
                    <groupSet>
                        <item>
                            <groupId>sg-31188d57</groupId>
                            <groupName>convox-dev-SecurityGroup-VZKZ1CGI51J4</groupName>
                        </item>
                    </groupSet>
                    <architecture>x86_64</architecture>
                    <rootDeviceType>ebs</rootDeviceType>
                    <rootDeviceName>/dev/xvda</rootDeviceName>
                    <blockDeviceMapping>
                        <item>
                            <deviceName>/dev/xvda</deviceName>
                            <ebs>
                                <volumeId>vol-dfb94422</volumeId>
                                <status>attached</status>
                                <attachTime>2015-11-19T02:59:56.000Z</attachTime>
                                <deleteOnTermination>true</deleteOnTermination>
                            </ebs>
                        </item>
                    </blockDeviceMapping>
                    <virtualizationType>hvm</virtualizationType>
                    <clientToken>2a49b8e3-6ed5-49f0-a62e-904c43347933_subnet-21bab178_1</clientToken>
                    <tagSet>
                        <item>
                            <key>Name</key>
                            <value>convox-dev</value>
                        </item>
                        <item>
                            <key>aws:cloudformation:stack-id</key>
                            <value>arn:aws:cloudformation:us-east-1:938166070011:stack/convox-dev/538ae350-8815-11e5-8a2d-5001b34fc89a</value>
                        </item>
                        <item>
                            <key>Rack</key>
                            <value>convox-dev</value>
                        </item>
                        <item>
                            <key>aws:autoscaling:groupName</key>
                            <value>convox-dev-Instances-1QUKKS9PIP4BS</value>
                        </item>
                        <item>
                            <key>aws:cloudformation:logical-id</key>
                            <value>Instances</value>
                        </item>
                        <item>
                            <key>aws:cloudformation:stack-name</key>
                            <value>convox-dev</value>
                        </item>
                    </tagSet>
                    <hypervisor>xen</hypervisor>
                    <networkInterfaceSet>
                        <item>
                            <networkInterfaceId>eni-6b4b7637</networkInterfaceId>
                            <subnetId>subnet-21bab178</subnetId>
                            <vpcId>vpc-e948f08d</vpcId>
                            <description/>
                            <ownerId>938166070011</ownerId>
                            <status>in-use</status>
                            <macAddress>0e:d6:3e:c3:21:15</macAddress>
                            <privateIpAddress>10.0.3.248</privateIpAddress>
                            <sourceDestCheck>true</sourceDestCheck>
                            <groupSet>
                                <item>
                                    <groupId>sg-31188d57</groupId>
                                    <groupName>convox-dev-SecurityGroup-VZKZ1CGI51J4</groupName>
                                </item>
                            </groupSet>
                            <attachment>
                                <attachmentId>eni-attach-d99f0c34</attachmentId>
                                <deviceIndex>0</deviceIndex>
                                <status>attached</status>
                                <attachTime>2015-11-19T02:59:53.000Z</attachTime>
                                <deleteOnTermination>true</deleteOnTermination>
                            </attachment>
                            <association>
                                <publicIp>52.71.252.224</publicIp>
                                <publicDnsName/>
                                <ipOwnerId>amazon</ipOwnerId>
                            </association>
                            <privateIpAddressesSet>
                                <item>
                                    <privateIpAddress>10.0.3.248</privateIpAddress>
                                    <primary>true</primary>
                                    <association>
                                    <publicIp>52.71.252.224</publicIp>
                                    <publicDnsName/>
                                    <ipOwnerId>amazon</ipOwnerId>
                                    </association>
                                </item>
                            </privateIpAddressesSet>
                        </item>
                    </networkInterfaceSet>
                    <iamInstanceProfile>
                        <arn>arn:aws:iam::938166070011:instance-profile/convox-dev-InstanceProfile-HJBF2SIK0R6W</arn>
                        <id>AIPAIR7O43WTX246KVAIM</id>
                    </iamInstanceProfile>
                    <ebsOptimized>false</ebsOptimized>
                </item>
            </instancesSet>
            <requesterId>226008221399</requesterId>
        </item>
    </reservationSet>
</DescribeInstancesResponse>`},
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

func appStackXML(appName string, status string) string {

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
        <StackStatus>` + status + `</StackStatus>
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

func describeInstancesResponse() string {
	return `<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2015-10-01/">
    <requestId>f215b40f-5a0c-4fe6-9624-657cd1f4ef6b</requestId>
    <reservationSet>
        <item>
            <reservationId>r-8d7e2072</reservationId>
            <ownerId>938166070011</ownerId>
            <groupSet/>
            <instancesSet>
                <item>
                    <instanceId>i-c6a72b76</instanceId>
                    <imageId>ami-c5fa5aae</imageId>
                    <instanceState>
                        <code>16</code>
                        <name>running</name>
                    </instanceState>
                    <privateDnsName>ip-10-0-3-248.ec2.internal</privateDnsName>
                    <dnsName/>
                    <reason/>
                    <amiLaunchIndex>0</amiLaunchIndex>
                    <productCodes/>
                    <instanceType>t2.small</instanceType>
                    <launchTime>2015-11-19T02:59:53.000Z</launchTime>
                    <placement>
                        <availabilityZone>us-east-1c</availabilityZone>
                        <groupName/>
                        <tenancy>default</tenancy>
                    </placement>
                    <monitoring>
                        <state>enabled</state>
                    </monitoring>
                    <subnetId>subnet-21bab178</subnetId>
                    <vpcId>vpc-e948f08d</vpcId>
                    <privateIpAddress>10.0.3.248</privateIpAddress>
                    <ipAddress>52.71.252.224</ipAddress>
                    <sourceDestCheck>true</sourceDestCheck>
                    <groupSet>
                        <item>
                            <groupId>sg-31188d57</groupId>
                            <groupName>convox-dev-SecurityGroup-VZKZ1CGI51J4</groupName>
                        </item>
                    </groupSet>
                    <architecture>x86_64</architecture>
                    <rootDeviceType>ebs</rootDeviceType>
                    <rootDeviceName>/dev/xvda</rootDeviceName>
                    <blockDeviceMapping>
                        <item>
                            <deviceName>/dev/xvda</deviceName>
                            <ebs>
                                <volumeId>vol-dfb94422</volumeId>
                                <status>attached</status>
                                <attachTime>2015-11-19T02:59:56.000Z</attachTime>
                                <deleteOnTermination>true</deleteOnTermination>
                            </ebs>
                        </item>
                    </blockDeviceMapping>
                    <virtualizationType>hvm</virtualizationType>
                    <clientToken>2a49b8e3-6ed5-49f0-a62e-904c43347933_subnet-21bab178_1</clientToken>
                    <tagSet>
                        <item>
                            <key>Name</key>
                            <value>convox-dev</value>
                        </item>
                        <item>
                            <key>aws:cloudformation:stack-id</key>
                            <value>arn:aws:cloudformation:us-east-1:938166070011:stack/convox-dev/538ae350-8815-11e5-8a2d-5001b34fc89a</value>
                        </item>
                        <item>
                            <key>Rack</key>
                            <value>convox-dev</value>
                        </item>
                        <item>
                            <key>aws:autoscaling:groupName</key>
                            <value>convox-dev-Instances-1QUKKS9PIP4BS</value>
                        </item>
                        <item>
                            <key>aws:cloudformation:logical-id</key>
                            <value>Instances</value>
                        </item>
                        <item>
                            <key>aws:cloudformation:stack-name</key>
                            <value>convox-dev</value>
                        </item>
                    </tagSet>
                    <hypervisor>xen</hypervisor>
                    <networkInterfaceSet>
                        <item>
                            <networkInterfaceId>eni-6b4b7637</networkInterfaceId>
                            <subnetId>subnet-21bab178</subnetId>
                            <vpcId>vpc-e948f08d</vpcId>
                            <description/>
                            <ownerId>938166070011</ownerId>
                            <status>in-use</status>
                            <macAddress>0e:d6:3e:c3:21:15</macAddress>
                            <privateIpAddress>10.0.3.248</privateIpAddress>
                            <sourceDestCheck>true</sourceDestCheck>
                            <groupSet>
                                <item>
                                    <groupId>sg-31188d57</groupId>
                                    <groupName>convox-dev-SecurityGroup-VZKZ1CGI51J4</groupName>
                                </item>
                            </groupSet>
                            <attachment>
                                <attachmentId>eni-attach-d99f0c34</attachmentId>
                                <deviceIndex>0</deviceIndex>
                                <status>attached</status>
                                <attachTime>2015-11-19T02:59:53.000Z</attachTime>
                                <deleteOnTermination>true</deleteOnTermination>
                            </attachment>
                            <association>
                                <publicIp>52.71.252.224</publicIp>
                                <publicDnsName/>
                                <ipOwnerId>amazon</ipOwnerId>
                            </association>
                            <privateIpAddressesSet>
                                <item>
                                    <privateIpAddress>10.0.3.248</privateIpAddress>
                                    <primary>true</primary>
                                    <association>
                                    <publicIp>52.71.252.224</publicIp>
                                    <publicDnsName/>
                                    <ipOwnerId>amazon</ipOwnerId>
                                    </association>
                                </item>
                            </privateIpAddressesSet>
                        </item>
                    </networkInterfaceSet>
                    <iamInstanceProfile>
                        <arn>arn:aws:iam::938166070011:instance-profile/convox-dev-InstanceProfile-HJBF2SIK0R6W</arn>
                        <id>AIPAIR7O43WTX246KVAIM</id>
                    </iamInstanceProfile>
                    <ebsOptimized>false</ebsOptimized>
                </item>
            </instancesSet>
            <requesterId>226008221399</requesterId>
        </item>
        <item>
            <reservationId>r-835b392a</reservationId>
            <ownerId>938166070011</ownerId>
            <groupSet/>
            <instancesSet>
                <item>
                    <instanceId>i-4a5513f4</instanceId>
                    <imageId>ami-c5fa5aae</imageId>
                    <instanceState>
                        <code>16</code>
                        <name>running</name>
                    </instanceState>
                    <privateDnsName>ip-10-0-1-182.ec2.internal</privateDnsName>
                    <dnsName/>
                    <reason/>
                    <amiLaunchIndex>0</amiLaunchIndex>
                    <productCodes/>
                    <instanceType>t2.small</instanceType>
                    <launchTime>2015-11-25T20:41:12.000Z</launchTime>
                    <placement>
                        <availabilityZone>us-east-1a</availabilityZone>
                        <groupName/>
                        <tenancy>default</tenancy>
                    </placement>
                    <monitoring>
                        <state>enabled</state>
                    </monitoring>
                    <subnetId>subnet-97ab91bc</subnetId>
                    <vpcId>vpc-e948f08d</vpcId>
                    <privateIpAddress>10.0.1.182</privateIpAddress>
                    <ipAddress>54.208.61.75</ipAddress>
                    <sourceDestCheck>true</sourceDestCheck>
                    <groupSet>
                        <item>
                            <groupId>sg-31188d57</groupId>
                            <groupName>convox-dev-SecurityGroup-VZKZ1CGI51J4</groupName>
                        </item>
                    </groupSet>
                    <architecture>x86_64</architecture>
                    <rootDeviceType>ebs</rootDeviceType>
                    <rootDeviceName>/dev/xvda</rootDeviceName>
                    <blockDeviceMapping>
                        <item>
                            <deviceName>/dev/xvda</deviceName>
                            <ebs>
                                <volumeId>vol-98ad7e77</volumeId>
                                <status>attached</status>
                                <attachTime>2015-11-25T20:41:15.000Z</attachTime>
                                <deleteOnTermination>true</deleteOnTermination>
                            </ebs>
                        </item>
                    </blockDeviceMapping>
                    <virtualizationType>hvm</virtualizationType>
                    <clientToken>7706163a-d190-4500-bb19-18850f687730_subnet-97ab91bc_1</clientToken>
                    <tagSet>
                        <item>
                            <key>Rack</key>
                            <value>convox-dev</value>
                        </item>
                        <item>
                            <key>aws:cloudformation:stack-id</key>
                            <value>arn:aws:cloudformation:us-east-1:938166070011:stack/convox-dev/538ae350-8815-11e5-8a2d-5001b34fc89a</value>
                        </item>
                        <item>
                            <key>Name</key>
                            <value>convox-dev</value>
                        </item>
                        <item>
                            <key>aws:cloudformation:stack-name</key>
                            <value>convox-dev</value>
                        </item>
                        <item>
                            <key>aws:autoscaling:groupName</key>
                            <value>convox-dev-Instances-1QUKKS9PIP4BS</value>
                        </item>
                        <item>
                            <key>aws:cloudformation:logical-id</key>
                            <value>Instances</value>
                        </item>
                    </tagSet>
                    <hypervisor>xen</hypervisor>
                    <networkInterfaceSet>
                        <item>
                            <networkInterfaceId>eni-f5bf25d5</networkInterfaceId>
                            <subnetId>subnet-97ab91bc</subnetId>
                            <vpcId>vpc-e948f08d</vpcId>
                            <description/>
                            <ownerId>938166070011</ownerId>
                            <status>in-use</status>
                            <macAddress>12:51:78:a6:f5:13</macAddress>
                            <privateIpAddress>10.0.1.182</privateIpAddress>
                            <sourceDestCheck>true</sourceDestCheck>
                            <groupSet>
                                <item>
                                    <groupId>sg-31188d57</groupId>
                                    <groupName>convox-dev-SecurityGroup-VZKZ1CGI51J4</groupName>
                                </item>
                            </groupSet>
                            <attachment>
                                <attachmentId>eni-attach-5d1cfdb3</attachmentId>
                                <deviceIndex>0</deviceIndex>
                                <status>attached</status>
                                <attachTime>2015-11-25T20:41:12.000Z</attachTime>
                                <deleteOnTermination>true</deleteOnTermination>
                            </attachment>
                            <association>
                                <publicIp>54.208.61.75</publicIp>
                                <publicDnsName/>
                                <ipOwnerId>amazon</ipOwnerId>
                            </association>
                            <privateIpAddressesSet>
                                <item>
                                    <privateIpAddress>10.0.1.182</privateIpAddress>
                                    <primary>true</primary>
                                    <association>
                                    <publicIp>54.208.61.75</publicIp>
                                    <publicDnsName/>
                                    <ipOwnerId>amazon</ipOwnerId>
                                    </association>
                                </item>
                            </privateIpAddressesSet>
                        </item>
                    </networkInterfaceSet>
                    <iamInstanceProfile>
                        <arn>arn:aws:iam::938166070011:instance-profile/convox-dev-InstanceProfile-HJBF2SIK0R6W</arn>
                        <id>AIPAIR7O43WTX246KVAIM</id>
                    </iamInstanceProfile>
                    <ebsOptimized>false</ebsOptimized>
                </item>
            </instancesSet>
            <requesterId>226008221399</requesterId>
        </item>
        <item>
            <reservationId>r-da7b7c0a</reservationId>
            <ownerId>938166070011</ownerId>
            <groupSet/>
            <instancesSet>
                <item>
                    <instanceId>i-3963798e</instanceId>
                    <imageId>ami-c5fa5aae</imageId>
                    <instanceState>
                        <code>16</code>
                        <name>running</name>
                    </instanceState>
                    <privateDnsName>ip-10-0-2-236.ec2.internal</privateDnsName>
                    <dnsName/>
                    <reason/>
                    <amiLaunchIndex>0</amiLaunchIndex>
                    <productCodes/>
                    <instanceType>t2.small</instanceType>
                    <launchTime>2015-11-24T17:35:49.000Z</launchTime>
                    <placement>
                        <availabilityZone>us-east-1b</availabilityZone>
                        <groupName/>
                        <tenancy>default</tenancy>
                    </placement>
                    <monitoring>
                        <state>enabled</state>
                    </monitoring>
                    <subnetId>subnet-8ff000f9</subnetId>
                    <vpcId>vpc-e948f08d</vpcId>
                    <privateIpAddress>10.0.2.236</privateIpAddress>
                    <ipAddress>54.85.115.31</ipAddress>
                    <sourceDestCheck>true</sourceDestCheck>
                    <groupSet>
                        <item>
                            <groupId>sg-31188d57</groupId>
                            <groupName>convox-dev-SecurityGroup-VZKZ1CGI51J4</groupName>
                        </item>
                    </groupSet>
                    <architecture>x86_64</architecture>
                    <rootDeviceType>ebs</rootDeviceType>
                    <rootDeviceName>/dev/xvda</rootDeviceName>
                    <blockDeviceMapping>
                        <item>
                            <deviceName>/dev/xvda</deviceName>
                            <ebs>
                                <volumeId>vol-a63bbc45</volumeId>
                                <status>attached</status>
                                <attachTime>2015-11-24T17:35:52.000Z</attachTime>
                                <deleteOnTermination>true</deleteOnTermination>
                            </ebs>
                        </item>
                    </blockDeviceMapping>
                    <virtualizationType>hvm</virtualizationType>
                    <clientToken>8f162830-4721-457d-bc2c-3ccbd96cb122_subnet-8ff000f9_1</clientToken>
                    <tagSet>
                        <item>
                            <key>aws:cloudformation:stack-id</key>
                            <value>arn:aws:cloudformation:us-east-1:938166070011:stack/convox-dev/538ae350-8815-11e5-8a2d-5001b34fc89a</value>
                        </item>
                        <item>
                            <key>aws:cloudformation:logical-id</key>
                            <value>Instances</value>
                        </item>
                        <item>
                            <key>aws:cloudformation:stack-name</key>
                            <value>convox-dev</value>
                        </item>
                        <item>
                            <key>aws:autoscaling:groupName</key>
                            <value>convox-dev-Instances-1QUKKS9PIP4BS</value>
                        </item>
                        <item>
                            <key>Rack</key>
                            <value>convox-dev</value>
                        </item>
                        <item>
                            <key>Name</key>
                            <value>convox-dev</value>
                        </item>
                    </tagSet>
                    <hypervisor>xen</hypervisor>
                    <networkInterfaceSet>
                        <item>
                            <networkInterfaceId>eni-9c18d0d0</networkInterfaceId>
                            <subnetId>subnet-8ff000f9</subnetId>
                            <vpcId>vpc-e948f08d</vpcId>
                            <description/>
                            <ownerId>938166070011</ownerId>
                            <status>in-use</status>
                            <macAddress>0a:2d:91:ea:29:49</macAddress>
                            <privateIpAddress>10.0.2.236</privateIpAddress>
                            <sourceDestCheck>true</sourceDestCheck>
                            <groupSet>
                                <item>
                                    <groupId>sg-31188d57</groupId>
                                    <groupName>convox-dev-SecurityGroup-VZKZ1CGI51J4</groupName>
                                </item>
                            </groupSet>
                            <attachment>
                                <attachmentId>eni-attach-dec85035</attachmentId>
                                <deviceIndex>0</deviceIndex>
                                <status>attached</status>
                                <attachTime>2015-11-24T17:35:49.000Z</attachTime>
                                <deleteOnTermination>true</deleteOnTermination>
                            </attachment>
                            <association>
                                <publicIp>54.85.115.31</publicIp>
                                <publicDnsName/>
                                <ipOwnerId>amazon</ipOwnerId>
                            </association>
                            <privateIpAddressesSet>
                                <item>
                                    <privateIpAddress>10.0.2.236</privateIpAddress>
                                    <primary>true</primary>
                                    <association>
                                    <publicIp>54.85.115.31</publicIp>
                                    <publicDnsName/>
                                    <ipOwnerId>amazon</ipOwnerId>
                                    </association>
                                </item>
                            </privateIpAddressesSet>
                        </item>
                    </networkInterfaceSet>
                    <iamInstanceProfile>
                        <arn>arn:aws:iam::938166070011:instance-profile/convox-dev-InstanceProfile-HJBF2SIK0R6W</arn>
                        <id>AIPAIR7O43WTX246KVAIM</id>
                    </iamInstanceProfile>
                    <ebsOptimized>false</ebsOptimized>
                </item>
            </instancesSet>
            <requesterId>226008221399</requesterId>
        </item>
    </reservationSet>
</DescribeInstancesResponse>`
}

func describeContainerInstancesResponse() string {
	return `{"containerInstances":[{"agentConnected":true,"containerInstanceArn":"arn:aws:ecs:us-east-1:938166070011:container-instance/0ac4bb1c-be98-4202-a9c1-03153e91c05e","ec2InstanceId":"i-4a5513f4","pendingTasksCount":0,"registeredResources":[{"doubleValue":0.0,"integerValue":1024,"longValue":0,"name":"CPU","type":"INTEGER"},{"doubleValue":0.0,"integerValue":2004,"longValue":0,"name":"MEMORY","type":"INTEGER"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS","stringSetValue":["22","2376","2375","51678"],"type":"STRINGSET"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS_UDP","stringSetValue":[],"type":"STRINGSET"}],"remainingResources":[{"doubleValue":0.0,"integerValue":1024,"longValue":0,"name":"CPU","type":"INTEGER"},{"doubleValue":0.0,"integerValue":2004,"longValue":0,"name":"MEMORY","type":"INTEGER"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS","stringSetValue":["22","2376","2375","51678"],"type":"STRINGSET"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS_UDP","stringSetValue":[],"type":"STRINGSET"}],"runningTasksCount":0,"status":"ACTIVE","versionInfo":{"agentHash":"4ab1051","agentVersion":"1.4.0","dockerVersion":"DockerVersion: 1.7.1"}},
{"agentConnected":true,"containerInstanceArn":"arn:aws:ecs:us-east-1:938166070011:container-instance/38a59629-6f5d-4d02-8733-fdb49500ae45","ec2InstanceId":"i-3963798e","pendingTasksCount":0,"registeredResources":[{"doubleValue":0.0,"integerValue":1024,"longValue":0,"name":"CPU","type":"INTEGER"},{"doubleValue":0.0,"integerValue":2004,"longValue":0,"name":"MEMORY","type":"INTEGER"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS","stringSetValue":["22","2376","2375","51678"],"type":"STRINGSET"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS_UDP","stringSetValue":[],"type":"STRINGSET"}],"remainingResources":[{"doubleValue":0.0,"integerValue":1024,"longValue":0,"name":"CPU","type":"INTEGER"},{"doubleValue":0.0,"integerValue":2004,"longValue":0,"name":"MEMORY","type":"INTEGER"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS","stringSetValue":["22","2376","2375","51678"],"type":"STRINGSET"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS_UDP","stringSetValue":[],"type":"STRINGSET"}],"runningTasksCount":0,"status":"ACTIVE","versionInfo":{"agentHash":"4ab1051","agentVersion":"1.4.0","dockerVersion":"DockerVersion: 1.7.1"}},
{"agentConnected":true,"containerInstanceArn":"arn:aws:ecs:us-east-1:938166070011:container-instance/e7c311ae-968f-4125-8886-f9b724860d4c","ec2InstanceId":"i-c6a72b76","pendingTasksCount":0,"registeredResources":[{"doubleValue":0.0,"integerValue":1024,"longValue":0,"name":"CPU","type":"INTEGER"},{"doubleValue":0.0,"integerValue":2004,"longValue":0,"name":"MEMORY","type":"INTEGER"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS","stringSetValue":["22","2376","2375","51678"],"type":"STRINGSET"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS_UDP","stringSetValue":[],"type":"STRINGSET"}],"remainingResources":[{"doubleValue":0.0,"integerValue":1024,"longValue":0,"name":"CPU","type":"INTEGER"},{"doubleValue":0.0,"integerValue":1620,"longValue":0,"name":"MEMORY","type":"INTEGER"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS","stringSetValue":["22","2376","2375","3101","3001","3100","51678","3000"],"type":"STRINGSET"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS_UDP","stringSetValue":[],"type":"STRINGSET"}],"runningTasksCount":1,"status":"ACTIVE","versionInfo":{"agentHash":"4ab1051","agentVersion":"1.4.0","dockerVersion":"DockerVersion: 1.7.1"}}],"failures":[]}`
}
