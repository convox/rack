package aws_test

import (
	"os"
	"testing"
	"time"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/api/structs"
	"github.com/stretchr/testify/assert"
)

func TestInstancesList(t *testing.T) {
	os.Setenv("CLUSTER", "convox-test-cluster")

	provider := StubAwsProvider(
		cycleInstanceDescribeInstances,
		listContainerInstancesCycle("cluster-test"),
		describeContainerInstancesCycle("cluster-test"),
		// TODO: GetMetricStatistics x 3
	)
	defer provider.Close()

	is, err := provider.InstanceList()

	assert.NoError(t, err)
	assert.EqualValues(t, structs.Instances{
		structs.Instance{
			Agent:     true,
			Cpu:       0,
			Id:        "i-3963798e",
			Memory:    0,
			PrivateIp: "10.0.2.236",
			Processes: 0,
			PublicIp:  "54.85.115.31",
			Status:    "active",
			Started:   time.Unix(1448386549, 0).UTC(),
		},
		structs.Instance{
			Agent:     true,
			Cpu:       0,
			Id:        "i-4a5513f4",
			Memory:    0,
			PrivateIp: "10.0.1.182",
			Processes: 0,
			PublicIp:  "54.208.61.75",
			Status:    "active",
			Started:   time.Unix(1448484072, 0).UTC(),
		},
		structs.Instance{
			Agent:     true,
			Cpu:       0,
			Id:        "i-c6a72b76",
			Memory:    0.19161676646706588,
			PrivateIp: "10.0.3.248",
			Processes: 1,
			PublicIp:  "52.71.252.224",
			Status:    "active",
			Started:   time.Unix(1447901993, 0).UTC(),
		},
	}, is)
}

func listContainerInstancesCycle(clusterName string) awsutil.Cycle {
	return awsutil.Cycle{
		awsutil.Request{"POST", "/", "AmazonEC2ContainerServiceV20141113.ListContainerInstances",
			`{"cluster":"` + clusterName + `","nextToken": ""}`},
		awsutil.Response{200,
			`{"containerInstanceArns":["arn:aws:ecs:us-east-1:901416387788:container-instance/0ac4bb1c-be98-4202-a9c1-03153e91c05e","arn:aws:ecs:us-east-1:901416387788:container-instance/38a59629-6f5d-4d02-8733-fdb49500ae45","arn:aws:ecs:us-east-1:901416387788:container-instance/e7c311ae-968f-4125-8886-f9b724860d4c"]}`},
	}
}

func describeContainerInstancesCycle(clusterName string) awsutil.Cycle {
	return awsutil.Cycle{
		awsutil.Request{"POST", "/", "AmazonEC2ContainerServiceV20141113.DescribeContainerInstances",
			`{"cluster":"` + clusterName + `",
        "containerInstances": [
          "arn:aws:ecs:us-east-1:901416387788:container-instance/0ac4bb1c-be98-4202-a9c1-03153e91c05e",
          "arn:aws:ecs:us-east-1:901416387788:container-instance/38a59629-6f5d-4d02-8733-fdb49500ae45",
          "arn:aws:ecs:us-east-1:901416387788:container-instance/e7c311ae-968f-4125-8886-f9b724860d4c"
        ]
    }`},
		awsutil.Response{200, describeContainerInstancesResponse()},
	}
}

