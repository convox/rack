package test

import (
	"net/http/httptest"
	"os"

	"github.com/convox/rack/api/awsutil"
)

/*
Create a test server that mocks an Docker request/response cycle,
suitable for a single test

Example:
		s := StubDocker(ListContainersCycle())
		defer s.Close()

		d, _ := Docker(test.TestConfig.DockerHost)
		d.ListContainers(...)
*/
func StubDocker(cycles ...awsutil.Cycle) (s *httptest.Server) {
	handler := awsutil.NewHandler(cycles)
	s = httptest.NewServer(handler)

	os.Setenv("TEST_DOCKER_HOST", s.URL)

	return s
}

func ListECSContainersCycle() awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/containers/json?filters=%7B%22label%22%3A%5B%22com.amazonaws.ecs.task-arn%3Darn%3Aaws%3Aecs%3Aus-east-1%3A901416387788%3Atask%2F320a8b6a-c243-47d3-a1d1-6db5dfcb3f58%22%2C%22com.amazonaws.ecs.container-name%3Dworker%22%5D%7D",
			Operation:  "",
			Body:       ``,
		},
		Response: awsutil.Response{
			StatusCode: 200,
			Body:       `[{"Id": "8dfafdbc3a40","Command": "echo 1"}]`,
		},
	}
}

func ListECSOneoffContainersCycle() awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/containers/json?filters=%7B%22label%22%3A%5B%22com.amazonaws.ecs.task-arn%3Darn%3Aaws%3Aecs%3Aus-east-1%3A901416387788%3Atask%2Fdbf4506f-6e57-44d5-9cfe-bc6ea10dbacc%22%2C%22com.amazonaws.ecs.container-name%3Dworker%22%5D%7D",
			Operation:  "",
			Body:       ``,
		},
		Response: awsutil.Response{
			StatusCode: 200,
			Body:       `[{"Id": "ae6fc7edad70","Command": "/bin/sh -c yes"}]`,
		},
	}
}

func ListOneoffContainersCycle(id string) awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/containers/json?filters=%7B%22label%22%3A%5B%22com.convox.rack.type%3Doneoff%22%2C%22com.convox.rack.app%3Dmyapp-staging%22%5D%7D",
			Operation:  "",
			Body:       ``,
		},
		Response: awsutil.Response{
			StatusCode: 200,
			Body:       `[{"Id": "` + id + `","Command": "/bin/sh -c bash"}]`,
		},
	}
}

func ListOneoffContainersEmptyCycle() awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/containers/json?filters=%7B%22label%22%3A%5B%22com.convox.rack.type%3Doneoff%22%2C%22com.convox.rack.app%3Dmyapp-staging%22%5D%7D",
			Operation:  "",
			Body:       ``,
		},
		Response: awsutil.Response{
			StatusCode: 200,
			Body:       `[]`,
		},
	}
}

func InspectCycle(id string) awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/containers/" + id + "/json",
			Operation:  "",
			Body:       ``,
		},
		Response: awsutil.Response{
			StatusCode: 200,
			Body:       inspectResponse(id),
		},
	}
}

func StatsCycle() awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/containers/8dfafdbc3a40/stats?stream=false",
			Operation:  "",
			Body:       ``,
		},
		Response: awsutil.Response{
			StatusCode: 200,
			Body:       statsResponse(),
		},
	}
}

