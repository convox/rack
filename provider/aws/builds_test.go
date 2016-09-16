package aws_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/api/models"
	"github.com/convox/rack/api/structs"

	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("RACK", "convox")
	os.Setenv("DYNAMO_BUILDS", "convox-builds")
	os.Setenv("DYNAMO_RELEASES", "convox-releases")
	models.PauseNotifications = true
}

func TestBuildGet(t *testing.T) {
	provider := StubAwsProvider(
		cycleBuildGetItem,
	)
	defer provider.Close()

	b, err := provider.BuildGet("httpd", "B123")

	assert.Nil(t, err)
	assert.EqualValues(t, &structs.Build{
		Id:       "BAFVEWUCAYT",
		App:      "httpd",
		Logs:     "object:///test/foo",
		Manifest: "version: \"2\"\nnetworks: {}\nservices:\n  web:\n    build: {}\n    command: null\n    image: httpd\n    ports:\n    - 80:80\n",
		Release:  "RVWOJNKRAXU",
		Status:   "complete",
		Started:  time.Unix(1459780456, 178278576).UTC(),
		Ended:    time.Unix(1459780542, 440881687).UTC(),
		Tags:     map[string]string{},
	}, b)
}

func TestBuildDelete(t *testing.T) {
	provider := StubAwsProvider(
		cycleBuildGetItem,
		cycleBuildDescribeStacks,
		cycleBuildDescribeStacks,
		cycleReleaseGetItem,
		cycleBuildDeleteItem,
		cycleBuildBatchDeleteImage,
	)
	defer provider.Close()

	b, err := provider.BuildDelete("httpd", "B123")

	assert.Nil(t, err)
	assert.EqualValues(t, &structs.Build{
		Id:       "BAFVEWUCAYT",
		App:      "httpd",
		Logs:     "object:///test/foo",
		Manifest: "version: \"2\"\nnetworks: {}\nservices:\n  web:\n    build: {}\n    command: null\n    image: httpd\n    ports:\n    - 80:80\n",
		Release:  "RVWOJNKRAXU",
		Status:   "complete",
		Started:  time.Unix(1459780456, 178278576).UTC(),
		Ended:    time.Unix(1459780542, 440881687).UTC(),
		Tags:     map[string]string{},
	}, b)
}

func TestBuildExport(t *testing.T) {
	provider := StubAwsProvider(
		cycleBuildGetItem,
		cycleBuildDescribeStacks,
		cycleBuildDescribeRepositories,
		cycleBuildGetAuthorizationToken,
	)
	defer provider.Close()

	d := stubDocker(
		cycleBuildDockerLogin,
		cycleBuildDockerPull,
		cycleBuildDockerSave,
	)
	defer d.Close()

	buf := &bytes.Buffer{}

	err := provider.BuildExport("httpd", "B123", buf)
	assert.Nil(t, err)

	gz, err := gzip.NewReader(buf)
	assert.Nil(t, err)

	tr := tar.NewReader(gz)

	h, err := tr.Next()
	assert.Nil(t, err)
	assert.Equal(t, "build.json", h.Name)
	assert.Equal(t, int64(400), h.Size)

	data, err := ioutil.ReadAll(tr)
	assert.Nil(t, err)

	var build structs.Build
	err = json.Unmarshal(data, &build)
	assert.Nil(t, err)
	assert.Equal(t, "BAFVEWUCAYT", build.Id)
	assert.Equal(t, "httpd", build.App)
	assert.Equal(t, "RVWOJNKRAXU", build.Release)

	h, err = tr.Next()
	assert.Nil(t, err)
	assert.Equal(t, "web.BAFVEWUCAYT.tar", h.Name)
	assert.Equal(t, int64(13), h.Size)

	h, err = tr.Next()
	assert.Equal(t, io.EOF, err)
	assert.Nil(t, h)
}