func describeContainerInstancesResponse() string {
	return `{"containerInstances":[{"agentConnected":true,"containerInstanceArn":"arn:aws:ecs:us-east-1:901416387788:container-instance/0ac4bb1c-be98-4202-a9c1-03153e91c05e","ec2InstanceId":"i-4a5513f4","pendingTasksCount":0,"registeredResources":[{"doubleValue":0.0,"integerValue":1024,"longValue":0,"name":"CPU","type":"INTEGER"},{"doubleValue":0.0,"integerValue":2004,"longValue":0,"name":"MEMORY","type":"INTEGER"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS","stringSetValue":["22","2376","2375","51678"],"type":"STRINGSET"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS_UDP","stringSetValue":[],"type":"STRINGSET"}],"remainingResources":[{"doubleValue":0.0,"integerValue":1024,"longValue":0,"name":"CPU","type":"INTEGER"},{"doubleValue":0.0,"integerValue":2004,"longValue":0,"name":"MEMORY","type":"INTEGER"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS","stringSetValue":["22","2376","2375","51678"],"type":"STRINGSET"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS_UDP","stringSetValue":[],"type":"STRINGSET"}],"runningTasksCount":0,"status":"ACTIVE","versionInfo":{"agentHash":"4ab1051","agentVersion":"1.4.0","dockerVersion":"DockerVersion: 1.7.1"}},
{"agentConnected":true,"containerInstanceArn":"arn:aws:ecs:us-east-1:901416387788:container-instance/38a59629-6f5d-4d02-8733-fdb49500ae45","ec2InstanceId":"i-3963798e","pendingTasksCount":0,"registeredResources":[{"doubleValue":0.0,"integerValue":1024,"longValue":0,"name":"CPU","type":"INTEGER"},{"doubleValue":0.0,"integerValue":2004,"longValue":0,"name":"MEMORY","type":"INTEGER"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS","stringSetValue":["22","2376","2375","51678"],"type":"STRINGSET"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS_UDP","stringSetValue":[],"type":"STRINGSET"}],"remainingResources":[{"doubleValue":0.0,"integerValue":1024,"longValue":0,"name":"CPU","type":"INTEGER"},{"doubleValue":0.0,"integerValue":2004,"longValue":0,"name":"MEMORY","type":"INTEGER"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS","stringSetValue":["22","2376","2375","51678"],"type":"STRINGSET"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS_UDP","stringSetValue":[],"type":"STRINGSET"}],"runningTasksCount":0,"status":"ACTIVE","versionInfo":{"agentHash":"4ab1051","agentVersion":"1.4.0","dockerVersion":"DockerVersion: 1.7.1"}},
{"agentConnected":true,"containerInstanceArn":"arn:aws:ecs:us-east-1:901416387788:container-instance/e7c311ae-968f-4125-8886-f9b724860d4c","ec2InstanceId":"i-c6a72b76","pendingTasksCount":0,"registeredResources":[{"doubleValue":0.0,"integerValue":1024,"longValue":0,"name":"CPU","type":"INTEGER"},{"doubleValue":0.0,"integerValue":2004,"longValue":0,"name":"MEMORY","type":"INTEGER"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS","stringSetValue":["22","2376","2375","51678"],"type":"STRINGSET"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS_UDP","stringSetValue":[],"type":"STRINGSET"}],"remainingResources":[{"doubleValue":0.0,"integerValue":1024,"longValue":0,"name":"CPU","type":"INTEGER"},{"doubleValue":0.0,"integerValue":1620,"longValue":0,"name":"MEMORY","type":"INTEGER"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS","stringSetValue":["22","2376","2375","3101","3001","3100","51678","3000"],"type":"STRINGSET"},{"doubleValue":0.0,"integerValue":0,"longValue":0,"name":"PORTS_UDP","stringSetValue":[],"type":"STRINGSET"}],"runningTasksCount":1,"status":"ACTIVE","versionInfo":{"agentHash":"4ab1051","agentVersion":"1.4.0","dockerVersion":"DockerVersion: 1.7.1"}}],"failures":[]}`
}

var cycleInstanceDescribeInstances = awsutil.Cycle{
	awsutil.Request{"POST", "/", "", `Action=DescribeInstances&Filter.1.Name=tag%3ARack&Filter.1.Value.1=convox&Filter.2.Name=tag%3Aaws%3Acloudformation%3Alogical-id&Filter.2.Value.1=Instances&Version=2016-11-15`},
	awsutil.Response{200, describeInstancesResponse()},
}

func describeInstancesResponse() string {
	return `<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2015-10-01/">
    <requestId>f215b40f-5a0c-4fe6-9624-657cd1f4ef6b</requestId>
    <reservationSet>
        <item>
            <reservationId>r-8d7e2072</reservationId>
            <ownerId>901416387788</ownerId>
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
                            <value>arn:aws:cloudformation:us-east-1:901416387788:stack/convox-dev/538ae350-8815-11e5-8a2d-5001b34fc89a</value>
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
                            <ownerId>901416387788</ownerId>
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
                        <arn>arn:aws:iam::901416387788:instance-profile/convox-dev-InstanceProfile-HJBF2SIK0R6W</arn>
                        <id>AIPAIR7O43WTX246KVAIM</id>
                    </iamInstanceProfile>
                    <ebsOptimized>false</ebsOptimized>
                </item>
            </instancesSet>
            <requesterId>226008221399</requesterId>
        </item>
        <item>
            <reservationId>r-835b392a</reservationId>
            <ownerId>901416387788</ownerId>
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
                            <value>arn:aws:cloudformation:us-east-1:901416387788:stack/convox-dev/538ae350-8815-11e5-8a2d-5001b34fc89a</value>
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
                            <ownerId>901416387788</ownerId>
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
                        <arn>arn:aws:iam::901416387788:instance-profile/convox-dev-InstanceProfile-HJBF2SIK0R6W</arn>
                        <id>AIPAIR7O43WTX246KVAIM</id>
                    </iamInstanceProfile>
                    <ebsOptimized>false</ebsOptimized>
                </item>
            </instancesSet>
            <requesterId>226008221399</requesterId>
        </item>
        <item>
            <reservationId>r-da7b7c0a</reservationId>
            <ownerId>901416387788</ownerId>
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
                            <value>arn:aws:cloudformation:us-east-1:901416387788:stack/convox-dev/538ae350-8815-11e5-8a2d-5001b34fc89a</value>
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
                            <ownerId>901416387788</ownerId>
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
                        <arn>arn:aws:iam::901416387788:instance-profile/convox-dev-InstanceProfile-HJBF2SIK0R6W</arn>
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
