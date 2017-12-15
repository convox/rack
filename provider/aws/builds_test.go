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

	"github.com/convox/rack/structs"
	"github.com/convox/rack/test/awsutil"

	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("RACK", "convox")
	os.Setenv("DYNAMO_BUILDS", "convox-builds")
	os.Setenv("DYNAMO_RELEASES", "convox-releases")
	// models.PauseNotifications = true
}

func TestBuildGet(t *testing.T) {
	provider := StubAwsProvider(
		cycleBuildGetItem,
	)
	defer provider.Close()

	b, err := provider.BuildGet("httpd", "B123")

	assert.NoError(t, err)
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

// func TestBuildCreate(t *testing.T) {
//   provider := StubAwsProvider(
//     cycleBuildDescribeStacks,
//     cycleBuildDescribeStacks,
//     cycleBuildPutItemCreate,
//     cycleBuildDescribeStackResources,
//     cycleEnvironmentGetRack,
//     cycleRegistryListRegistries,
//     cycleRegistryGetRegistry,
//     cycleRegistryDecrypt,
//     cycleBuildGetAuthorizationTokenPrivate1,
//     cycleBuildRunTask,
//     cycleBuildGetItem,
//     cycleBuildDescribeStacks,
//     cycleBuildPutItemCreate2,
//     cycleBuildDescribeTasks,
//     cycleBuildDescribeContainerInstances,
//     cycleBuildDescribeInstances,
//     cycleBuildDescribeStacks,
//     cycleBuildQuery150,
//   )
//   defer provider.Close()

//   d := stubDocker(
//     cycleBuildDockerListContainers,
//   )
//   defer d.Close()

//   b, err := provider.BuildCreate("httpd", "git", "http://example.org/build.tgz", structs.BuildOptions{
//     Cache: true,
//   })

//   assert.NoError(t, err)
//   assert.EqualValues(t, &structs.Build{
//     Id:      "B123",
//     App:     "httpd",
//     Status:  "created",
//     Started: time.Unix(1473028693, 0).UTC(),
//     Ended:   time.Unix(1473028892, 0).UTC(),
//     Tags:    map[string]string{},
//   }, b)
// }

// func TestBuildCreateWithCluster(t *testing.T) {
//   provider := StubAwsProvider(
//     cycleBuildDescribeStacks,
//     cycleBuildDescribeStacks,
//     cycleBuildPutItemCreate,
//     cycleBuildDescribeStackResources,
//     cycleBuildDescribeStacks,
//     cycleEnvironmentGetRack,
//     cycleRegistryListRegistries,
//     cycleRegistryGetRegistry,
//     cycleRegistryDecrypt,
//     cycleBuildDescribeStacks,
//     cycleBuildGetAuthorizationTokenPrivate1,
//     cycleBuildRunTaskCluster,
//     cycleBuildGetItem,
//     cycleBuildDescribeStacks,
//     cycleBuildPutItemCreate2,
//     cycleBuildDescribeTasks,
//     cycleBuildDescribeContainerInstances,
//     cycleBuildDescribeInstances,
//     cycleBuildDescribeStacks,
//     cycleBuildQuery150,
//   )
//   defer provider.Close()

//   d := stubDocker(
//     cycleBuildDockerListContainers,
//   )
//   defer d.Close()

//   provider.BuildCluster = "cluster-build"

//   b, err := provider.BuildCreate("httpd", "git", "http://example.org/build.tgz", structs.BuildOptions{
//     Cache: true,
//   })

//   assert.NoError(t, err)
//   assert.EqualValues(t, &structs.Build{
//     Id:      "B123",
//     App:     "httpd",
//     Status:  "created",
//     Started: time.Unix(1473028693, 0).UTC(),
//     Ended:   time.Unix(1473028892, 0).UTC(),
//     Tags:    map[string]string{},
//   }, b)
// }

func TestBuildDelete(t *testing.T) {
	provider := StubAwsProvider(
		cycleBuildGetItem,
		cycleBuildDescribeStacks,
		cycleReleaseGetItem,
		cycleReleaseDescribeStackResources,
		cycleReleaseEnvironmentGet,
		cycleSystemDescribeStackResources,
		cycleBuildDeleteItem,
		cycleBuildBatchDeleteImage,
	)
	defer provider.Close()

	b, err := provider.BuildDelete("httpd", "B123")

	assert.NoError(t, err)
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
		cycleBuildDescribeStacks,
		cycleBuildDescribeRepositories,
		cycleBuildGetAuthorizationToken,
	)
	defer provider.Close()

	d := stubDocker(
		cycleBuildDockerPing,
		cycleBuildDockerInfo,
		cycleBuildDockerLogin,
		cycleBuildDockerPing,
		cycleBuildDockerPull,
		cycleBuildDockerPing,
		cycleBuildDockerSave,
	)
	defer d.Close()

	buf := &bytes.Buffer{}

	err := provider.BuildExport("httpd", "B123", buf)
	assert.NoError(t, err)

	gz, err := gzip.NewReader(buf)
	assert.NoError(t, err)

	tr := tar.NewReader(gz)

	h, err := tr.Next()
	assert.NoError(t, err)
	assert.Equal(t, "build.json", h.Name)
	assert.Equal(t, int64(454), h.Size)

	data, err := ioutil.ReadAll(tr)
	assert.NoError(t, err)

	var build structs.Build
	err = json.Unmarshal(data, &build)
	assert.NoError(t, err)
	assert.Equal(t, "BAFVEWUCAYT", build.Id)
	assert.Equal(t, "httpd", build.App)
	assert.Equal(t, "RVWOJNKRAXU", build.Release)

	h, err = tr.Next()
	assert.NoError(t, err)
	assert.Equal(t, "httpd.BAFVEWUCAYT.tar", h.Name)
	assert.Equal(t, int64(13), h.Size)

	h, err = tr.Next()
	assert.Equal(t, io.EOF, err)
	assert.Nil(t, h)
}