func TestBuildImport(t *testing.T) {
	provider := StubAwsProvider(
		cycleBuildDescribeStacks,
		cycleBuildDescribeRepositories,
		cycleBuildGetAuthorizationToken,
		cycleBuildDescribeStacks,
		cycleEnvironmentGet,
		cycleBuildDescribeStacks,
		cycleBuildPutItem,
		cycleBuildDescribeStacks,
		cycleEnvironmentPut,
		cycleBuildReleasePutItem,
	)
	defer provider.Close()

	d := stubDocker(
		cycleBuildDockerLogin,
		cycleBuildDockerLoad,
		cycleBuildDockerTag,
		cycleBuildDockerPush,
	)
	defer d.Close()

	build := &structs.Build{
		Id:      "B12345",
		App:     "httpd",
		Release: "R23456",
	}

	data, err := json.Marshal(build)
	assert.Nil(t, err)

	buf := &bytes.Buffer{}

	gz := gzip.NewWriter(buf)
	tw := tar.NewWriter(gz)

	err = tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "build.json",
		Size:     int64(len(data)),
	})
	assert.Nil(t, err)

	n, err := tw.Write(data)
	assert.Nil(t, err)
	assert.Equal(t, 177, n)

	lbuf := &bytes.Buffer{}

	ltw := tar.NewWriter(lbuf)

	data = []byte(`[{"RepoTags":["test-tag"]}]`)

	err = ltw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "manifest.json",
		Size:     int64(len(data)),
	})
	assert.Nil(t, err)

	n, err = ltw.Write(data)
	assert.Nil(t, err)
	assert.Equal(t, 27, n)

	err = ltw.Close()
	assert.Nil(t, err)

	err = tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "web.B12345.tar",
		Size:     int64(lbuf.Len()),
	})
	assert.Nil(t, err)

	n, err = tw.Write(lbuf.Bytes())
	assert.Nil(t, err)
	assert.Equal(t, 2048, n)

	err = tw.Close()
	assert.Nil(t, err)

	err = gz.Close()
	assert.Nil(t, err)

	build, err = provider.BuildImport("httpd", buf)
	assert.Nil(t, err)
	assert.Equal(t, "B12345", build.Id)
	assert.Equal(t, "httpd", build.App)
	assert.Equal(t, "R23456", build.Release)
}

func TestBuildList(t *testing.T) {
	provider := StubAwsProvider(
		cycleBuildDescribeStacks,
		cycleBuildQuery,
	)
	defer provider.Close()

	b, err := provider.BuildList("httpd", 20)

	assert.Nil(t, err)
	assert.EqualValues(t, structs.Builds{
		structs.Build{
			Id:       "BHINCLZYYVN",
			App:      "httpd",
			Logs:     "",
			Manifest: "web:\n  image: httpd\n  ports:\n  - 80:80\n",
			Release:  "RVFETUHHKKD",
			Status:   "complete",
			Started:  time.Unix(1459780456, 178278576).UTC(),
			Ended:    time.Unix(1459780542, 440881687).UTC(),
			Tags:     map[string]string{},
		},
		structs.Build{
			Id:       "BNOARQMVHUO",
			App:      "httpd",
			Logs:     "",
			Manifest: "web:\n  image: httpd\n  ports:\n  - 80:80\n",
			Release:  "RFVZFLKVTYO",
			Status:   "complete",
			Started:  time.Unix(1459780456, 178278576).UTC(),
			Ended:    time.Unix(1459780542, 440881687).UTC(),
			Tags:     map[string]string{},
		},
	}, b)
}

func TestBuildLogsRunning(t *testing.T) {
	provider := StubAwsProvider(
		cycleBuildGetItemRunning,
		cycleBuildDescribeTasks,
		cycleBuildDescribeContainerInstances,
		cycleBuildDescribeInstances,
	)
	defer provider.Close()

	d := stubDocker(
		cycleBuildDockerListContainers,
		cycleBuildDockerLogs,
	)
	defer d.Close()

	buf := &bytes.Buffer{}

	err := provider.BuildLogs("httpd", "B123", buf)

	assert.Nil(t, err)
	assert.Equal(t, "RUNNING: docker pull httpd", buf.String())
}

func TestBuildLogsNotRunning(t *testing.T) {
	provider := StubAwsProvider(
		cycleBuildGetItem,
		cycleBuildFetchLogs,
	)
	defer provider.Close()

	buf := &bytes.Buffer{}

	err := provider.BuildLogs("httpd", "B123", buf)

	assert.Nil(t, err)
	assert.Equal(t, "RUNNING: docker pull httpd", buf.String())
}

var cycleBuildBatchDeleteImage = awsutil.Cycle{
	awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerRegistry_V20150921.BatchDeleteImage",
		Body: `{
			"imageIds": [
				{
					"imageTag": "web.BAFVEWUCAYT"
				}
			],
			"registryId": "132866487567",
			"repositoryName": "convox-httpd-hqvvfosgxt"
		}`,
	},
	awsutil.Response{
		StatusCode: 200,
		Body:       `{}`,
	},
}

var cycleBuildDeleteItem = awsutil.Cycle{
	awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.DeleteItem",
		Body: `{
			"Key": {
				"id": {
					"S": "B123"
				}
			},
			"TableName": "convox-builds"
		}`,
	},
	awsutil.Response{
		StatusCode: 200,
		Body:       `{}`,
	},
}

