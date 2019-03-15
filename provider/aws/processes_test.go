package aws_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/pkg/test/awsutil"
	"github.com/stretchr/testify/assert"
)

type streamTester struct {
	io.Reader
	io.Writer
}

func (st streamTester) Close() error {
	return nil
}

func TestProcessExec(t *testing.T) {
	provider := StubAwsProvider(
		cycleProcessListStackResources,
		cycleProcessDescribeStacks,
		cycleProcessListTasksByStack,
		cycleProcessListTasksByService1,
		cycleProcessListTasksByService2,
		cycleProcessListTasksByStarted,
		cycleProcessDescribeTasksAll,
		cycleProcessDescribeTaskDefinition1,
		cycleProcessDescribeContainerInstances,
		cycleProcessDescribeTaskDefinition1,
		cycleProcessDescribeContainerInstances,
		cycleProcessDescribeRackInstances,
		cycleProcessListTasksRunning,
		cycleProcessListTasksStopped,
		cycleProcessDescribeTasks,
		cycleProcessDescribeContainerInstances,
		cycleProcessDescribeInstances,
		cycleProcessListTasksRunning,
		cycleProcessListTasksStopped,
		cycleProcessDescribeTasks,
		cycleProcessDescribeContainerInstances,
		cycleProcessDescribeInstances,
		cycleProcessListTasksRunning,
		cycleProcessListTasksStopped,
		cycleProcessDescribeStacks,
		cycleProcessDescribeStackResources,
	)
	defer provider.Close()

	d := stubDocker(
		cycleProcessDockerListContainers1,
		cycleProcessDockerInspect,
		cycleProcessDockerCreateExec,
		cycleProcessDockerStartExec,
		cycleProcessDockerResizeExec,
		cycleProcessDockerInspectExec,
	)
	defer d.Close()

	in := &bytes.Buffer{}
	out := &bytes.Buffer{}

	code, err := provider.ProcessExec("myapp", "5850760f0845", "ls -la", streamTester{in, out}, structs.ProcessExecOptions{
		Height: options.Int(10),
		Width:  options.Int(20),
	})

	assert.NoError(t, err)
	assert.Equal(t, []byte("foo"), out.Bytes())
	assert.Equal(t, 0, code)
}

func TestProcessList(t *testing.T) {
	provider := StubAwsProvider(
		cycleProcessListStackResources,
		cycleProcessDescribeStacks,
		cycleProcessListTasksByStack,
		cycleProcessListTasksByService1,
		cycleProcessListTasksByService2,
		cycleProcessListTasksByStarted,
		cycleProcessDescribeTasksAll,
		cycleProcessDescribeTaskDefinition1,
		cycleProcessDescribeContainerInstances,
		cycleProcessDescribeTaskDefinition2,
		cycleProcessDescribeContainerInstances,
		cycleProcessDescribeRackInstances,
	)
	defer provider.Close()

	s, err := provider.ProcessList("myapp", structs.ProcessListOptions{})

	ps := structs.Processes{
		structs.Process{
			Id:       "5850760f0845",
			App:      "myapp",
			Name:     "web",
			Release:  "R1234",
			Command:  "",
			Host:     "10.0.1.244",
			Image:    "778743527532.dkr.ecr.us-east-1.amazonaws.com/convox-myapp-nkdecwppkq:web.BMPBJLITPZT",
			Instance: "i-5bc45dc2",
			Ports:    []string{},
			Cpu:      0,
			Memory:   0,
			Status:   "running",
		},
		structs.Process{
			Id:       "5850760f0846",
			App:      "myapp",
			Name:     "web",
			Release:  "R1234",
			Command:  "ls -la 'name space'",
			Host:     "10.0.1.244",
			Image:    "778743527532.dkr.ecr.us-east-1.amazonaws.com/convox-myapp-nkdecwppkq:web.BMPBJLITPZT",
			Instance: "i-5bc45dc2",
			Ports:    []string{},
			Cpu:      0,
			Memory:   0,
			Status:   "pending",
		},
	}

	assert.NoError(t, err)
	assert.EqualValues(t, ps, s)
}

func TestProcessListEmpty(t *testing.T) {
	provider := StubAwsProvider(
		cycleProcessListStackResources,
		cycleProcessDescribeStacks,
		cycleProcessListTasksByStackEmpty,
		cycleProcessListTasksByService1Empty,
		cycleProcessListTasksByService2Empty,
		cycleProcessListTasksByStartedEmpty,
		cycleProcessDescribeRackInstances,
	)
	defer provider.Close()

	s, err := provider.ProcessList("myapp", structs.ProcessListOptions{})

	assert.NoError(t, err)
	assert.EqualValues(t, structs.Processes{}, s)
}