// cycleSystemDescribeStackResources,
// func TestBuildImport(t *testing.T) {
//   provider := StubAwsProvider(
//     cycleBuildDescribeStacks,
//     cycleBuildDescribeRepositories,
//     cycleBuildGetAuthorizationToken,
//     cycleBuildGetNoItem,
//     cycleBuildDescribeStacks,
//     cycleReleaseDescribeStackResources,
//     cycleEnvironmentGet,
//     cycleSystemDescribeStackResources,
//     cycleBuildDescribeStacks,
//     cycleBuildPutItem,
//     cycleSystemDescribeStackResources,
//     // cycleBuildDescribeStacks,
//     // cycleReleaseDescribeStackResources,
//     // cycleEnvironmentPut,
//     // cycleBuildReleasePutItem,
//   )
//   defer provider.Close()

//   d := stubDocker(
//     cycleBuildDockerPing,
//     cycleBuildDockerInfo,
//     cycleBuildDockerLogin,
//     cycleBuildDockerPing,
//     cycleBuildDockerLoad,
//     cycleBuildDockerPing,
//     cycleBuildDockerTag,
//     cycleBuildDockerPing,
//     cycleBuildDockerPush,
//   )
//   defer d.Close()

//   build := &structs.Build{
//     Id:      "B12345",
//     App:     "httpd",
//     Release: "R23456",
//   }

//   data, err := json.Marshal(build)
//   require.NoError(t, err)

//   buf := &bytes.Buffer{}

//   gz := gzip.NewWriter(buf)
//   tw := tar.NewWriter(gz)

//   err = tw.WriteHeader(&tar.Header{
//     Typeflag: tar.TypeReg,
//     Name:     "build.json",
//     Size:     int64(len(data)),
//   })
//   require.NoError(t, err)

//   n, err := tw.Write(data)
//   require.NoError(t, err)
//   assert.Equal(t, 177, n)

//   lbuf := &bytes.Buffer{}

//   ltw := tar.NewWriter(lbuf)

//   data = []byte(`[{"RepoTags":["12345.dkr.ecr.us-east-1.amazonaws.com/convox-httpd-aaaaaaa:web.BRZMXKKHCMR"]}]`)

//   err = ltw.WriteHeader(&tar.Header{
//     Typeflag: tar.TypeReg,
//     Name:     "manifest.json",
//     Size:     int64(len(data)),
//   })
//   require.NoError(t, err)

//   n, err = ltw.Write(data)
//   require.NoError(t, err)
//   assert.Equal(t, 93, n)

//   err = ltw.Close()
//   require.NoError(t, err)