var cycleBuildDescribeContainerInstances = awsutil.Cycle{
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

var cycleBuildDescribeInstances = awsutil.Cycle{
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

var cycleBuildDescribeRepositories = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerRegistry_V20150921.DescribeRepositories",
		Body: `{
			"repositoryNames": [
				"convox-httpd-hqvvfosgxt"
			]
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"repositories": [
				{
					"registryId": "778743527532",
					"repositoryName": "convox-rails-sslibosttb",
					"repositoryArn": "arn:aws:ecr:us-east-1:778743527532:repository/convox-rails-sslibosttb",
					"repositoryUri": "778743527532.dkr.ecr.us-east-1.amazonaws.com/convox-rails-sslibosttb"
				}
			]
		}`,
	},
}

var cycleBuildDescribeStacks = awsutil.Cycle{
	awsutil.Request{"/", "", `Action=DescribeStacks&StackName=convox-httpd&Version=2010-05-15`},
	awsutil.Response{200, `
		<DescribeStacksResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
			<DescribeStacksResult>
				<Stacks>
					<member>
						<Tags>
							<member>
								<Value>httpd</Value>
								<Key>Name</Key>
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
								<Value>convox</Value>
								<Key>Rack</Key>
							</member>
						</Tags>
						<StackId>arn:aws:cloudformation:us-east-1:132866487567:stack/convox-httpd/53df3c30-f763-11e5-bd5d-50d5cd148236</StackId>
						<StackStatus>UPDATE_COMPLETE</StackStatus>
						<StackName>convox-httpd</StackName>
						<LastUpdatedTime>2016-03-31T17:12:16.275Z</LastUpdatedTime>
						<NotificationARNs/>
						<CreationTime>2016-03-31T17:09:28.583Z</CreationTime>
						<Parameters>
							<member>
								<ParameterValue>https://convox-httpd-settings-139bidzalmbtu.s3.amazonaws.com/releases/RVFETUHHKKD/env</ParameterValue>
								<ParameterKey>Environment</ParameterKey>
							</member>
							<member>
								<ParameterValue/>
								<ParameterKey>WebPort80Certificate</ParameterKey>
							</member>
							<member>
								<ParameterValue>No</ParameterValue>
								<ParameterKey>WebPort80ProxyProtocol</ParameterKey>
							</member>
							<member>
								<ParameterValue>256</ParameterValue>
								<ParameterKey>WebCpu</ParameterKey>
							</member>
							<member>
								<ParameterValue>256</ParameterValue>
								<ParameterKey>WebMemory</ParameterKey>
							</member>
							<member>
								<ParameterValue></ParameterValue>
								<ParameterKey>Key</ParameterKey>
							</member>
							<member>
								<ParameterValue/>
								<ParameterKey>Repository</ParameterKey>
							</member>
							<member>
								<ParameterValue>80</ParameterValue>
								<ParameterKey>WebPort80Balancer</ParameterKey>
							</member>
							<member>
								<ParameterValue>56694</ParameterValue>
								<ParameterKey>WebPort80Host</ParameterKey>
							</member>
							<member>
								<ParameterValue>vpc-f8006b9c</ParameterValue>
								<ParameterKey>VPC</ParameterKey>
							</member>
							<member>
								<ParameterValue>1</ParameterValue>
								<ParameterKey>WebDesiredCount</ParameterKey>
							</member>
							<member>
								<ParameterValue>convox-Cluster-1E4XJ0PQWNAYS</ParameterValue>
								<ParameterKey>Cluster</ParameterKey>
							</member>
							<member>
								<ParameterValue>subnet-d4e85cfe,subnet-103d5a66,subnet-57952a0f</ParameterValue>
								<ParameterKey>SubnetsPrivate</ParameterKey>
							</member>
							<member>
								<ParameterValue>RVFETUHHKKD</ParameterValue>
								<ParameterKey>Release</ParameterKey>
							</member>
							<member>
								<ParameterValue>No</ParameterValue>
								<ParameterKey>WebPort80Secure</ParameterKey>
							</member>
							<member>
								<ParameterValue>subnet-13de3139,subnet-b5578fc3,subnet-21c13379</ParameterValue>
								<ParameterKey>Subnets</ParameterKey>
							</member>
							<member>
								<ParameterValue>20160330143438-command-exec-form</ParameterValue>
								<ParameterKey>Version</ParameterKey>
							</member>
							<member>
								<ParameterValue>Yes</ParameterValue>
								<ParameterKey>Private</ParameterKey>
							</member>
						</Parameters>
						<DisableRollback>false</DisableRollback>
						<Capabilities>
							<member>CAPABILITY_IAM</member>
						</Capabilities>
						<Outputs>
							<member>
								<OutputValue>httpd-web-7E5UPCM-1241527783.us-east-1.elb.amazonaws.com</OutputValue>
								<OutputKey>BalancerWebHost</OutputKey>
							</member>
							<member>
								<OutputValue>convox-httpd-Kinesis-1MAP0GJ6RITJF</OutputValue>
								<OutputKey>Kinesis</OutputKey>
							</member>
							<member>
								<OutputValue>convox-httpd-LogGroup-L4V203L35WRM</OutputValue>
								<OutputKey>LogGroup</OutputKey>
							</member>
							<member>
								<OutputValue>132866487567</OutputValue>
								<OutputKey>RegistryId</OutputKey>
							</member>
							<member>
								<OutputValue>convox-httpd-hqvvfosgxt</OutputValue>
								<OutputKey>RegistryRepository</OutputKey>
							</member>
							<member>
								<OutputValue>convox-httpd-settings-139bidzalmbtu</OutputValue>
								<OutputKey>Settings</OutputKey>
							</member>
							<member>
								<OutputValue>80</OutputValue>
								<OutputKey>WebPort80Balancer</OutputKey>
							</member>
							<member>
								<OutputValue>httpd-web-7E5UPCM</OutputValue>
								<OutputKey>WebPort80BalancerName</OutputKey>
							</member>
						</Outputs>
					</member>
				</Stacks>
			</DescribeStacksResult>
			<ResponseMetadata>
				<RequestId>d5220387-f76d-11e5-912c-531803b112a4</RequestId>
			</ResponseMetadata>
		</DescribeStacksResponse>
	`},
}