func TestProcessListWithBuildCluster(t *testing.T) {
	provider := StubAwsProvider(
		cycleProcessListStackResources,
		cycleProcessDescribeStacks,
		cycleProcessListTasksByStack,
		cycleProcessListTasksByService1,
		cycleProcessListTasksByService2,
		cycleProcessListTasksByStarted,
		cycleProcessListTasksBuildCluster,
		cycleProcessDescribeTasksAllWithBuildCluster,
		cycleProcessDescribeTasksAllOnBuildCluster,
		cycleProcessDescribeTaskDefinition1,
		cycleProcessDescribeContainerInstances,
		cycleProcessDescribeTaskDefinition2,
		cycleProcessDescribeContainerInstances,
		cycleProcessDescribeTaskDefinition2,
		cycleProcessDescribeContainerInstances,
		cycleProcessDescribeRackInstances,
	)
	defer provider.Close()

	provider.BuildCluster = "cluster-build"

	s, err := provider.ProcessList("myapp", structs.ProcessListOptions{})

	ps := structs.Processes{
		structs.Process{
			Id:       "5850760f0845",
			App:      "myapp",
			Name:     "web",
			Release:  "R1234",
			Command:  "",
			Host:     "10.0.1.244",
			Image:    "778743527532.dkr.ecr.us-east-1.amazonaws.com/convox-myapp-nkdecwppkq:web.BMPBJLITPZT",
			Instance: "i-5bc45dc2",
			Ports:    []string{},
			Cpu:      0,
			Memory:   0,
			Status:   "running",
		},
		structs.Process{
			Id:       "5850760f0846",
			App:      "myapp",
			Name:     "web",
			Release:  "R1234",
			Command:  "ls -la 'name space'",
			Host:     "10.0.1.244",
			Image:    "778743527532.dkr.ecr.us-east-1.amazonaws.com/convox-myapp-nkdecwppkq:web.BMPBJLITPZT",
			Instance: "i-5bc45dc2",
			Ports:    []string{},
			Cpu:      0,
			Memory:   0,
			Status:   "running",
		},
		structs.Process{
			Id:       "5850760f0848",
			App:      "myapp",
			Name:     "web",
			Release:  "R1234",
			Command:  "",
			Host:     "10.0.1.244",
			Image:    "778743527532.dkr.ecr.us-east-1.amazonaws.com/convox-myapp-nkdecwppkq:web.BMPBJLITPZT",
			Instance: "i-5bc45dc2",
			Ports:    []string{},
			Cpu:      0,
			Memory:   0,
			Status:   "running",
		},
	}

	assert.NoError(t, err)
	assert.EqualValues(t, ps, s)
}

func TestProcessRunDetached(t *testing.T) {
	provider := StubAwsProvider(
		cycleProcessReleaseGetItem,
		cycleProcessDescribeStacks,
		cycleProcessDescribeStacks,
		cycleProcessReleaseGetItem,
		cycleProcessReleaseListStackResources,
		cycleProcessReleaseEnvironmentGet,
		cycleSystemListStackResources,
		cycleProcessBuildGetItem,
		cycleProcessListStackResources,
		cycleProcessDescribeServices,
		cycleProcessDescribeTaskDefinition1,
		cycleProcessListStackResources,
		cycleProcessListStackResources,
		cycleProcessRegisterTaskDefinitionDetached,
		cycleProcessReleaseUpdateItem,
		cycleProcessRunTaskDetached,
		cycleProcessListStackResources,
		cycleProcessDescribeStacks,
		cycleProcessListTasksByStack,
		cycleProcessListTasksByService1,
		cycleProcessListTasksByService2,
		cycleProcessListTasksByStarted,
		cycleProcessDescribeTasks,
		cycleProcessDescribeTaskDefinition1,
		cycleProcessDescribeContainerInstances,
		cycleProcessDescribeRackInstances,
	)
	defer provider.Close()

	psa, err := provider.ProcessRun("myapp", "web", structs.ProcessRunOptions{
		Command: options.String("ls test"),
		Release: options.String("RVFETUHHKKD"),
	})

	pse := &structs.Process{Id: "5850760f0845", App: "", Command: "ls -la 'name space'", Cpu: 0, Host: "10.0.1.244", Image: "778743527532.dkr.ecr.us-east-1.amazonaws.com/convox-myapp-nkdecwppkq:web.BMPBJLITPZT", Instance: "i-5bc45dc2", Memory: 0, Name: "web", Ports: []string{}, Release: "R1234", Status: "running"}

	assert.NoError(t, err)
	assert.Equal(t, pse, psa)
}

func TestProcessStop(t *testing.T) {
	provider := StubAwsProvider(
		cycleProcessListTasksRunning,
		cycleProcessListTasksStopped,
		cycleProcessStopTask,
	)
	defer provider.Close()

	err := provider.ProcessStop("myapp", "5850760f0845")

	assert.NoError(t, err)
}

var cycleProcessDescribeContainerInstances = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.DescribeContainerInstances",
		Body: `{
			"cluster": "cluster-test",
			"containerInstances": [
				"arn:aws:ecs:us-east-1:778743527532:container-instance/e126c67d-fa95-4b09-8b4a-3723932cd2aa"
			]
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"failures": [],
			"containerInstances": [
				{
					"ec2InstanceId": "i-5bc45dc2"
				}
			]
		}`,
	},
}

var cycleProcessDescribeInstances = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "",
		Body:       `Action=DescribeInstances&InstanceId.1=i-5bc45dc2&Version=2016-11-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<?xml version="1.0" encoding="UTF-8"?>
			<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
				<reservationSet>
					<item>
						<reservationId>r-003ed1d7</reservationId>
						<ownerId>778743527532</ownerId>
						<groupSet/>
						<instancesSet>
							<item>
								<instanceId>i-5bc45dc2</instanceId>
								<privateIpAddress>10.0.1.244</privateIpAddress>
							</item>
						</instancesSet>
					</item>
				</reservationSet>
			</DescribeInstancesResponse>
		}`,
	},
}

var cycleProcessDescribeRackInstances = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "",
		Body:       `Action=DescribeInstances&Filter.1.Name=tag%3ARack&Filter.1.Value.1=convox&Version=2016-11-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<?xml version="1.0" encoding="UTF-8"?>
			<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
				<reservationSet>
					<item>
						<reservationId>r-003ed1d7</reservationId>
						<ownerId>778743527532</ownerId>
						<groupSet/>
						<instancesSet>
							<item>
								<instanceId>i-5bc45dc2</instanceId>
								<privateIpAddress>10.0.1.244</privateIpAddress>
							</item>
						</instancesSet>
					</item>
				</reservationSet>
			</DescribeInstancesResponse>
		}`,
	},
}

