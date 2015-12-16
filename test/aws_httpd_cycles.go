package test

import "github.com/convox/rack/api/awsutil"

// $ aws cloudformation describe-stack-resources --stack-name httpd --debug
func HttpdDescribeStackResourcesCycle() awsutil.Cycle {
	return awsutil.Cycle{
		awsutil.Request{"/", "", `Action=DescribeStackResources&StackName=httpd&Version=2010-05-15`},
		awsutil.Response{200, httpdDescribeStackResourcesResponse()},
	}
}

// $ aws ecs describe-services --cluster convox-Cluster-1NCWX9EC0JOV4 --services httpd-web-SRZPVERKQOL
func HttpdDescribeServicesCycle() awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/",
			Operation:  "AmazonEC2ContainerServiceV20141113.DescribeServices",
			Body:       `{"cluster":"convox-test", "services":["arn:aws:ecs:us-west-2:901416387788:service/httpd-web-SRZPVERKQOL"]}`,
		},
		Response: awsutil.Response{
			StatusCode: 200,
			Body:       HttpdDescribeServicesResponse(),
		},
	}
}

func HttpdDescribeServicesResponse() string {
	return `{
    "services": [
        {
            "status": "ACTIVE", 
            "taskDefinition": "arn:aws:ecs:us-west-2:901416387788:task-definition/httpd-web:1", 
            "pendingCount": 0, 
            "loadBalancers": [
                {
                    "containerName": "web", 
                    "containerPort": 80, 
                    "loadBalancerName": "httpd"
                }
            ], 
            "roleArn": "arn:aws:iam::901416387788:role/httpd-ServiceRole-1K9DGK9MPLXZO", 
            "desiredCount": 1, 
            "serviceName": "httpd-web-SRZPVERKQOL", 
            "clusterArn": "arn:aws:ecs:us-west-2:901416387788:cluster/convox-Cluster-1NCWX9EC0JOV4", 
            "serviceArn": "arn:aws:ecs:us-west-2:901416387788:service/httpd-web-SRZPVERKQOL", 
            "deployments": [
                {
                    "status": "PRIMARY", 
                    "pendingCount": 0, 
                    "createdAt": 1450120203.716, 
                    "desiredCount": 1, 
                    "taskDefinition": "arn:aws:ecs:us-west-2:901416387788:task-definition/httpd-web:1", 
                    "updatedAt": 1450120203.716, 
                    "id": "ecs-svc/9223370586734572091", 
                    "runningCount": 1
                }
            ], 
            "events": [
                {
                    "message": "(service httpd-web-SRZPVERKQOL) has reached a steady state.", 
                    "id": "7a8cd970-01ff-4d34-aa34-fa0deff70e48", 
                    "createdAt": 1450120334.038
                }, 
                {
                    "message": "(service httpd-web-SRZPVERKQOL) registered 1 instances in (elb httpd)", 
                    "id": "6f89b306-87b3-41a4-8f92-68491f4941a7", 
                    "createdAt": 1450120220.028
                }, 
                {
                    "message": "(service httpd-web-SRZPVERKQOL) deregistered 1 instances in (elb httpd)", 
                    "id": "16099f01-abaa-4389-b7e6-e7c4b1b78c30", 
                    "createdAt": 1450120209.495
                }, 
                {
                    "message": "(service httpd-web-SRZPVERKQOL) has started 1 tasks: (task 04394454-6c7e-4879-a826-d576a47c7fdc).", 
                    "id": "2636ed0e-05c0-4945-93da-cf44f964cb3d", 
                    "createdAt": 1450120209.495
                }
            ], 
            "runningCount": 1
        }
    ], 
    "failures": []
}`
}