var cycleBuildDescribeTasks = awsutil.Cycle{
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

var cycleBuildGetAuthorizationToken = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerRegistry_V20150921.GetAuthorizationToken",
		Body: `{
			"registryIds": [
				"778743527532"
			]
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"authorizationData": [
				{
					"authorizationToken": "dXNlcjoxMjM0NQo=",
					"expiresAt": 1473039114.46,
					"proxyEndpoint": "https://778743527532.dkr.ecr.us-east-1.amazonaws.com"
				}
			]
		}`,
	},
}

var cycleBuildGetItem = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.GetItem",
		Body: `{
			"ConsistentRead": true,
			"Key": {
				"id": {
					"S": "B123"
				}
			},
			"TableName": "convox-builds"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"Item": {
				"status": {
					"S": "complete"
				},
				"created": {
					"S": "20160404.143416.178278576"
				},
				"app": {
					"S": "httpd"
				},
				"logs": {
					"S": "object:///test/foo"
				},
				"manifest": {
					"S": "version: \"2\"\nnetworks: {}\nservices:\n  web:\n    build: {}\n    command: null\n    image: httpd\n    ports:\n    - 80:80\n"
				},
				"ended": {
					"S": "20160404.143542.440881687"
				},
				"release": {
					"S": "RVWOJNKRAXU"
				},
				"id": {
					"S": "BAFVEWUCAYT"
				}
			}
		}`,
	},
}

var cycleBuildGetItemRunning = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.GetItem",
		Body: `{
			"ConsistentRead": true,
			"Key": {
				"id": {
					"S": "B123"
				}
			},
			"TableName": "convox-builds"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"Item": {
				"status": {
					"S": "running"
				},
				"created": {
					"S": "20160404.143416.178278576"
				},
				"app": {
					"S": "httpd"
				},
				"manifest": {
					"S": "version: \"2\"\nnetworks: {}\nservices:\n  web:\n    build: {}\n    command: null\n    image: httpd\n    ports:\n    - 80:80\n"
				},
				"ended": {
					"S": "20160404.143542.440881687"
				},
				"release": {
					"S": "RVWOJNKRAXU"
				},
				"id": {
					"S": "BAFVEWUCAYT"
				},
				"tags": {
					"B": "eyJ0YXNrIjoiYXJuOmF3czplY3M6dXMtZWFzdC0xOjc3ODc0MzUyNzUzMjp0YXNrLzUwYjhkZTk5LWY5NGYtNGVjZC1hOThmLTU4NTA3NjBmMDg0NSJ9Cg=="
				}
			}
		}`,
	},
}

var cycleBuildFetchLogs = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/convox-settings/test/foo",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       "RUNNING: docker pull httpd",
	},
}