var cycleProcessDescribeServices = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.DescribeServices",
		Body: `{
			"cluster": "cluster-test",
			"services": [
				"arn:aws:ecs:us-east-1:778743527532:service/convox-myapp-ServiceWeb-1I2PTXAZ5ECRD"
			]
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"services": [
				{
					"status": "ACTIVE",
					"taskDefinition": "arn:aws:ecs:us-east-1:778743527532:task-definition/convox-myapp-web:34",
					"pendingCount": 0,
					"loadBalancers": [
						{
							"containerName": "web",
							"containerPort": 4000,
							"loadBalancerName": "rails-web-HY3CGZN"
						}
					],
					"roleArn": "arn:aws:iam::778743527532:role/convox/convox-myapp-ServiceRole-1U94NKEJV4H6U",
					"createdAt": 1472493833.436,
					"desiredCount": 1,
					"serviceName": "convox-myapp-ServiceWeb-1OKBY3I5WYIIP",
					"clusterArn": "arn:aws:ecs:us-east-1:778743527532:cluster/david-Cluster-11CH3SUXA7BQH",
					"serviceArn": "arn:aws:ecs:us-east-1:778743527532:service/convox-myapp-ServiceWeb-1OKBY3I5WYIIP",
					"deploymentConfiguration": {
						"maximumPercent": 200,
						"minimumHealthyPercent": 100
					},
					"deployments": [
						{
							"status": "PRIMARY",
							"pendingCount": 0,
							"createdAt": 1473481958.792,
							"desiredCount": 1,
							"taskDefinition": "arn:aws:ecs:us-east-1:778743527532:task-definition/convox-myapp-web:34",
							"updatedAt": 1473481958.792,
							"id": "ecs-svc/9223370563372817015",
							"runningCount": 1
						}
					],
					"events": [],
					"runningCount": 1
				}
			],
			"failures": []
		}`,
	},
}

