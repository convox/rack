package aws_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/provider/aws"
	"github.com/stretchr/testify/assert"
)

func TestProcessExec(t *testing.T) {
	provider := StubAwsProvider(
		cycleProcessListTasksAll,
		cycleProcessDescribeTasks,
		cycleProcessDescribeContainerInstances,
		cycleProcessDescribeInstances,
	)
	defer provider.Close()

	d := stubDocker(
		cycleProcessDockerListContainers,
		cycleProcessDockerCreateExec,
		cycleProcessDockerStartExec,
		cycleProcessDockerResizeExec,
		cycleProcessDockerInspectExec,
	)
	defer d.Close()

	buf := &bytes.Buffer{}

	err := provider.ProcessExec("myapp", "5850760f0845", "ls -la", buf, structs.ProcessExecOptions{
		Height: 10,
		Width:  20,
	})

	assert.Nil(t, err)
	assert.Equal(t, []byte(fmt.Sprintf("foo%s%d\n", aws.StatusCodePrefix, 0)), buf.Bytes())
}

func TestProcessList(t *testing.T) {
	provider := StubAwsProvider(
		cycleProcessDescribeStackResources,
		cycleProcessListTasksByService,
		cycleProcessListTasksByStarted,
		cycleProcessDescribeTasksAll,
		cycleProcessDescribeTaskDefinition1,
		cycleProcessDescribeTaskDefinition2,
		cycleProcessDescribeContainerInstances,
		cycleProcessDescribeContainerInstances,
		cycleProcessDescribeInstances,
		cycleProcessDescribeInstances,
	)
	defer provider.Close()

	s, err := provider.ProcessList("myapp")

	ps := structs.Processes{
		structs.Process{
			ID:       "5850760f0845",
			App:      "myapp",
			Name:     "web",
			Release:  "R1234",
			Command:  "sh -c foo",
			Host:     "10.0.1.244",
			Image:    "778743527532.dkr.ecr.us-east-1.amazonaws.com/convox-myapp-nkdecwppkq:web.BMPBJLITPZT",
			Instance: "i-5bc45dc2",
			Ports:    []string{},
		},
		structs.Process{
			ID:       "5850760f0846",
			App:      "myapp",
			Name:     "web",
			Release:  "R1234",
			Command:  "",
			Host:     "10.0.1.244",
			Image:    "778743527532.dkr.ecr.us-east-1.amazonaws.com/convox-myapp-nkdecwppkq:web.BMPBJLITPZT",
			Instance: "i-5bc45dc2",
			Ports:    []string{},
		},
	}

	assert.Nil(t, err)
	assert.EqualValues(t, ps, s)
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
		Body:       `Action=DescribeInstances&InstanceId.1=i-5bc45dc2&Version=2016-04-01`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<?xml version="1.0" encoding="UTF-8"?>
			<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-04-01/">
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
			</DescribeInstancesRepsonse>
		}`,
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
							<PhysicalResourceId>arn:aws:ecs:us-east-1:778743527532:service/convox-myapp-ServiceDatabase-1I2PTXAZ5ECRD</PhysicalResourceId>
							<ResourceStatus>UPDATE_COMPLETE</ResourceStatus>
							<StackId>arn:aws:cloudformation:us-east-1:778743527532:stack/convox-myapp/5c05e0c0-6e10-11e6-8a4e-50fae98a10d2</StackId>
							<StackName>convox-myapp</StackName>
							<LogicalResourceId>ServiceDatabase</LogicalResourceId>
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

var cycleProcessListTasksByService = awsutil.Cycle{
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

var cycleProcessDockerListContainers = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/containers/json?all=1&filters=%7B%22label%22%3A%5B%22com.amazonaws.ecs.task-arn%3Darn%3Aaws%3Aecs%3Aus-east-1%3A778743527532%3Atask%2F50b8de99-f94f-4ecd-a98f-5850760f0845%22%5D%7D",
		Body:       ``,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `[
			{
				"Id": "8dfafdbc3a40",
				"Names":["/boring_feynman"],
				"Image": "ubuntu:latest",
				"ImageID": "d74508fb6632491cea586a1fd7d748dfc5274cd6fdfedee309ecdcbc2bf5cb82",
				"Command": "echo 1",
				"Created": 1367854155,
				"State": "Exited",
				"Status": "Exit 0",
				"Ports": [{"PrivatePort": 2222, "PublicPort": 3333, "Type": "tcp"}]
			}
		]`,
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
			"ErrorStream": {},
			"InputStream": {},
			"OutputStream": {},
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
		RequestURI: "/exec/123456/json",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{"ExitCode":0}`,
	},
}
