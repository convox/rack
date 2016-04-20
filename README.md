# Convox Rack

[![Build Status](https://travis-ci.org/convox/rack.svg?branch=master)](https://travis-ci.org/convox/rack)

Convox Rack is open source PaaS built on top of expert infrastructure automation and devops best practices.

Rack gives you a simple developer-focused API that lets you build, deploy, scale and manage apps on private infrastructure with ease.

### Private and Secure

Rack runs in an isolated VPC that only you and your team have access to. Application builds take place in a single-tenant build service, and the resulting Docker images are stored in a private ECS Container Registry. Application secrets are stored in S3, encrypted with KMS, a hardware security module. Application logs are archived in CloudWatch LogsGroups.

Your network is isolated, your platform is single-tenant, and your application data never leaves your AWS account.

### Simple and Reliable

Apps run as Docker containers on ECS with HTTP access through ELBs. This architecture is modern, simple and provably reliable and scalable.

Container Logs are extracted from Docker with its native APIs and log drivers. Docker daemon options are minimally changed to avoid observed log rotation problems.

Logs are stored in CloudWatch Logs for archival and search, and Kinesis for streaming. Lambda subscribers extract metrics and forward to 3rd party systems. This is simple and cost effective for any volume of logs.

Complex and experimental things like overlay networking, persistent container volumes, and distributed file systems are simply not supported at the moment.

All throughout the stack we aim to leverage managed services and mature systems to accomplish tasks at hand. AWS offers the vast majority of infrastructure services and Docker the vast majority of runtime functionality.

### Easy to Maintain

Platform updates are automatically applied with the `convox rack update` command.

Updates are executed with CloudFormation, so you can be confident that they will be safely executed.

Some updates are simple Rack API changes that will roll out in seconds.

Some updates are base security updates like a new AMI, Linux Kernel, or Docker engine. These are rolled out one instance at a time and are guaranteed to not cause application downtime.

Some updates are infrastructure migrations. For example, ECR is still in limited availability. When it does become available, a future `convox rack update` will safely migrate your clouds over to it.

### Open Source

Rack is open source and free (as in beer and in speech) to use. You can look at the source code to audit how it configures your AWS account. You you fork it and modify. You can contribute your ideas and patches back to the project so we can all share.

### Philosophy

The Convox team and the Rack project have a strong philosophy about how to manage cloud services. Some choices we frequently consider:

* Open over Closed
* [Integration over Invention](https://convox.com/blog/integration-over-invention/)
* Services over Software
* Robots over Humans
* Shared Expertise vs Bespoke
* Porcelain over Plumbing

## Installation Quick Start

You need an AWS account and access credentials, the Convox CLI, and 10 minutes.

```
# Create and pass AWS access keys to the installer. These should be temporary keys and can be deleted after the install.

$ export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
$ export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

# Download and install the convox CLI for your platform

$ curl -Ls https://install.convox.com/osx.zip > /tmp/convox.zip
$ unzip /tmp/convox.zip -d /usr/local/bin

# Install Rack. You may be interested in options like `--region us-west-2`

$ convox install
...

Success, try `convox apps`
```

See the [Getting Started Guide](http://convox.com/docs/getting-started/) for more instructions.

### Development Quick Start

You need a Rack installed on AWS, a laptop with the Convox CLI, Go, and Docker, jq, and the Rack repo to run and develop the Rack API locally.


```
# Copy Rack AWS credentials and resource names to your development environment

$ STACK_NAME=$(convox api get /system | jq -r .name)
$ WEB_PID=$(convox api get /apps/$STACK_NAME/processes | jq -r '.[] | select(.name == "web") | .id' | head -1)
$ convox exec $WEB_PID env --app $STACK_NAME > .env

# Check out the Rack golang package

$ go get github.com/convox/rack/...
$ cd $GOPATH/src/github.com/convox/rack

# Start Rack locally in Docker

$ docker-machine start default
$ convox start
RUNNING: docker build -t convox-icytafnqqb /Users/noah/go/src/github.com/convox/rack
web      | running: docker run -i --name rack-web...
web      | [negroni] listening on :3000

# Log into the development server

$ convox login ($docker-machine ip default)
```

See the [Development Guide](Development.md) for more instructions to develop, contribute and release changes for Rack and related components.

## Contributing

* Join the [Convox Slack](https://invite.convox.com) channel to ask questions from the community and team
* Open a [GitHub Issue](https://github.com/convox/rack/issues/new) for bugs and feature requests
* Initiate a [GitHub Pull Request](https://help.github.com/articles/using-pull-requests/) to submit patches

## License

Apache 2.0 &copy; 2015 Convox, Inc.