var cycleProcessDescribeStacks = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=DescribeStacks&StackName=convox-myapp&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
				<DescribeStacksResult>
					<Stacks>
						<member>
							<Outputs>
								<member>
									<OutputKey>RegistryId</OutputKey>
									<OutputValue>778743527532</OutputValue>
								</member>
								<member>
									<OutputKey>RegistryRepository</OutputKey>
									<OutputValue>convox-myapp-nkdecwppkq</OutputValue>
								</member>
								<member>
									<OutputKey>ServiceWebService</OutputKey>
									<OutputValue>service-web</OutputValue>
								</member>
							</Outputs>
							<Capabilities>
								<member>CAPABILITY_IAM</member>
							</Capabilities>
							<CreationTime>2016-08-29T17:45:22.396Z</CreationTime>
							<NotificationARNs/>
							<StackId>arn:aws:cloudformation:us-east-1:778743527532:stack/convox-myapp/5c05e0c0-6e10-11e6-8a4e-50fae98a10d2</StackId>
							<StackName>convox-myapp</StackName>
							<StackStatus>UPDATE_COMPLETE</StackStatus>
							<DisableRollback>false</DisableRollback>
							<Tags>
								<member>
									<Value>convox</Value>
									<Key>Rack</Key>
								</member>
								<member>
									<Value>app</Value>
									<Key>Type</Key>
								</member>
								<member>
									<Value>convox</Value>
									<Key>System</Key>
								</member>
								<member>
									<Value>myapp</Value>
									<Key>Name</Key>
								</member>
							</Tags>
							<LastUpdatedTime>2016-09-10T04:32:19.081Z</LastUpdatedTime>
							<Parameters>
							</Parameters>
						</member>
					</Stacks>
				</DescribeStacksResult>
				<ResponseMetadata>
					<RequestId>9627285a-7903-11e6-a36d-77452275e1ca</RequestId>
				</ResponseMetadata>
			</DescribeStacksResponse>
		`,
	},
}

var cycleProcessListStackResources = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "",
		Body:       `Action=ListStackResources&StackName=convox-myapp&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<ListStackResourcesResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
				<ListStackResourcesResult>
					<StackResourceSummaries>
						<member>
							<PhysicalResourceId>loggroup-test</PhysicalResourceId>
							<ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
							<StackId>arn:aws:cloudformation:us-east-1:778743527532:stack/convox-myapp/5c05e0c0-6e10-11e6-8a4e-50fae98a10d2</StackId>
							<StackName>convox-myapp</StackName>
							<LogicalResourceId>LogGroup</LogicalResourceId>
							<Timestamp>2016-09-10T04:35:11.280Z</Timestamp>
							<ResourceType>AWS::Logs::LogGroup</ResourceType>
						</member>
						<member>
							<PhysicalResourceId>settings</PhysicalResourceId>
							<ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
							<StackId>arn:aws:cloudformation:us-east-1:778743527532:stack/convox-myapp/5c05e0c0-6e10-11e6-8a4e-50fae98a10d2</StackId>
							<StackName>convox-myapp</StackName>
							<LogicalResourceId>Settings</LogicalResourceId>
							<Timestamp>2016-09-10T04:35:11.280Z</Timestamp>
							<ResourceType>AWS::S3::Bucket</ResourceType>
						</member>
						<member>
							<PhysicalResourceId>arn:aws:ecs:us-east-1:778743527532:service/convox-myapp-ServiceDatabase-1I2PTXAZ5ECRD</PhysicalResourceId>
							<ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
							<StackId>arn:aws:cloudformation:us-east-1:778743527532:stack/convox-myapp/5c05e0c0-6e10-11e6-8a4e-50fae98a10d2</StackId>
							<StackName>convox-myapp</StackName>
							<LogicalResourceId>ServiceDatabase</LogicalResourceId>
							<Timestamp>2016-09-10T04:35:11.280Z</Timestamp>
							<ResourceType>AWS::ECS::Service</ResourceType>
						</member>
						<member>
							<PhysicalResourceId>arn:aws:ecs:us-east-1:778743527532:service/convox-myapp-ServiceWeb-1I2PTXAZ5ECRD</PhysicalResourceId>
							<ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
							<StackId>arn:aws:cloudformation:us-east-1:778743527532:stack/convox-myapp/5c05e0c0-6e10-11e6-8a4e-50fae98a10d2</StackId>
							<StackName>convox-myapp</StackName>
							<LogicalResourceId>ServiceWeb</LogicalResourceId>
							<Timestamp>2016-09-10T04:35:11.280Z</Timestamp>
							<ResourceType>AWS::ECS::Service</ResourceType>
						</member>
					</StackResourceSummaries>
				</ListStackResourcesResult>
				<ResponseMetadata>
					<RequestId>8be86de9-7760-11e6-b2f2-6b253bb2c005</RequestId>
				</ResponseMetadata>
			</ListStackResourcesResponse>
		`,
	},
}

var cycleProcessDescribeStackResources = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "",
		Body:       `Action=DescribeStackResources&StackName=convox-myapp&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<DescribeStackResourcesResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
				<DescribeStackResourcesResult>
					<StackResources>
						<member>
							<PhysicalResourceId>settings</PhysicalResourceId>
							<ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
							<StackId>arn:aws:cloudformation:us-east-1:778743527532:stack/convox-myapp/5c05e0c0-6e10-11e6-8a4e-50fae98a10d2</StackId>
							<StackName>convox-myapp</StackName>
							<LogicalResourceId>Settings</LogicalResourceId>
							<Timestamp>2016-09-10T04:35:11.280Z</Timestamp>
							<ResourceType>AWS::S3::Bucket</ResourceType>
						</member>
						<member>
							<PhysicalResourceId>arn:aws:ecs:us-east-1:778743527532:service/convox-myapp-ServiceDatabase-1I2PTXAZ5ECRD</PhysicalResourceId>
							<ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
							<StackId>arn:aws:cloudformation:us-east-1:778743527532:stack/convox-myapp/5c05e0c0-6e10-11e6-8a4e-50fae98a10d2</StackId>
							<StackName>convox-myapp</StackName>
							<LogicalResourceId>ServiceDatabase</LogicalResourceId>
							<Timestamp>2016-09-10T04:35:11.280Z</Timestamp>
							<ResourceType>AWS::ECS::Service</ResourceType>
						</member>
						<member>
							<PhysicalResourceId>arn:aws:ecs:us-east-1:778743527532:service/convox-myapp-ServiceWeb-1I2PTXAZ5ECRD</PhysicalResourceId>
							<ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
							<StackId>arn:aws:cloudformation:us-east-1:778743527532:stack/convox-myapp/5c05e0c0-6e10-11e6-8a4e-50fae98a10d2</StackId>
							<StackName>convox-myapp</StackName>
							<LogicalResourceId>ServiceWeb</LogicalResourceId>
							<Timestamp>2016-09-10T04:35:11.280Z</Timestamp>
							<ResourceType>AWS::ECS::Service</ResourceType>
						</member>
					</StackResources>
				</DescribeStackResourcesResult>
				<ResponseMetadata>
					<RequestId>8be86de9-7760-11e6-b2f2-6b253bb2c005</RequestId>
				</ResponseMetadata>
			</DescribeStackResourcesResponse>
		`,
	},
}

var cycleProcessDescribeTasks = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTasks",
		Body: `{
			"cluster": "cluster-test",
			"tasks": [
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845"
			]
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"failures": [],
			"tasks": [
				{
					"taskArn": "arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845",
					"overrides": {
						"containerOverrides": [
							{
								"command": ["sh", "-c", "foo"]
							}
						]
					},
					"lastStatus": "RUNNING",
					"taskDefinitionArn": "arn:aws:ecs:us-east-1:778743527532:task-definition/convox-myapp-web:34",
					"containerInstanceArn": "arn:aws:ecs:us-east-1:778743527532:container-instance/e126c67d-fa95-4b09-8b4a-3723932cd2aa",
					"containers": [
						{
							"name": "web",
							"containerArn": "arn:aws:ecs:us-east-1:778743527532:container/3ab3b8c5-aa5c-4b54-89f8-5f1193aff5f9"
						}
					]
				}
			]
		}`,
	},
}

var cycleProcessDescribeTasksStopped = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTasks",
		Body: `{
			"cluster": "cluster-test",
			"tasks": [
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845"
			]
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"failures": [],
			"tasks": [
				{
					"taskArn": "arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845",
					"overrides": {
						"containerOverrides": [
							{
								"command": ["sh", "-c", "foo"]
							}
						]
					},
					"lastStatus": "STOPPED",
					"stoppedReason": "exit:3",
					"taskDefinitionArn": "arn:aws:ecs:us-east-1:778743527532:task-definition/convox-myapp-web:34",
					"containerInstanceArn": "arn:aws:ecs:us-east-1:778743527532:container-instance/e126c67d-fa95-4b09-8b4a-3723932cd2aa",
					"containers": [
						{
							"name": "web",
							"containerArn": "arn:aws:ecs:us-east-1:778743527532:container/3ab3b8c5-aa5c-4b54-89f8-5f1193aff5f9"
						}
					]
				}
			]
		}`,
	},
}

var cycleProcessDescribeTasksAll = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTasks",
		Body: `{
			"cluster": "cluster-test",
			"tasks": [
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0846",
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0846",
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0847",
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845"
			]
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"failures": [],
			"tasks": [
				{
					"taskArn": "arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0846",
					"overrides": {
						"containerOverrides": []
					},
					"taskDefinitionArn": "arn:aws:ecs:us-east-1:778743527532:task-definition/convox-myapp-web:34",
					"lastStatus": "PENDING",
					"containerInstanceArn": "arn:aws:ecs:us-east-1:778743527532:container-instance/e126c67d-fa95-4b09-8b4a-3723932cd2aa",
					"containers": [
						{
							"name": "web",
							"containerArn": "arn:aws:ecs:us-east-1:778743527532:container/3ab3b8c5-aa5c-4b54-89f8-5f1193aff5f9"
						}
					]
				},
				{
					"taskArn": "arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845",
					"overrides": {
						"containerOverrides": [
							{
								"command": ["sh", "-c", "foo"]
							}
						]
					},
					"taskDefinitionArn": "arn:aws:ecs:us-east-1:778743527532:task-definition/convox-myapp-web:34",
					"lastStatus": "RUNNING",
					"containerInstanceArn": "arn:aws:ecs:us-east-1:778743527532:container-instance/e126c67d-fa95-4b09-8b4a-3723932cd2aa",
					"containers": [
						{
							"name": "web",
							"containerArn": "arn:aws:ecs:us-east-1:778743527532:container/3ab3b8c5-aa5c-4b54-89f8-5f1193aff5f9"
						}
					]
				}
			]
		}`,
	},
}

var cycleProcessDescribeTasksAllWithBuildCluster = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTasks",
		Body: `{
			"cluster": "cluster-test",
			"tasks": [
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0846",
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0846",
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0847",
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845",
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0848"
			]
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"failures": [],
			"tasks": [
				{
					"taskArn": "arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0846",
					"overrides": {
						"containerOverrides": []
					},
					"taskDefinitionArn": "arn:aws:ecs:us-east-1:778743527532:task-definition/convox-myapp-web:34",
					"lastStatus": "RUNNING",
					"containerInstanceArn": "arn:aws:ecs:us-east-1:778743527532:container-instance/e126c67d-fa95-4b09-8b4a-3723932cd2aa",
					"containers": [
						{
							"name": "web",
							"containerArn": "arn:aws:ecs:us-east-1:778743527532:container/3ab3b8c5-aa5c-4b54-89f8-5f1193aff5f9"
						}
					]
				},
				{
					"taskArn": "arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845",
					"overrides": {
						"containerOverrides": [
							{
								"command": ["sh", "-c", "foo"]
							}
						]
					},
					"taskDefinitionArn": "arn:aws:ecs:us-east-1:778743527532:task-definition/convox-myapp-web:34",
					"lastStatus": "RUNNING",
					"containerInstanceArn": "arn:aws:ecs:us-east-1:778743527532:container-instance/e126c67d-fa95-4b09-8b4a-3723932cd2aa",
					"containers": [
						{
							"name": "web",
							"containerArn": "arn:aws:ecs:us-east-1:778743527532:container/3ab3b8c5-aa5c-4b54-89f8-5f1193aff5f9"
						}
					]
				}
			]
		}`,
	},
}

var cycleProcessDescribeTasksAllOnBuildCluster = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTasks",
		Body: `{
			"cluster": "cluster-build",
			"tasks": [
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0846",
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0846",
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0847",
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845",
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0848"
			]
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"failures": [],
			"tasks": [
				{
					"taskArn": "arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0848",
					"overrides": {
						"containerOverrides": [
							{
								"command": ["sh", "-c", "foo"]
							}
						]
					},
					"taskDefinitionArn": "arn:aws:ecs:us-east-1:778743527532:task-definition/convox-myapp-web:34",
					"lastStatus": "RUNNING",
					"containerInstanceArn": "arn:aws:ecs:us-east-1:778743527532:container-instance/e126c67d-fa95-4b09-8b4a-3723932cd2aa",
					"containers": [
						{
							"name": "web",
							"containerArn": "arn:aws:ecs:us-east-1:778743527532:container/3ab3b8c5-aa5c-4b54-89f8-5f1193aff5f9"
						}
					]
				}
			]
		}`,
	},
}

var cycleProcessDescribeTaskDefinition1 = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition",
		Body: `{
			"taskDefinition": "arn:aws:ecs:us-east-1:778743527532:task-definition/convox-myapp-web:34"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskDefinition": {
				"status": "ACTIVE",
				"family": "convox-myapp-web",
				"requiresAttributes": [
					{
						"name": "com.amazonaws.ecs.capability.ecr-auth"
					}
				],
				"volumes": [],
				"taskDefinitionArn": "arn:aws:ecs:us-east-1:778743527532:task-definition/convox-myapp-web:34",
				"containerDefinitions": [
					{
						"command": [ "ls", "-la", "name space" ],
						"environment": [
							{
								"name": "RELEASE",
								"value": "R1234"
							}
						],
						"name": "web",
						"mountPoints": [],
						"image": "778743527532.dkr.ecr.us-east-1.amazonaws.com/convox-myapp-nkdecwppkq:web.BMPBJLITPZT",
						"cpu": 0,
						"portMappings": [],
						"memory": 256,
						"privileged": false,
						"essential": true,
						"volumesFrom": []
					}
				],
				"revision": 34
			}
		}`,
	},
}

var cycleProcessDescribeTaskDefinition2 = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition",
		Body: `{
			"taskDefinition": "arn:aws:ecs:us-east-1:778743527532:task-definition/convox-myapp-web:34"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskDefinition": {
				"status": "ACTIVE",
				"family": "convox-myapp-web",
				"requiresAttributes": [
					{
						"name": "com.amazonaws.ecs.capability.ecr-auth"
					}
				],
				"volumes": [],
				"taskDefinitionArn": "arn:aws:ecs:us-east-1:778743527532:task-definition/convox-myapp-web:34",
				"containerDefinitions": [
					{
						"environment": [
							{
								"name": "RELEASE",
								"value": "R1234"
							}
						],
						"name": "web",
						"mountPoints": [],
						"image": "778743527532.dkr.ecr.us-east-1.amazonaws.com/convox-myapp-nkdecwppkq:web.BMPBJLITPZT",
						"cpu": 0,
						"portMappings": [],
						"memory": 256,
						"privileged": false,
						"essential": true,
						"volumesFrom": []
					}
				],
				"revision": 34
			}
		}`,
	},
}

var cycleProcessListTasksAll = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.ListTasks",
		Body: `{
			"cluster": "cluster-test"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskArns": [
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845"
			]
		}`,
	},
}

var cycleProcessListTasksRunning = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.ListTasks",
		Body: `{
			"cluster": "cluster-test",
			"desiredStatus": "RUNNING"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskArns": [
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845"
			]
		}`,
	},
}

var cycleProcessListTasksStopped = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.ListTasks",
		Body: `{
			"cluster": "cluster-test",
			"desiredStatus": "STOPPED"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskArns": [
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845"
			]
		}`,
	},
}

var cycleProcessListTasksByService1 = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.ListTasks",
		Body: `{
			"cluster": "cluster-test",
			"serviceName": "arn:aws:ecs:us-east-1:778743527532:service/convox-myapp-ServiceDatabase-1I2PTXAZ5ECRD"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskArns": [
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0846"
			]
		}`,
	},
}

var cycleProcessListTasksByService1Empty = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.ListTasks",
		Body: `{
			"cluster": "cluster-test",
			"serviceName": "arn:aws:ecs:us-east-1:778743527532:service/convox-myapp-ServiceDatabase-1I2PTXAZ5ECRD"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskArns": []
		}`,
	},
}

var cycleProcessListTasksByService2 = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.ListTasks",
		Body: `{
			"cluster": "cluster-test",
			"serviceName": "arn:aws:ecs:us-east-1:778743527532:service/convox-myapp-ServiceWeb-1I2PTXAZ5ECRD"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskArns": [
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0847"
			]
		}`,
	},
}

var cycleProcessListTasksByService2Empty = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.ListTasks",
		Body: `{
			"cluster": "cluster-test",
			"serviceName": "arn:aws:ecs:us-east-1:778743527532:service/convox-myapp-ServiceWeb-1I2PTXAZ5ECRD"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskArns": []
		}`,
	},
}

var cycleProcessListTasksByStarted = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.ListTasks",
		Body: `{
			"cluster": "cluster-test",
			"startedBy": "convox.myapp"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskArns": [
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845"
			]
		}`,
	},
}

var cycleProcessListTasksByStartedEmpty = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.ListTasks",
		Body: `{
			"cluster": "cluster-test",
			"startedBy": "convox.myapp"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskArns": []
		}`,
	},
}

var cycleProcessListTasksByStack = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.ListTasks",
		Body: `{
			"cluster": "cluster-test",
			"serviceName": "service-web"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskArns": [
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0846"
			]
		}`,
	},
}

var cycleProcessListTasksByStackEmpty = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.ListTasks",
		Body: `{
			"cluster": "cluster-test",
			"serviceName": "service-web"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskArns": []
		}`,
	},
}

var cycleProcessListTasksBuildCluster = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.ListTasks",
		Body: `{
			"cluster": "cluster-build",
			"startedBy": "convox.myapp"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskArns": [
				"arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0848"
			]
		}`,
	},
}

var cycleProcessRegisterTaskDefinitionAttached = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.RegisterTaskDefinition",
		Body: `{
			"containerDefinitions": [
				{
					"dockerLabels": {
						"convox.process.type": "oneoff"
					},
					"environment": [
						{
							"name": "APP",
							"value": "myapp"
						},
						{
							"name": "AWS_REGION",
							"value": "us-test-1"
						},
						{
							"name": "BUILD",
							"value": "BHINCLZYYVN"
						},
						{
							"name": "BUILD_DESCRIPTION",
							"value": "test desc"
						},
						{
							"name": "LOG_GROUP",
							"value": ""
						},
						{
							"name": "PROCESS",
							"value": "web"
						},
						{
							"name": "RACK",
							"value": "convox"
						},
						{
							"name": "RELEASE",
							"value": "RVFETUHHKKD"
						},
						{
							"name": "foo",
							"value": "bar"
						}
					],
					"essential": true,
					"image": "778743527532.dkr.ecr.us-test-1.amazonaws.com/convox-myapp-nkdecwppkq:web.BHINCLZYYVN",
					"memoryReservation": 512,
					"name": "web",
					"privileged": false
				}
			],
			"family": "convox-myapp-web",
			"taskRoleArn": ""
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskDefinition": {
				"taskDefinitionArn": "arn:aws:ecs:us-east-1:012345678910:task-definition/hello_world:4"
			}
		}`,
	},
}

var cycleProcessRegisterTaskDefinitionDetached = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.RegisterTaskDefinition",
		Body: `{
			"containerDefinitions": [
				{
					"dockerLabels": {
						"convox.process.type": "oneoff",
						"convox.release": "RVFETUHHKKD"
					},
					"environment": [
						{
							"name": "APP",
							"value": "myapp"
						},
						{
							"name": "AWS_REGION",
							"value": "us-test-1"
						},
						{
							"name": "BUILD",
							"value": "BHINCLZYYVN"
						},
						{
							"name": "BUILD_DESCRIPTION",
							"value": "test desc"
						},
						{
							"name": "LOG_GROUP",
							"value": ""
						},
						{
							"name": "PROCESS",
							"value": "web"
						},
						{
							"name": "RACK",
							"value": "convox"
						},
						{
							"name": "RELEASE",
							"value": "RVFETUHHKKD"
						},
						{
							"name": "foo",
							"value": "bar"
						}
					],
					"essential": true,
					"image": "778743527532.dkr.ecr.us-test-1.amazonaws.com/convox-myapp-nkdecwppkq:web.BHINCLZYYVN",
					"logConfiguration": {
						"logDriver": "awslogs",
						"options": {
							"awslogs-group": "loggroup-test",
							"awslogs-region": "us-test-1",
							"awslogs-stream-prefix": "service"
						}
					},
					"memoryReservation": 512,
					"name": "web",
					"privileged": false
				}
			],
			"family": "convox-myapp-web",
			"taskRoleArn": ""
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"taskDefinition": {
				"taskDefinitionArn": "arn:aws:ecs:us-east-1:012345678910:task-definition/hello_world:4"
			}
		}`,
	},
}

var cycleProcessRunTaskAttached = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.RunTask",
		Body: `{
			"cluster": "cluster-test",
			"count": 1,
			"overrides": {
				"containerOverrides": [
					{
						"command": [
							"sleep",
							"3600"
						],
						"environment": [
						  {
								"name": "COMMAND",
								"value": "ls -la"
							}
						],
						"name": "web"
					}
				]
			},
			"startedBy": "convox.myapp",
			"taskDefinition": "arn:aws:ecs:us-east-1:012345678910:task-definition/hello_world:4"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"tasks": [
				{
					"containers": [
						{
							"containerArn": "arn:aws:ecs:us-east-1:012345678910:container/e1ed7aac-d9b2-4315-8726-d2432bf11868",
							"lastStatus": "PENDING",
							"name": "wordpress",
							"taskArn": "arn:aws:ecs:us-east-1:012345678910:task/d8c67b3c-ac87-4ffe-a847-4785bc3a8b55"
						}
					],
					"taskArn": "arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845"
				}
			]
		}`,
	},
}

var cycleProcessRunTaskDetached = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.RunTask",
		Body: `{
			"cluster": "cluster-test",
			"count": 1,
			"overrides": {
				"containerOverrides": [
					{
						"command": [
							"sh",
							"-c",
							"ls test"
						],
						"name": "web"
					}
				]
			},
			"startedBy": "convox.myapp",
			"taskDefinition": "arn:aws:ecs:us-east-1:012345678910:task-definition/hello_world:4"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"tasks": [
				{
					"containers": [
						{
							"containerArn": "arn:aws:ecs:us-east-1:012345678910:container/e1ed7aac-d9b2-4315-8726-d2432bf11868",
							"lastStatus": "PENDING",
							"name": "wordpress",
							"taskArn": "arn:aws:ecs:us-east-1:012345678910:task/d8c67b3c-ac87-4ffe-a847-4785bc3a8b55"
						}
					],
					"taskArn": "arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845"
				}
			]
		}`,
	},
}

var cycleProcessStopTask = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.StopTask",
		Body: `{
			"cluster": "cluster-test",
			"task": "arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"task": {
				"taskArn": "arn:aws:ecs:us-east-1:778743527532:task/014b7e61-cc23-47e8-9dc6-0f51f03ff369"
			}
		}`,
	},
}

var cycleProcessStopTaskReason = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.StopTask",
		Body: `{
			"cluster": "cluster-test",
			"reason": "exit:0",
			"task": "arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"task": {
				"taskArn": "arn:aws:ecs:us-east-1:778743527532:task/014b7e61-cc23-47e8-9dc6-0f51f03ff369"
			}
		}`,
	},
}

var cycleProcessBuildGetItem = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.GetItem",
		Body:       `{"ConsistentRead":true,"Key":{"id":{"S":"BHINCLZYYVN"}},"TableName":"convox-builds"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{"Item":{"id":{"S":"BHINCLZYYVN"},"description":{"S":"test desc"}}}`,
	},
}

var cycleProcessReleaseGetItem = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.GetItem",
		Body:       `{"ConsistentRead":true,"Key":{"id":{"S":"RVFETUHHKKD"}},"TableName":"convox-releases"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{"Item":{"id":{"S":"RVFETUHHKKD"},"build":{"S":"BHINCLZYYVN"},"app":{"S":"myapp"},"manifest":{"S":"web:\n  image: myapp\n  ports:\n  - 80:80\n"},"created":{"S":"20160404.143542.627770380"}}}`,
	},
}

var cycleProcessReleaseListStackResources = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "",
		Body:       `Action=ListStackResources&StackName=convox-myapp&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<ListStackResourcesResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
  <ListStackResourcesResult>
    <StackResourceSummaries>
    <member>
      <PhysicalResourceId></PhysicalResourceId>
      <ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
      <LogicalResourceId>Settings</LogicalResourceId>
      <Timestamp>2016-10-22T02:53:23.817Z</Timestamp>
      <ResourceType>AWS::Logs::LogGroup</ResourceType>
    </member>
    </StackResourceSummaries>
  </ListStackResourcesResult>
  <ResponseMetadata>
    <RequestId>50ce1445-9805-11e6-8ba2-2b306877d289</RequestId>
  </ResponseMetadata>
</ListStackResourcesResponse>
		`,
	},
}

var cycleProcessReleaseDescribeStackResources = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "",
		Body:       `Action=DescribeStackResources&StackName=convox-myapp&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<DescribeStackResourcesResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
  <DescribeStackResourcesResult>
    <StackResources>
    <member>
      <PhysicalResourceId></PhysicalResourceId>
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

var cycleProcessReleaseEnvironmentGet = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "GET",
		RequestURI: "/releases/RVFETUHHKKD/env",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       "foo=bar",
	},
}

