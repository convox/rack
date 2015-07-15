# convox/cli 

<a href="https://travis-ci.org/convox/cli">
  <img align="right" src="https://travis-ci.org/convox/cli.svg?branch=master">
</a>

Convox command line interface.
## Prerequisites

`convox` depends on a working Docker 1.7 environment. See the [Docker Setup Guide](docker-setup.md) for more information.

## Installation

**OS X**

    $ curl http://www.convox.com.s3.amazonaws.com/install-osx.sh | sudo bash

**Golang**

    $ go get -u github.com/convox/cli/convox

## Usage

    $ convox help

## convox start

Start runs any app with a [Docker Compose](https://docs.docker.com/compose/) configuration.

If `docker-compose.yml` and/or `Dockerfile` do not exist, start will create them
for you, then build and pull images.

    $ cd myapp
    $ convox start
    Procfile app detected. Writing Dockerfile, docker-compose.yml.
    Step 0 : FROM convox/cedar
    ...

    Trigger 7, RUN /buildkit/bin/detect /app /cache
    Step 0 : RUN /buildkit/bin/detect /app /cache
     ---> Running in d7bef824d297
    Buildkit+Ruby
    Trigger 8, RUN /buildkit/bin/compile /app /cache
    Step 0 : RUN /buildkit/bin/compile /app /cache
     ---> Running in 731df369192e
    -----> Compiling for Ruby
    ...

    Successfully built d4c4605f1f09
    [2015-06-23 17:31:42] INFO  WEBrick 1.3.1
    [2015-06-23 17:31:42] INFO  ruby 2.1.3 (2014-09-19) [x86_64-linux]
    == Sinatra (v1.4.6) has taken the stage on 3000 for production with backup from WEBrick
    [2015-06-23 17:31:42] INFO  WEBrick::HTTPServer#start: pid=1 port=3000

    $ curl $(boot2docker ip):5000
    Hello, World

Start will also help set up and debug your Docker / Boot2Docker environment when
it encounters problems.

## convox login

Login to your Convox API.

    $ convox login convox-424363854.us-east-1.elb.amazonaws.com
    Password: 
    Login Succeeded

## convox deploy

Deploy any app to AWS.

If `docker-compose.yml` and/or `Dockerfile` do not exist, deploy will create 
them for you, then build and pull images. Then deploy tags images and pushes 
them to your private registry and creates an app and release.

    $ cd myapp
    $ convox deploy
    Docker Compose app detected.
    web uses an image, skipping
    latest: Pulling from httpd
    ...

    Tagging httpd
    Pushing convox-424363854.us-east-1.elb.amazonaws.com:5000/myapp-web:1435598703
    ...

    Created app myapp6
    Status running
    Created release 1435598703
    Status running

## convox apps

List apps.

    $ convox apps
    myapp
    myapp2

## convox build

Build an app for local development.

    $ cd myapp
    $ convox build
    Docker Compose app detected.
    Building web...
    Step 0 : FROM gliderlabs/alpine:3.1
    Pulling repository gliderlabs/alpine
    ...

## convox debug

Get an app's system events for debugging purposes.

    $ convox debug
    2015-06-10T16:11:07Z: [CFM] (myapp) CREATE_IN_PROGRESS User Initiated
    2015-06-10T16:11:32Z: [CFM] (ServiceRole) CREATE_IN_PROGRESS
    2015-06-10T16:11:32Z: [CFM] (DynamoChanges) CREATE_IN_PROGRESS
    ...

## convox env

Inspect and edit environment variables.

    $ convox env set FOO=bar BAZ=qux

    $ convox env
    BAZ=qux
    FOO=bar

    $ convox env set FOO=quux CORGE=grault

    $ convox env
    BAZ=qux
    CORGE=grault
    FOO=quux

    $ convox env unset FOO

    $ convox env
    BAZ=qux
    CORGE=grault

    $ convox env get BAZ
    qux

## convox info

See info about an app.

    $ convox info
    Name         myapp
    Status       running
    Release      RBWGAZAFGDI
    Web          [image]
    Web Host     myapp-104798329.us-east-1.elb.amazonaws.com:5000

## convox logs

Stream the logs for an app.

    $ convox logs
    web: 2015-07-01T22:06:39.409270747Z 10.0.1.92 - - [01/Jul/2015:22:06:39 +0000] "GET / HTTP/1.1" 200 883 0.0042
    web: 2015-07-01T22:06:44.036603010Z 10.0.1.92 - - [01/Jul/2015:22:06:44 +0000] "POST /message HTTP/1.1" 303 - 0.0037

## convox ps

List the app processes.

    $ convox ps
     web

## convox run

Run a one-off process.

## convox scale

Scale the number of processes for an app.

    $ convox scale 2
    Scale 2

## convox stop

Stop a process.

## convox update

Update the CLI.

    $ convox update
    Updated to 0.9

## License

Apache 2.0 &copy; 2015 Convox, Inc.