var cycleBuildPutItem = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.PutItem",
		Body: `{
			"Item": {
				"app": {
					"S": "httpd"
				},
				"created": {
					"S": "20160904.223813.000000000"
				},
				"description": {
					"S": "imported"
				},
				"ended": {
					"S": "20160904.224132.000000000"
				},
				"id": {
					"S": "B12345"
				},
				"release": {
					"S": "R23456"
				},
				"status": {
					"S": "complete"
				}
			},
			"TableName": "convox-builds"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{}`,
	},
}

var cycleBuildQuery = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.Query",
		Body:       `{"IndexName":"app.created","KeyConditions":{"app":{"AttributeValueList":[{"S":"httpd"}],"ComparisonOperator":"EQ"}},"Limit":20,"ScanIndexForward":false,"TableName":"convox-builds"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"Count": 2,
			"Items": [
				{
					"id": {"S":"BHINCLZYYVN"},
					"app": {"S":"httpd"},
					"created": {"S":"20160404.143416.178278576"},
					"ended": {"S":"20160404.143542.440881687"},
					"env": {"S":"foo=bar"},
					"manifest": {"S":"web:\n  image: httpd\n  ports:\n  - 80:80\n"},
					"release": {"S":"RVFETUHHKKD"},
					"status": {"S":"complete"}
				},
				{
					"id": {"S":"BNOARQMVHUO"},
					"app": {"S":"httpd"},
					"created": {"S":"20160404.143416.178278576"},
					"ended": {"S":"20160404.143542.440881687"},
					"env": {"S":"foo=bar"},
					"manifest": {"S":"web:\n  image: httpd\n  ports:\n  - 80:80\n"},
					"release": {"S":"RFVZFLKVTYO"},
					"status": {"S":"complete"}
				}
			],
			"ScannedCount":2
		}`,
	},
}

var cycleBuildReleasePutItem = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.PutItem",
		Body: `{
			"Item": {
				"app": {
					"S": "httpd"
				},
				"build": {
					"S": "B12345"
				},
				"created": {
					"S": "20160904.223813.000000000"
				},
				"env": {
					"S": "BAZ=qux\nFOO=bar"
				},
				"id": {
					"S": "R23456"
				}
			},
			"TableName": "convox-releases"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{}`,
	},
}

var cycleBuildDockerListContainers = awsutil.Cycle{
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

var cycleBuildDockerLogs = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/containers/8dfafdbc3a40/logs?follow=1&stderr=1&stdout=1&tail=all",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       "\x01\x00\x00\x00\x00\x00\x00\x1aRUNNING: docker pull httpd",
	},
}

var cycleBuildDockerLoad = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/v1.24/images/load?quiet=1",
		Body:       "//",
	},
	Response: awsutil.Response{
		StatusCode: 200,
	},
}

var cycleBuildDockerLogin = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/v1.24/auth",
		Body: `{
			"password": "12345\n",
			"serveraddress": "778743527532.dkr.ecr.us-east-1.amazonaws.com/convox-rails-sslibosttb",
			"username": "user"
		}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `{
			"Status": "Login Successful",
			"IdentityToken": "foo"
		}`,
	},
}

var cycleBuildDockerPull = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/v1.24/images/create?fromImage=778743527532.dkr.ecr.us-east-1.amazonaws.com%2Fconvox-rails-sslibosttb&tag=web.BAFVEWUCAYT",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{}`,
	},
}

var cycleBuildDockerPush = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/v1.24/images/778743527532.dkr.ecr.us-east-1.amazonaws.com/convox-rails-sslibosttb/push?tag=web.B12345",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{}`,
	},
}

var cycleBuildDockerSave = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/v1.24/images/get?names=778743527532.dkr.ecr.us-east-1.amazonaws.com%2Fconvox-rails-sslibosttb%3Aweb.BAFVEWUCAYT",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `should-be-tar`,
	},
}

var cycleBuildDockerTag = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/v1.24/images/test-tag/tag?repo=778743527532.dkr.ecr.us-east-1.amazonaws.com%2Fconvox-rails-sslibosttb&tag=web.B12345",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{}`,
	},
}

var cycleEnvironmentGet = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/convox-httpd-settings-139bidzalmbtu/env",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       "FOO=bar\nBAZ=qux",
	},
}

var cycleEnvironmentPut = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/convox-httpd-settings-139bidzalmbtu/releases/R23456/env",
		Body:       "BAZ=qux\nFOO=bar",
	},
	Response: awsutil.Response{
		StatusCode: 200,
	},
}