var cycleProcessReleaseUpdateItem = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.UpdateItem",
		Body: `{
			"ExpressionAttributeValues": {
				":definitions": {
					"S": "{\"web.run\":\"arn:aws:ecs:us-east-1:012345678910:task-definition/hello_world:4\"}"
				}
			},
			"Key": {
				"id": {
					"S": "RVFETUHHKKD"
				}
			},
			"TableName": "convox-releases",
			"UpdateExpression": "set definitions = :definitions"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{}`,
	},
}

var cycleProcessDockerListContainers1 = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "GET",
		RequestURI: "/containers/json?all=1&filters=%7B%22label%22%3A%5B%22com.amazonaws.ecs.task-arn%3Darn%3Aaws%3Aecs%3Aus-east-1%3A778743527532%3Atask%2F50b8de99-f94f-4ecd-a98f-5850760f0845%22%2C%22convox.release%22%5D%7D",
		Body:       ``,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `[
			{
				"Id": "8dfafdbc3a40",
				"Names":["/boring_feynman"],
				"Image": "ubuntu:latest",
				"ImageId": "d74508fb6632491cea586a1fd7d748dfc5274cd6fdfedee309ecdcbc2bf5cb82",
				"Command": "echo 1",
				"Created": 1367854155,
				"State": "Exited",
				"Status": "Exit 0",
				"Ports": [{"PrivatePort": 2222, "PublicPort": 3333, "Type": "tcp"}]
			}
		]`,
	},
}