//   err = tw.WriteHeader(&tar.Header{
//     Typeflag: tar.TypeReg,
//     Name:     "httpd.B12345.tar",
//     Size:     int64(lbuf.Len()),
//   })
//   require.NoError(t, err)

//   n, err = tw.Write(lbuf.Bytes())
//   require.NoError(t, err)
//   assert.Equal(t, 2048, n)

//   err = tw.Close()
//   require.NoError(t, err)

//   err = gz.Close()
//   require.NoError(t, err)

//   build, err = provider.BuildImport("httpd", buf)
//   require.NoError(t, err)
//   assert.Equal(t, "B12345", build.Id)
//   assert.Equal(t, "httpd", build.App)
//   assert.Equal(t, "R23456", build.Release)
// }

// func TestBuildList(t *testing.T) {
//   provider := StubAwsProvider(
//     cycleBuildDescribeStacks,
//     cycleBuildQuery,
//   )
//   defer provider.Close()

//   b, err := provider.BuildList("httpd", 150)

//   assert.NoError(t, err)
//   assert.EqualValues(t, structs.Builds{
//     structs.Build{
//       Id:       "BHINCLZYYVN",
//       App:      "httpd",
//       Logs:     "",
//       Manifest: "web:\n  image: httpd\n  ports:\n  - 80:80\n",
//       Release:  "RVFETUHHKKD",
//       Status:   "complete",
//       Started:  time.Unix(1459780456, 178278576).UTC(),
//       Ended:    time.Unix(1459780542, 440881687).UTC(),
//       Tags:     map[string]string{},
//     },
//     structs.Build{
//       Id:       "BNOARQMVHUO",
//       App:      "httpd",
//       Logs:     "",
//       Manifest: "web:\n  image: httpd\n  ports:\n  - 80:80\n",
//       Release:  "RFVZFLKVTYO",
//       Status:   "complete",
//       Started:  time.Unix(1459780456, 178278576).UTC(),
//       Ended:    time.Unix(1459780542, 440881687).UTC(),
//       Tags:     map[string]string{},
//     },
//   }, b)
// }

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

	r, err := provider.BuildLogs("httpd", "B123", structs.LogsOptions{})

	io.Copy(buf, r)

	assert.NoError(t, err)
	assert.Equal(t, "RUNNING: docker pull httpd", buf.String())
}

func TestBuildLogsNotRunning(t *testing.T) {
	provider := StubAwsProvider(
		cycleBuildGetItem,
		cycleObjectDescribeStackResources,
		cycleBuildFetchLogs,
	)
	defer provider.Close()

	buf := &bytes.Buffer{}

	r, err := provider.BuildLogs("httpd", "B123", structs.LogsOptions{})

	io.Copy(buf, r)

	assert.NoError(t, err)
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

var cyclePing = awsutil.Cycle{
	awsutil.Request{
		Method:     "GET",
		RequestURI: "/_ping",
		Operation:  "",
		Body:       "",
	},
	awsutil.Response{
		StatusCode: 200,
		Body:       `OK`,
	},
}

var cycleBuildDescribeStacks = awsutil.Cycle{
	awsutil.Request{"POST", "/", "", `Action=DescribeStacks&StackName=convox-httpd&Version=2010-05-15`},
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

var cycleBuildDescribeStackResources = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "",
		Body:       `Action=DescribeStackResources&StackName=convox&Version=2010-05-15`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<DescribeStackResourcesResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/">
				<DescribeStackResourcesResult>
					<StackResources>
						<member>
							<PhysicalResourceId>build-task-arn</PhysicalResourceId>
							<LogicalResourceId>ApiBuildTasks</LogicalResourceId>
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
		Body:       `{}`,
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

var cycleBuildGetAuthorizationTokenPrivate1 = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerRegistry_V20150921.GetAuthorizationToken",
		Body: `{
			"registryIds": [
				"132866487567"
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
					"proxyEndpoint": "https://132866487567.dkr.ecr.us-east-1.amazonaws.com"
				}
			]
		}`,
	},
}

var cycleBuildGetAuthorizationTokenPrivate2 = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerRegistry_V20150921.GetAuthorizationToken",
		Body: `{
			"registryIds": [
				"132866487567"
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

var cycleBuildInfo = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/1.24/info",
		Method:     "GET",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{}`,
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

