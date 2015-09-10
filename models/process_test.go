package models

import (
	"fmt"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/convox/kernel/awsutil"
)

func TestRunAttached(t *testing.T) {
	s := httptest.NewServer(awsutil.NewHandler([]awsutil.Cycle{
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.ListTasks",
				Body:       `{"cluster":"convox"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"taskArns":["arn:aws:ecs:us-east-1:901416387788:task/320a8b6a-c243-47d3-a1d1-6db5dfcb3f58"]}`,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTasks",
				Body:       `{"cluster":"convox","tasks":["arn:aws:ecs:us-east-1:901416387788:task/320a8b6a-c243-47d3-a1d1-6db5dfcb3f58"]}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"tasks":[{"containerInstanceArn":"arn:aws:ecs:us-east-1:901416387788:container-instance/434bbfd7-3454-4527-a770-d3ab0fad88b6","containers":[{"containerArn":"arn:aws:ecs:us-east-1:901416387788:container/821cc6e1-b120-422c-9092-4932cce0897b","name":"worker1"}], "taskArn":"arn:aws:ecs:us-east-1:901416387788:task/320a8b6a-c243-47d3-a1d1-6db5dfcb3f58","taskDefinitionArn":"arn:aws:ecs:us-east-1:901416387788:task-definition/worker-worker1:3","lastStatus":"RUNNING"}]}`,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition",
				Body:       `{"taskDefinition":"arn:aws:ecs:us-east-1:901416387788:task-definition/worker-worker1:3"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"taskDefinition":{"volumes":[{"host":{"sourcePath":"/var/run/docker.sock"},"name":"worker-0-0"}],"containerDefinitions":[{"name":"worker1","cpu":200,"memory":256,"image":"test-image","environment":[{"name":"PROCESS","value":"worker1"}],"mountPoints":[{"sourceVolume":"worker-0-0","readOnly":false,"containerPath":"/var/run/docker.sock"}]}],"family":"worker-worker1"}}`,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "AmazonEC2ContainerServiceV20141113.DescribeContainerInstances",
				Body:       `{"cluster":"convox","containerInstances":["arn:aws:ecs:us-east-1:901416387788:container-instance/434bbfd7-3454-4527-a770-d3ab0fad88b6"]}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"containerInstances":[{"containerInstanceArn":"arn:aws:ecs:us-east-1:901416387788:container-instance/434bbfd7-3454-4527-a770-d3ab0fad88b6","ec2InstanceId":"i-8e94fa67"}]}`,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "",
				Body:       `Action=DescribeInstances&Filter.1.Name=instance-id&Filter.1.Value.1=i-8e94fa67&Version=2015-03-01`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `<DescribeInstancesResponse><reservationSet><item><instancesSet><item><instanceId>i-1a2b3c4d</instanceId><privateIpAddress>10.0.0.12</privateIpAddress></item></instanceSet></item></reservationSet></DescribeInstancesResponse>`,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/containers/json?filters=%7B%22label%22%3A%5B%22com.amazonaws.ecs.task-arn%3Darn%3Aaws%3Aecs%3Aus-east-1%3A901416387788%3Atask%2F320a8b6a-c243-47d3-a1d1-6db5dfcb3f58%22%2C%22com.amazonaws.ecs.container-name%3Dworker1%22%5D%7D",
				Operation:  "",
				Body:       ``,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `[{"Id": "8dfafdbc3a40","Command": "echo 1"}]`,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "",
				Body:       `Action=DescribeStacks&StackName=worker&Version=2010-05-15`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `<DescribeStacksResult><Stacks><member><StackName>worker</StackName><StackId>arn:aws:cloudformation:us-east-1:123456789:stack/worker/aaf549a0-a413-11df-adb3-5081b3858e83</StackId><CreationTime>2010-07-27T22:28:28Z</CreationTime><StackStatus>CREATE_COMPLETE</StackStatus></member></Stacks></DescribeStacksResult>`,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "DynamoDB_20120810.Query",
				Body:       `{"IndexName":"app.created","KeyConditions":{"app":{"AttributeValueList":[{"S":"worker"}],"ComparisonOperator":"EQ"}},"Limit":10,"ScanIndexForward":false,"TableName":"releases"}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"ConsumedCapacity":{"CapacityUnits":1,"TableName":"releases"},"Count":1,"Items":[{"App":{"S":"worker"},"Build":{"S":"B1234"}}],"ScannedCount":1}`,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/",
				Operation:  "",
				Body:       `Action=DescribeStacks&StackName=worker&Version=2010-05-15`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `<DescribeStacksResult><Stacks><member><StackName>worker</StackName><StackId>arn:aws:cloudformation:us-east-1:123456789:stack/worker/aaf549a0-a413-11df-adb3-5081b3858e83</StackId><CreationTime>2010-07-27T22:28:28Z</CreationTime><StackStatus>CREATE_COMPLETE</StackStatus><Outputs><member><OutputKey>Settings</OutputKey><OutputValue>worker-settings-13d5zljhvfr90</OutputValue></member></Outputs></member></Stacks></DescribeStacksResult>`,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/containers/create?hostconfig=%7B%22Binds%22%3A%5B%22%2Fvar%2Frun%2Fdocker.sock%3A%2Fvar%2Frun%2Fdocker.sock%22%5D%2C%22RestartPolicy%22%3A%7B%7D%2C%22LogConfig%22%3A%7B%7D%7D",
				Operation:  "",
				Body:       `{"AttachStderr":true,"AttachStdin":true,"AttachStdout":true,"Cmd":["sh","-c","echo hi"],"HostConfig":{"Binds":["/var/run/docker.sock:/var/run/docker.sock"],"LogConfig":{},"RestartPolicy":{}},"Image":"/worker-worker1:","OpenStdin":true,"Tty":true}`,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `{"Id":"e90e34656806"}`,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/containers/e90e34656806/attach?stderr=1&stdin=1&stdout=1&stream=1",
				Operation:  "",
				Body:       ``,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       `hello world`,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/containers/e90e34656806/start",
				Operation:  "",
				Body:       `null`, // wtf?
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       ``,
			},
		},
		awsutil.Cycle{
			Request: awsutil.Request{
				RequestURI: "/containers/e90e34656806/wait",
				Operation:  "",
				Body:       ``,
			},
			Response: awsutil.Response{
				StatusCode: 200,
				Body:       ``,
			},
		},
	}))

	defer s.Close()

	os.Setenv("AWS_REGION", "test")
	os.Setenv("AWS_ENDPOINT", s.URL)

	app, err := GetApp("worker")

	fmt.Printf("app: %+v\n", app)
	fmt.Printf("err: %+v\n", err)

	// assert.Nil(t, err, "")
	// assert.Equal(t, "worker", ps[0].App)
	// assert.Equal(t, "worker1", ps[0].Name)

	// ps[0].RunAttached("echo hi", &bytes.Buffer{})
}