func httpdDescribeStackResourcesResponse() string {
	return `<DescribeStackResourcesResult>
    <StackResources>
      <member>
        <Timestamp>2015-12-14T19:10:00.038Z</Timestamp>
        <ResourceStatus>CREATE_COMPLETE</ResourceStatus>
        <StackId>arn:aws:cloudformation:us-west-2:901416387788:stack/httpd/58c3c540-a295-11e5-bb58-50d50031c6e0</StackId>
        <LogicalResourceId>Balancer</LogicalResourceId>
        <StackName>httpd</StackName>
        <PhysicalResourceId>httpd</PhysicalResourceId>
        <ResourceType>AWS::ElasticLoadBalancing::LoadBalancer</ResourceType>
      </member>
      <member>
        <Timestamp>2015-12-14T19:09:55.792Z</Timestamp>
        <ResourceStatus>CREATE_COMPLETE</ResourceStatus>
        <StackId>arn:aws:cloudformation:us-west-2:901416387788:stack/httpd/58c3c540-a295-11e5-bb58-50d50031c6e0</StackId>
        <LogicalResourceId>BalancerSecurityGroup</LogicalResourceId>
        <StackName>httpd</StackName>
        <PhysicalResourceId>sg-d5fdc1b1</PhysicalResourceId>
        <ResourceType>AWS::EC2::SecurityGroup</ResourceType>
      </member>
      <member>
        <Timestamp>2015-12-14T19:04:00.683Z</Timestamp>
        <ResourceStatus>CREATE_COMPLETE</ResourceStatus>
        <StackId>arn:aws:cloudformation:us-west-2:901416387788:stack/httpd/58c3c540-a295-11e5-bb58-50d50031c6e0</StackId>
        <LogicalResourceId>CustomTopic</LogicalResourceId>
        <StackName>httpd</StackName>
        <PhysicalResourceId>httpd-CustomTopic-1XKD0E3PM22G6</PhysicalResourceId>
        <ResourceType>AWS::Lambda::Function</ResourceType>
      </member>
      <member>
        <Timestamp>2015-12-14T19:03:56.985Z</Timestamp>
        <ResourceStatus>CREATE_COMPLETE</ResourceStatus>
        <StackId>arn:aws:cloudformation:us-west-2:901416387788:stack/httpd/58c3c540-a295-11e5-bb58-50d50031c6e0</StackId>
        <LogicalResourceId>CustomTopicRole</LogicalResourceId>
        <StackName>httpd</StackName>
        <PhysicalResourceId>httpd-CustomTopicRole-EOP193O880F7</PhysicalResourceId>
        <ResourceType>AWS::IAM::Role</ResourceType>
      </member>
      <member>
        <Timestamp>2015-12-14T19:04:07.586Z</Timestamp>
        <ResourceStatus>CREATE_COMPLETE</ResourceStatus>
        <StackId>arn:aws:cloudformation:us-west-2:901416387788:stack/httpd/58c3c540-a295-11e5-bb58-50d50031c6e0</StackId>
        <LogicalResourceId>Kinesis</LogicalResourceId>
        <StackName>httpd</StackName>
        <PhysicalResourceId>httpd-Kinesis-1MM7WF087XN4A</PhysicalResourceId>
        <ResourceType>AWS::Kinesis::Stream</ResourceType>
      </member>
      <member>
        <Timestamp>2015-12-14T19:09:43.075Z</Timestamp>
        <ResourceStatus>CREATE_COMPLETE</ResourceStatus>
        <StackId>arn:aws:cloudformation:us-west-2:901416387788:stack/httpd/58c3c540-a295-11e5-bb58-50d50031c6e0</StackId>
        <LogicalResourceId>LogsAccess</LogicalResourceId>
        <StackName>httpd</StackName>
        <PhysicalResourceId>AKIAJKMHFYRRAA6FNNNQ</PhysicalResourceId>
        <ResourceType>AWS::IAM::AccessKey</ResourceType>
      </member>
      <member>
        <Timestamp>2015-12-14T19:09:40.392Z</Timestamp>
        <ResourceStatus>CREATE_COMPLETE</ResourceStatus>
        <StackId>arn:aws:cloudformation:us-west-2:901416387788:stack/httpd/58c3c540-a295-11e5-bb58-50d50031c6e0</StackId>
        <LogicalResourceId>LogsUser</LogicalResourceId>
        <StackName>httpd</StackName>
        <PhysicalResourceId>httpd-LogsUser-LMMTUSLEH0J3</PhysicalResourceId>
        <ResourceType>AWS::IAM::User</ResourceType>
      </member>
      <member>
        <Timestamp>2015-12-14T19:03:57.144Z</Timestamp>
        <ResourceStatus>CREATE_COMPLETE</ResourceStatus>
        <StackId>arn:aws:cloudformation:us-west-2:901416387788:stack/httpd/58c3c540-a295-11e5-bb58-50d50031c6e0</StackId>
        <LogicalResourceId>ServiceRole</LogicalResourceId>
        <StackName>httpd</StackName>
        <PhysicalResourceId>httpd-ServiceRole-1K9DGK9MPLXZO</PhysicalResourceId>
        <ResourceType>AWS::IAM::Role</ResourceType>
      </member>
      <member>
        <Timestamp>2015-12-14T19:04:17.594Z</Timestamp>
        <ResourceStatus>CREATE_COMPLETE</ResourceStatus>
        <StackId>arn:aws:cloudformation:us-west-2:901416387788:stack/httpd/58c3c540-a295-11e5-bb58-50d50031c6e0</StackId>
        <LogicalResourceId>Settings</LogicalResourceId>
        <StackName>httpd</StackName>
        <PhysicalResourceId>httpd-settings-gclooujvfwww</PhysicalResourceId>
        <ResourceType>AWS::S3::Bucket</ResourceType>
      </member>
      <member>
        <Timestamp>2015-12-14T19:10:05.312Z</Timestamp>
        <ResourceStatus>CREATE_COMPLETE</ResourceStatus>
        <StackId>arn:aws:cloudformation:us-west-2:901416387788:stack/httpd/58c3c540-a295-11e5-bb58-50d50031c6e0</StackId>
        <LogicalResourceId>WebECSService</LogicalResourceId>
        <StackName>httpd</StackName>
        <PhysicalResourceId>arn:aws:ecs:us-west-2:901416387788:service/httpd-web-SRZPVERKQOL</PhysicalResourceId>
        <ResourceType>Custom::ECSService</ResourceType>
      </member>
      <member>
        <Timestamp>2015-12-14T19:09:44.343Z</Timestamp>
        <ResourceStatus>CREATE_COMPLETE</ResourceStatus>
        <StackId>arn:aws:cloudformation:us-west-2:901416387788:stack/httpd/58c3c540-a295-11e5-bb58-50d50031c6e0</StackId>
        <LogicalResourceId>WebECSTaskDefinition</LogicalResourceId>
        <StackName>httpd</StackName>
        <PhysicalResourceId>arn:aws:ecs:us-west-2:901416387788:task-definition/httpd-web:1</PhysicalResourceId>
        <ResourceType>Custom::ECSTaskDefinition</ResourceType>
      </member>
    </StackResources>
  </DescribeStackResourcesResult>`
}
