# convox/agent

Instance agent for collecting logs and metrics.

## Development

Copy select outputs from a dev rack into .env:

```bash
$ cat .env
AWS_REGION=us-west-2
AWS_ACCESS_KEY_ID=XXX
AWS_SECRET_ACCESS_KEY=YYY
ECS_CLUSTER=convox-Cluster-1NCWX9EC0JOV4
```

This approximates the IAM role and env the agent has on a cluster instance.

Start a dev agent:

```bash
$ convox start
agent | monitor new region=us-west-2 cluster=convox-Cluster-1NCWX9EC0JOV4
agent | disk monitor hostname=4f7ca7a9232a
agent | disk monitor hostname=4f7ca7a9232a utilization=16.02% used=0.1209G available=0.7547G
agent | disk monitor upload to=kinesis stream="convox-Kinesis-2NQ3Q5ASHY1N" lines=1
```

Run a Docker container to see Docker event Kinesis upload activity:

```bash
$ docker run -e KINESIS=convox-Kinesis-2NQ3Q5ASHY1N -e PROCESS=hello-world hello-world
```

```
agent | monitor event id=a5018a56adc3 status=create
agent | monitor event id=a5018a56adc3 status=attach
agent | monitor event id=a5018a56adc3 status=start
agent | monitor event id=a5018a56adc3 status=die
agent | monitor upload to=kinesis stream="convox-Kinesis-2NQ3Q5ASHY1N" lines=21
```

Run a Docker container to see cgroup fun:

```bash
$ docker run -e SWAP=1 redis
```

```
agent | monitor event id=6176a834e31a status=create
agent | monitor event id=6176a834e31a status=attach
agent | monitor event id=6176a834e31a status=start
agent | monitor cgroups id=6176a834e31a cgroup=memory.memsw.limit_in_bytes value=18446744073709551615
agent | error: open /cgroup/memory/docker/6176a834e31ac355bcc18dc83a113c64bd00ada284dd9e61153ed18715438365/memory.memsw.limit_in_bytes: no such file or directory
agent | monitor cgroups id=6176a834e31a cgroup=memory.soft_limit_in_bytes value=18446744073709551615
agent | error: open /cgroup/memory/docker/6176a834e31ac355bcc18dc83a113c64bd00ada284dd9e61153ed18715438365/memory.soft_limit_in_bytes: no such file or directory
agent | monitor cgroups id=6176a834e31a cgroup=memory.limit_in_bytes value=18446744073709551615
agent | error: open /cgroup/memory/docker/6176a834e31ac355bcc18dc83a113c64bd00ada284dd9e61153ed18715438365/memory.limit_in_bytes: no such file or directory
```

## Release

convox/agent is released as a public Docker image on Docker Hub, and public
config file on S3.

```bash
$ make release
```

## Production

convox/agent runs on every ECS cluster instance. This is configured by the
convox/rack CloudFormation template UserData and an upstart script.

Pseudocode:

```bash
$ mkdir -p /etc/convox
$ echo us-east-1 > /etc/convox/region
$ curl -s http://convox.s3.amazonaws.com/agent/0.3/convox.conf > /etc/init/convox.conf
$ start convox

# Spawns
$ docker run -a STDOUT -a STDERR --sig-proxy -e AWS_REGION=$(cat /etc/convox/region) -v /cgroup:/cgroup -v /var/run/docker.sock:/var/run/docker.sock convox/agent:0.3
```

The running agent can

* Watch Docker events via the host Docker socket
* Modify contaier settings via the host /cgroup control groups
* Put events (logs) to Kinesis streams via the InstanceProfile
* Put metric data to CloudWatch via the InstanceProfile

## License

Apache 2.0 &copy; 2015 Convox, Inc.