var cycleProcessDockerListContainers2 = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "GET",
		RequestURI: "/containers/json?all=1&filters=%7B%22label%22%3A%5B%22com.amazonaws.ecs.task-arn%3Darn%3Aaws%3Aecs%3Aus-east-1%3A778743527532%3Atask%2F50b8de99-f94f-4ecd-a98f-5850760f0846%22%5D%7D",
		Body:       ``,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `[
			{
				"Id": "8dfafdbc3a40",
				"Names":["/boring_feynman"],
				"Image": "ubuntu:latest",
				"ImageId": "d74508fb6632491cea586a1fd7d748dfc5274cd6fdfedee309ecdcbc2bf5cb82",
				"Command": "echo 1",
				"Created": 1367854155,
				"State": "Exited",
				"Status": "Exit 0",
				"Ports": [{"PrivatePort": 2222, "PublicPort": 3333, "Type": "tcp"}]
			}
		]`,
	},
}

var cycleProcessDockerListContainers3 = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "GET",
		RequestURI: "/containers/json?all=1&filters=%7B%22label%22%3A%5B%22com.amazonaws.ecs.task-arn%3Darn%3Aaws%3Aecs%3Aus-east-1%3A778743527532%3Atask%2F50b8de99-f94f-4ecd-a98f-5850760f0848%22%5D%7D",
		Body:       ``,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `[
			{
				"Id": "8dfafdbc3a40",
				"Names":["/boring_feynman"],
				"Image": "ubuntu:latest",
				"ImageId": "d74508fb6632491cea586a1fd7d748dfc5274cd6fdfedee309ecdcbc2bf5cb82",
				"Command": "echo 1",
				"Created": 1367854155,
				"State": "Exited",
				"Status": "Exit 0",
				"Ports": [{"PrivatePort": 2222, "PublicPort": 3333, "Type": "tcp"}]
			}
		]`,
	},
}

