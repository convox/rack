# Convox Rack Development Guide

Rack is open source with the goal of everyone to understand the platform, provide suggestions, and contribute improvements.

This guide documents how to:

* Set up a sandbox AWS Rack to develop and test infrastructure template changes
* Set up a laptop with a Docker VM to run, develop and test API changes locally
* Run the Rack unit test suite locally
* Submit changes upstream
* Run an AWS integration test suite locally or on a CI server
* Release artifacts to enable `convox rack update`

## Sponsorship and Mentoring

Developing Rack will incur AWS costs. If this is an obstacle for you to contribute you can contact [support@convox.com](mailto:support@convox.com) to request sponsorship.

Much of the technical design and implementation in Rack requires understanding of AWS, Docker, Golang, systems engineering and more. If you would like to better learn these systems to contribute, you can contact [support@convox.com](mailto:support@convox.com), join the [Public Slack](http://invite.convox.com/), or open up issues on [GitHub](http://github.com/convox/rack) to ask questions and/or request a mentor.

## Sandbox AWS Rack Install

Rack consumes numerous AWS and Docker APIs. The easiest way to develop Rack is with real AWS access keys interacting with real AWS resources like a Dynamo Table, ECS Cluster, and CloudFormation Stack.

This is easy to bootstrap with the Rack project itself:

```
$ convox install --stack-name dev
```

You can also install a Rack with the CloudFormation template on master or with your own changes by:

```
$ TEMPLATE_FILE=api/dist/kernel.json convox install --stack-name=convox-dev
```

You can also use any existing Rack with the caveat that running a local Rack against it could have side effects like terminating instances.

## AWS Rack Ingress

Parts of the API like `convox ps --stats` and `convox instances ssh` interact with the Docker daemon and other services running on various AWS instance. To enable these calls to work locally you will want to open up access to the instances:

* Open the [Security Group Management Console](https://console.aws.amazon.com/ec2/v2/home?region=us-east-1#SecurityGroups)
* Select the Security Group with the Group Name like "dev-SecurityGroup-4PNOYR5HUH83" and the Description "Instances"
* Click the Inbound tab, then the Edit button, then the Add Rule button
* Keep Custom TCP Rule and TCP protocol
* Add "0 - 65535" for Port Range, and select "My IP" for Source
* Click the Save button

**Warning: DO NOT expose the inbound rules to "Anywhere"! This will expose your instances (including the Docker daemons) to the whole Internet.**

## Rack Golang Project

Rack is written in Golang. To setup a Go environment, see the excellent [Getting Started](https://golang.org/doc/install) docs. You can then clone and build the project with the `go get` tool:

```
$ go get github.com/convox/rack/...
```

After this, `which convox` should refer to `$GOPATH/bin/convox`.

## Local Rack Environment

The local Rack is running an API process that has AWS Access Keys, AWS resource names, and other various settings in its environment. You need to copy this to your laptop.

```
$ cd $GOPATH/src/github.com/convox/rack

# Introspect the dev rack to find the PID of the API web process

$ STACK_NAME=$(convox api get /system | jq -r .name)
$ WEB_PID=$(convox api get /apps/$STACK_NAME/processes | jq -r '.[] | select(.name == "web") | .id' | head -1)

# Introspect the API web process to get its environment

$ convox exec $WEB_PID env --app $STACK_NAME > .env
```

Now you have a bunch of secrets that will let you interact with AWS APIs from your laptop:

```
$ cat .env
CLIENT_ID=noah@convox.com
RACK=dev
SUBNETS=subnet-13de3139,subnet-b5578fc3,subnet-21c13379
AWS_ACCESS=AKIAIOSFODNN7EXAMPLE
AWS_SECRET=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
AWS_REGION=us-east-1
DYNAMO_RELEASES=dev-releases
CLUSTER=dev-Cluster-1E4XJ0PQWNAYS
PASSWORD=45e0f109-3f56-4b30-9b5a-b0939b8a4c25
...
```

## Local Rack Docker VM

A local Rack is started with `convox start` which requires a working Docker environment. To setup a Docker environment, see the [Docker Machine](https://docs.docker.com/machine/) docs. You can then run the project:

```
$ docker-machine start default
$ eval $(docker-machine env default)

$ convox start
RUNNING: docker build -t convox-icytafnqqb /Users/noah/go/src/github.com/convox/rack
web      | running: docker run -i --name rack-web...
web      | [negroni] listening on :3000
```

Now you can log into the development Rack API and interact with your Convox resources:

```
$ convox login $(docker-machine ip default)

$ convox instances
ID          AGENT  STATUS  STARTED      PS  CPU    MEM   
i-6cf228f7  on     active  2 hours ago  4   0.00%  10.42%
i-146c2f97  on     active  2 hours ago  1   0.00%  3.21%
i-c7de605c  on     active  2 hours ago  0   0.00%  0.00%
```

## Golang Unit Test Suite

Rack has a suite of Golang unit and integration tests that offer very fast feedback (< 1 minute) about system correctness. They require a running Docker environment.

```
$ make test
docker info >/dev/null
go get -t ./...
env PROVIDER=test go test -v -cover ./...
=== RUN   TestGitUrl
=== RUN   TestDockerCompose
=== RUN   TestEnvFile
...

$ echo $?
0
```

GitHub and Travis CI are configured to require that tests are passing before a pull request can be merged.

The most complex tests, such as `TestProcessesListWithDetached` setup a stub AWS and Docker httptest web servers to simulate various request and response cycles. This can be challenging to write but represents a very powerful way to verify Convox behavior.

## API Changes

A common thing to do is to fix a bug or make an enhancement to the Rack APIs. For example, maybe the `POST /apps/{app}/builds` endpoint would be more helpful if it accepted a GitLab URL to clone and build from, so you'd like to add this.

The `convox/rack/api` package has a few key concepts:

* Swagger Manifest (rack/api/manifest.yml). Defines all API endpoints and responses.
* Golang Client (rack/client). Bindings that talk to the HTTP API and returns Golang structs, slices and errors.
* CLI (rack/cmd/convox). High level tool that lets developers issue commands like `convox/deploy`.
* Routes (rack/api/controllers/routes). A `gorilla/mux` configuration of request URL patterns, HTTP verbs, and handler functions.
* Controllers (rack/api/controllers). HTTP handlers for every route.
* Models (rack/api/models). Key primatives like "app", "service", "build", and "release" and corresponding logic to control AWS and Docker.

It is common for API changes to require corresponding changes across a model, controller, swagger manifest, client and CLI.

When `convox start` successfully starts the Rack web, monitor, and registry processes locally, changes to the Golang source are detected, and the web process is rebuilt and restarted.

## Systems Changes

Many API calls need to execute changes across subsystems. For example:

* `convox build` needs to create a new Docker container for the build and collect its output and return code
* `convox release promote` needs to perform a CloudFormation stack update

Systems engineering best practices are encouraged:

* Robust error handling
* Logging that makes a developer's life easier
* Logging that can be turned into operational metrics (e.g. `count#push.retry=1)`
* Code strategies that make it easy to simulate subsystem requests/responses in a test environment

## Infrastructure Changes

Racks apps and services are created, updated and destroyed via automated means. This is a DevOps best practice that minimizes human errors and accidents that cause downtime. On AWS this is accomplished with CloudFormation. Some examples of changes:

* Rack should have a new option to provision and use private subnets
* An app load balancer should have a new option to configure Proxy Protocol
* `convox services create elasticsearch` should provision an ElasticSearch cluster

Some general notes when making changes to the infrastructure templates:

* Run `make -C api templates` to compile the templates and restart the webserver. The `templates.go` file updates should be checked in.
* Run `make test` to exercise the app template regression tests. Changes to app.tmpl almost always need accompanying test changes.
* Pay careful attention to both the update and rollback safety of changes. Rollbacks are extremely important for failure recovery.
* Convox uses [CloudFormation Custom Resources](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/template-custom-resources.html) by releasing `api/cmd/formation` as a Lambda handler that every Rack and App can use to provision things that aren't supported by CloudFormation.
* Run `make fixtures` to rebuild the test fixtures after you change manifests. Carefully inspect the diff to ensure only your desired changes are made.

## Opening a Pull Request

Rack follows the traditional [GitHub Flow](https://guides.github.com/introduction/flow/) where all changes start as a pull request.

We encourage you to open a pull request for anything! For example:

* A fully designed and tested enhancement
* A untested but reasonable effort towards an enhancement
* Work in progress (WIP) for questions or review

The Rack maintainers aim to help land every reasonable pull request, and to provide clarity if a pull request can not be included.

## Checklists

Convox relies on checklists to safely and reliabily take code from a Pull Request to a published release. The standard release checklist is [https://github.com/convox/rack/blob/master/.github/PULL_REQUEST_TEMPLATE.md](PULL_REQUEST_TEMPLATE.md)

We aim to automate and simplify the checklist over the life of the project to make releasing software easy, fast, and safe.

## Release Changes for `convox rack update`

The ultimate goal is to package changes so that a user can apply them with `convox rack update`. This involves:

* Generate a release ID to tag every artifact with
* Tagging a commit in GitHub with the release ID
* Build Docker Images for the Rack API and Registry and publish them to Docker Hub
* Build and publish Zip files for the Rack CloudFormation Lambda Handler and publish into public S3 files for every region Convox supports
* Inject the release ID into kernel.json and publish it to S3
* Appending the release ID to releases.json in S3
* Setting/unsetting the "published" bit in the releases.json file in S3

Convox coordinates this with the [convox/release](https://github.com/convox/release) utility and Slack.

This functionality needs to be merged into convox/rack and generalized to support registries and S3 buckets that are not owned by Convox. See [Issue #447](https://github.com/convox/rack/issues/477) for more details.

To publish a release of both the API and CLI, issue Slack commands:

```
/release create
rack release created: 20160328231208

/release cli
cli release created

/release publish 20160328231208
```

All users will get this when they issue:

```
$ convox rack update
```

For testing, you often want to build from a branch and not publish it without additional testing. To release a branch, issue a Slack command:

```
/release create my-branch
rack release created: 20160328231208-my-branch
```

A Rack can use this release by specifying a specific version:

```
$ convox rack update 20160328231208-my-branch
```

## AWS Integration Test Suite

Rack has a suite of integration tests that install, deploy apps, then tear down Racks on AWS, then collect lots of logs for analysis afterwareds. This is slow feedback (~45 minutes) but offers good guarantees of general release quality.

Currently it deploys 3 Racks into 3 different regions and deploys, introspects, then deletes two apps on each Rack.

This is run on CircleCI which coordinates parallelizing the regions, collecting artifacts, and reporting results. An example test run can be reviewed [here](https://circleci.com/gh/convox/rack/667). You need to sign into CircleCI and have access to the Rack repo to review the artifacts.

You can also run the CI scripts locally:

```
$ export AWS_ACCESS_KEY_ID=foo
$ export AWS_SECRET_ACCESS_KEY=bar
$ export AWS_REGION=us-east-1
$ export AWS_DEFAULT_REGION=us-east-1
$ export VERSION=20160323164322

$ ci/install.sh
$ ci/tests/example-app httpd
$ ci/tests/example-app node-workers
$ ci/delete-all-apps.sh
$ ci/uninstall.sh
$ ci/cleanup.sh
```

This generally uses a specific version number (e.g. 20160323164322) that has been released but not published. Passing integration tests offer confidence that the release can be published.