func inspectResponse(id string) string {
	return `{
    "AppArmorProfile": "",
    "Args": [
        "-c",
        "exit 9"
    ],
    "Config": {
        "AttachStderr": true,
        "AttachStdin": false,
        "AttachStdout": true,
        "Cmd": [
            "/bin/sh",
            "-c",
            "bash"
        ],
        "Domainname": "",
        "Entrypoint": null,
        "Env": [
            "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
        ],
        "ExposedPorts": null,
        "Hostname": "ba033ac44011",
        "Image": "ubuntu",
        "Labels": {
            "com.example.vendor": "Acme",
            "com.example.license": "GPL",
            "com.example.version": "1.0"
        },
        "MacAddress": "",
        "NetworkDisabled": false,
        "OnBuild": null,
        "OpenStdin": false,
        "StdinOnce": false,
        "Tty": false,
        "User": "",
        "Volumes": null,
        "WorkingDir": "",
        "StopSignal": "SIGTERM"
    },
    "Created": "2015-01-06T15:47:31.485331387Z",
    "Driver": "devicemapper",
    "ExecDriver": "native-0.2",
    "ExecIDs": null,
    "HostConfig": {
        "Binds": null,
        "BlkioWeight": 0,
        "BlkioWeightDevice": [{}],
        "BlkioDeviceReadBps": [{}],
        "BlkioDeviceWriteBps": [{}],
        "BlkioDeviceReadIOps": [{}],
        "BlkioDeviceWriteIOps": [{}],
        "CapAdd": null,
        "CapDrop": null,
        "ContainerIDFile": "",
        "CpusetCpus": "",
        "CpusetMems": "",
        "CpuShares": 0,
        "CpuPeriod": 100000,
        "Devices": [],
        "Dns": null,
        "DnsOptions": null,
        "DnsSearch": null,
        "ExtraHosts": null,
        "IpcMode": "",
        "Links": null,
        "LxcConf": [],
        "Memory": 0,
        "MemorySwap": 0,
        "MemoryReservation": 0,
        "KernelMemory": 0,
        "OomKillDisable": false,
        "OomScoreAdj": 500,
        "NetworkMode": "bridge",
        "PortBindings": {},
        "Privileged": false,
        "ReadonlyRootfs": false,
        "PublishAllPorts": false,
        "RestartPolicy": {
            "MaximumRetryCount": 2,
            "Name": "on-failure"
        },
        "LogConfig": {
            "Config": null,
            "Type": "json-file"
        },
        "SecurityOpt": null,
        "VolumesFrom": null,
        "Ulimits": [{}],
        "VolumeDriver": "",
        "ShmSize": 67108864
    },
    "HostnamePath": "/var/lib/docker/containers/` + id + `/hostname",
    "HostsPath": "/var/lib/docker/containers/` + id + `/hosts",
    "LogPath": "/var/lib/docker/containers/1eb5fabf5a03807136561b3c00adcd2992b535d624d5e18b6cdc6a6844d9767b/1eb5fabf5a03807136561b3c00adcd2992b535d624d5e18b6cdc6a6844d9767b-json.log",
    "Id": "` + id + `",
    "Image": "04c5d3b7b0656168630d3ba35d8889bd0e9caafcaeb3004d2bfbc47e7c5d35d2",
    "MountLabel": "",
    "Name": "/boring_euclid",
    "NetworkSettings": {
        "Bridge": "",
        "SandboxID": "",
        "HairpinMode": false,
        "LinkLocalIPv6Address": "",
        "LinkLocalIPv6PrefixLen": 0,
        "Ports": null,
        "SandboxKey": "",
        "SecondaryIPAddresses": null,
        "SecondaryIPv6Addresses": null,
        "EndpointID": "",
        "Gateway": "",
        "GlobalIPv6Address": "",
        "GlobalIPv6PrefixLen": 0,
        "IPAddress": "",
        "IPPrefixLen": 0,
        "IPv6Gateway": "",
        "MacAddress": "",
        "Networks": {
            "bridge": {
                "NetworkID": "7ea29fc1412292a2d7bba362f9253545fecdfa8ce9a6e37dd10ba8bee7129812",
                "EndpointID": "7587b82f0dada3656fda26588aee72630c6fab1536d36e394b2bfbcf898c971d",
                "Gateway": "172.17.0.1",
                "IPAddress": "172.17.0.2",
                "IPPrefixLen": 16,
                "IPv6Gateway": "",
                "GlobalIPv6Address": "",
                "GlobalIPv6PrefixLen": 0,
                "MacAddress": "02:42:ac:12:00:02"
            }
        }
    },
    "Path": "/bin/sh",
    "ProcessLabel": "",
    "ResolvConfPath": "/var/lib/docker/containers/` + id + `/resolv.conf",
    "RestartCount": 1,
    "State": {
        "Error": "",
        "ExitCode": 9,
        "FinishedAt": "2015-01-06T15:47:32.080254511Z",
        "OOMKilled": false,
        "Dead": false,
        "Paused": false,
        "Pid": 0,
        "Restarting": false,
        "Running": true,
        "StartedAt": "2015-01-06T15:47:32.072697474Z",
        "Status": "running"
    },
    "Mounts": [
        {
            "Name": "fac362...80535",
            "Source": "/data",
            "Destination": "/data",
            "Driver": "local",
            "Mode": "ro,Z",
            "RW": false,
            "Propagation": ""
        }
    ]
}`
}

func statsResponse() string {
	return `{
 "read" : "2015-01-08T22:57:31.547920715Z",
 "network" : {
    "rx_dropped" : 0,
    "rx_bytes" : 648,
    "rx_errors" : 0,
    "tx_packets" : 8,
    "tx_dropped" : 0,
    "rx_packets" : 8,
    "tx_errors" : 0,
    "tx_bytes" : 648
 },
 "memory_stats" : {
    "stats" : {
       "total_pgmajfault" : 0,
       "cache" : 0,
       "mapped_file" : 0,
       "total_inactive_file" : 0,
       "pgpgout" : 414,
       "rss" : 6537216,
       "total_mapped_file" : 0,
       "writeback" : 0,
       "unevictable" : 0,
       "pgpgin" : 477,
       "total_unevictable" : 0,
       "pgmajfault" : 0,
       "total_rss" : 6537216,
       "total_rss_huge" : 6291456,
       "total_writeback" : 0,
       "total_inactive_anon" : 0,
       "rss_huge" : 6291456,
       "hierarchical_memory_limit" : 67108864,
       "total_pgfault" : 964,
       "total_active_file" : 0,
       "active_anon" : 6537216,
       "total_active_anon" : 6537216,
       "total_pgpgout" : 414,
       "total_cache" : 0,
       "inactive_anon" : 0,
       "active_file" : 0,
       "pgfault" : 964,
       "inactive_file" : 0,
       "total_pgpgin" : 477
    },
    "max_usage" : 6651904,
    "usage" : 6537216,
    "failcnt" : 0,
    "limit" : 67108864
 },
 "blkio_stats" : {},
 "cpu_stats" : {
    "cpu_usage" : {
       "percpu_usage" : [
          16970827,
          1839451,
          7107380,
          10571290
       ],
       "usage_in_usermode" : 10000000,
       "total_usage" : 36488948,
       "usage_in_kernelmode" : 20000000
    },
    "system_cpu_usage" : 20091722000000000,
    "throttling_data" : {}
 }
}`
}
