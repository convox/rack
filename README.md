# convox/agent

Instance agent for collecting logs and metrics.

## Development

```bash
$ convox start
agent | agent region=us-west-2 cluster=convox-Cluster-1NCWX9EC0JOV4
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
