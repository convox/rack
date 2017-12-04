# convox/syslog

Effortlessly forward all your container logs to a Syslog server.

On a container cluster, logs are generated from countless containers on countless hosts. We need to:

* Collect all the container logs
* Forward logs off the instances a centralized **utility log service**
* Deliver, process, reformat and forward every log to one or more 3rd party log services

On AWS, CloudWatch Logs is the **utility log service**, offering high ingestion throughput and cheap storage. A Logs Subscription Filter **coordinates** the deceptively tough job of **delivering every log** to a custom Lambda Function **log processor** and **syslog forwarder**.

With Convox and AWS, you get a log pipeline that is:

* **Simple.** `convox logs` and `convox services add syslog --url` are all you need to know
* **Secure.** Logs stay in your AWS account and are wrapped in TLS before going anywhere else
* **Reliable.** Let AWS do all the heavy-lifting
* **Scalable.** 5 MB/sec per app
* **Affordable.** 10 GB/mo of logs costs $5/mo to ingest and pennies to store and process
* **Configurable.** Open source means you can understand and hack

## Usage

```bash
$ convox services create syslog --name pt --url tcp+tls://log1.papertrailapp.com:11235
$ convox services link pt --app myapp
```

## How It Works

Infrastructure:

* Logs are sent from ECS to CloudWatch logs with the Docker awslogs driver.

Install:

* Lambda Function, Permission, Role and Subscription are managed with CloudFormation
* In convox/rack, the LogGroup Parameter is substitued with many App LogGroup values

Invoke:

* On Lambda function invoke, describe CF stack to get runtime information like destination URL
* Cache the URL to /tmp/url minimize subsequent CF DescribeStack calls

Process:

* Dial syslog URL
* Unpack CloudWatch Log events from Lambda context
* Write logs lines

Report:

* Log errors to Lambda CloudWatch Logs
* TODO: Log how many lines processed, and how many successful and failed lines transmitted to CloudWatch Custom Metrics

## Contributing

* Join the [Convox Slack](https://invite.convox.com) channel to ask questions from the community and team
* Open a [GitHub Issue](https://github.com/convox/rack/issues/new) for bugs and feature requests
* Initiate a [GitHub Pull Request](https://help.github.com/articles/using-pull-requests/) to submit patches

## References

* convox/papertrail - https://github.com/convox/papertrail
* Docker awslogs driver - https://github.com/docker/docker/tree/master/daemon/logger/awslogs
* LambdaProc - https://github.com/jasonmoo/lambda_proc
* Sparta - https://github.com/mweagle/Sparta
* srslog - https://github.com/RackSec/srslog
