package aws_test

import (
	"testing"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/api/structs"
	"github.com/stretchr/testify/assert"
)

func TestProcessExec(t *testing.T) {
}

func TestProcessList(t *testing.T) {
	provider := StubAwsProvider(
		cycleProcessDescribeStackResources,
		cycleProcessListTasks1,
		cycleProcessListTasks2,
		cycleProcessDescribeTasks,
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

var cycleProcessListTasks1 = awsutil.Cycle{
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

var cycleProcessListTasks2 = awsutil.Cycle{
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