var cycleBuildGetNoItem = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.GetItem",
		Body: `{
			"ConsistentRead": true,
			"Key": {
				"id": {
					"S": "B12345"
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
		Method:     "GET",
		RequestURI: "/convox-httpd-settings-139bidzalmbtu/test/foo",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       "RUNNING: docker pull httpd",
	},
}

var cycleBuildNotificationPublish = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Body:       `Action=Publish&Message=%7B%22action%22%3A%22build%3Acreate%22%2C%22status%22%3A%22success%22%2C%22data%22%3A%7B%22app%22%3A%22httpd%22%2C%22id%22%3A%22B123%22%7D%2C%22timestamp%22%3A%220001-01-01T00%3A00%3A00Z%22%7D&Subject=build%3Acreate&TargetArn=&Version=2010-03-31`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body: `
			<PublishResponse xmlns="http://sns.amazonaws.com/doc/2010-03-31/">
				<PublishResult>
					<MessageId>94f20ce6-13c5-43a0-9a9e-ca52d816e90b</MessageId>
				</PublishResult>
				<ResponseMetadata>
					<RequestId>f187a3c1-376f-11df-8963-01868b7c937a</RequestId>
				</ResponseMetadata>
			</PublishResponse>
		`,
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

var cycleBuildPutItemCreate = awsutil.Cycle{
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
				"ended": {
					"S": "20160904.224132.000000000"
				},
				"id": {
					"S": "B123"
				},
				"status": {
					"S": "created"
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

var cycleBuildPutItemCreate2 = awsutil.Cycle{
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
				"ended": {
					"S": "20160904.224132.000000000"
				},
				"id": {
					"S": "BAFVEWUCAYT"
				},
				"logs": {
					"S": "object:///test/foo"
				},
				"manifest": {
					"S": "version: \"2\"\nnetworks: {}\nservices:\n  web:\n    build: {}\n    command: null\n    image: httpd\n    ports:\n    - 80:80\n"
				},
				"release": {
					"S": "RVWOJNKRAXU"
				},
				"status": {
					"S": "running"
				},
				"tags": {
					"B": "eyJ0YXNrIjoiYXJuOmF3czplY3M6dXMtZWFzdC0xOjc3ODc0MzUyNzUzMjp0YXNrLzUwYjhkZTk5LWY5NGYtNGVjZC1hOThmLTU4NTA3NjBmMDg0NSJ9"
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
		Body:       `{"IndexName":"app.created","KeyConditions":{"app":{"AttributeValueList":[{"S":"httpd"}],"ComparisonOperator":"EQ"}},"Limit":150,"ScanIndexForward":false,"TableName":"convox-builds"}`,
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

var cycleBuildQuery150 = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.Query",
		Body:       `{"IndexName":"app.created","KeyConditions":{"app":{"AttributeValueList":[{"S":"httpd"}],"ComparisonOperator":"EQ"}},"Limit":150,"ScanIndexForward":false,"TableName":"convox-builds"}`,
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

var cycleBuildRunTask = awsutil.Cycle{
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
							"build",
							"-method",
							"git",
							"-cache",
							"true"
						],
						"environment": [
							{
								"name": "BUILD_APP",
								"value": "httpd"
							},
							{
								"name": "BUILD_AUTH",
								"value": "{\"132866487567.dkr.ecr.us-test-1.amazonaws.com\":{\"Username\":\"user\",\"Password\":\"12345\\n\"},\"quay.io\":{\"Username\":\"ddollar+test\",\"Password\":\"B0IT2U7BZ4VDZUYFM6LFMTJPF8YGKWYBR39AWWPAUKZX6YKZX3SQNBCCQKMX08UF\"}}"
							},
							{
								"name": "BUILD_CONFIG",
								"value": ""
							},
							{
								"name": "BUILD_ID",
								"value": "B123"
							},
							{
								"name": "BUILD_PUSH",
								"value": "132866487567.dkr.ecr.us-test-1.amazonaws.com/convox-httpd-hqvvfosgxt:{service}.{build}"
							},
							{
								"name": "BUILD_URL",
								"value": "http://example.org/build.tgz"
							},
							{
								"name": "HTTP_PROXY",
								"value": ""
							},
							{
								"name": "RELEASE",
								"value": "B123"
							}
						],
						"name": "build"
      }
    ]
  },
  "startedBy": "convox.httpd",
  "taskDefinition": "build-task-arn"
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
							"taskArn": "arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845"
						}
					],
					"containerInstanceArn": "arn:aws:ecs:us-east-1:778743527532:container-instance/e126c67d-fa95-4b09-8b4a-3723932cd2aa",
					"taskArn": "arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845"
				}
			]
		}`,
	},
}