var cycleProcessDockerInspect = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "GET",
		RequestURI: "/containers/8dfafdbc3a40/json",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"Id": "8dfafdbc3a40",
			"Config": {
				"Cmd": [
						"ls",
						"-la"
				]
			}
		}`,
	},
}

var cycleProcessDockerStats = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "GET",
		RequestURI: "/containers/8dfafdbc3a40/stats?stream=false",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"memory_stats": {
				"usage" : 6537216,
				"limit" : 67108864
			},
			"precpu_stats" : {
				"cpu_usage" : {
					"total_usage" : 100093996
				},
				"system_cpu_usage" : 9492140000000
			}
		}`,
	},
}

var cycleProcessDockerCreateExec = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/containers/8dfafdbc3a40/exec",
		Body: `{
			"AttachStderr": true,
			"AttachStdin": true,
			"AttachStdout": true,
			"Cmd": [
				"sh",
				"-c",
				"ls -la"
			],
			"Container": "8dfafdbc3a40",
			"Tty": true
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"Id": "123456",
			"Warnings": []
		}`,
	},
}

var cycleProcessDockerStartExec = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/exec/123456/start",
		Body: `{
			"ErrorStream": {
				"Reader": {},
				"Writer": {}
			},
			"InputStream": {
				"Reader": {
					"Reader": {},
					"Writer": {}
				}
			},
			"OutputStream": {
				"Reader": {},
				"Writer": {}
			},
			"RawTerminal": true,
			"Tty": true
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       "foo",
	},
}

var cycleProcessDockerResizeExec = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/exec/123456/resize?h=10&w=20",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       "",
	},
}

var cycleProcessDockerInspectExec = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "GET",
		RequestURI: "/exec/123456/json",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{"ExitCode":0}`,
	},
}