var cycleBuildRunTaskCluster = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "AmazonEC2ContainerServiceV20141113.RunTask",
		Body: `{
			"cluster": "cluster-build",
			"count": 1,
			"overrides": {
				"containerOverrides": [
					{
						"command": [
							"build",
							"-method",
							"git",
							"-cache",
							"true"
						],
						"environment": [
							{
								"name": "BUILD_APP",
								"value": "httpd"
							},
							{
								"name": "BUILD_AUTH",
								"value": "{\"132866487567.dkr.ecr.us-test-1.amazonaws.com\":{\"Username\":\"user\",\"Password\":\"12345\\n\"},\"quay.io\":{\"Username\":\"ddollar+test\",\"Password\":\"B0IT2U7BZ4VDZUYFM6LFMTJPF8YGKWYBR39AWWPAUKZX6YKZX3SQNBCCQKMX08UF\"}}"
							},
							{
								"name": "BUILD_CONFIG",
								"value": ""
							},
							{
								"name": "BUILD_ID",
								"value": "B123"
							},
							{
								"name": "BUILD_PUSH",
								"value": "132866487567.dkr.ecr.us-test-1.amazonaws.com/convox-httpd-hqvvfosgxt:{service}.{build}"
							},
							{
								"name": "BUILD_URL",
								"value": "http://example.org/build.tgz"
							},
							{
								"name": "HTTP_PROXY",
								"value": ""
							},
							{
								"name": "RELEASE",
								"value": "B123"
							}
						],
						"name": "build"
      }
    ]
  },
  "startedBy": "convox.httpd",
  "taskDefinition": "build-task-arn"
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
							"taskArn": "arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845"
						}
					],
					"containerInstanceArn": "arn:aws:ecs:us-east-1:778743527532:container-instance/e126c67d-fa95-4b09-8b4a-3723932cd2aa",
					"taskArn": "arn:aws:ecs:us-east-1:778743527532:task/50b8de99-f94f-4ecd-a98f-5850760f0845"
				}
			]
		}`,
	},
}

var cycleBuildDockerListContainers = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "GET",
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
		Method:     "GET",
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

var cycleBuildDockerInfo = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "GET",
		RequestURI: "/v1.24/info",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `OK`, //FIXME
	},
}
var cycleBuildDockerPing = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "GET",
		RequestURI: "/_ping",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `OK`,
	},
}

var cycleBuildDockerLogin = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/v1.24/auth",
		Body: `{
			"password": "12345\n",
			"serveraddress": "778743527532.dkr.ecr.us-east-1.amazonaws.com",
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
		Method:     "GET",
		RequestURI: "/v1.24/images/get?names=778743527532.dkr.ecr.us-east-1.amazonaws.com%2Fconvox-rails-sslibosttb%3Aweb.BAFVEWUCAYT",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `should-be-tar`,
	},
}

var cycleBuildDockerTag = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/v1.24/images/12345.dkr.ecr.us-east-1.amazonaws.com/convox-httpd-aaaaaaa:web.BRZMXKKHCMR/tag?repo=778743527532.dkr.ecr.us-east-1.amazonaws.com%2Fconvox-rails-sslibosttb&tag=web.B12345",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{}`,
	},
}

var cycleEnvironmentGet = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "GET",
		RequestURI: "/convox-httpd-settings-139bidzalmbtu/env",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       "FOO=bar\nBAZ=qux",
	},
}

var cycleEnvironmentGetRack = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "GET",
		RequestURI: "/convox-settings/env",
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{}`,
	},
}

var cycleEnvironmentPut = awsutil.Cycle{
	Request: awsutil.Request{
		Method:     "PUT",
		RequestURI: "/convox-httpd-settings-139bidzalmbtu/releases/R23456/env",
		Body:       "BAZ=qux\nFOO=bar",
	},
	Response: awsutil.Response{
		StatusCode: 200,
	},
}
