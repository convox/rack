# convox/cli

Convox command line interface.

## Installation

    $ go get github.com/convox/cli/convox

## Usage

    $ convox help

## convox start

Start runs any app with [Docker Compose](https://docs.docker.com/compose/).

If `docker-compose.yml` and/or `Dockerfile` do not exist, start will create them
for you.

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

## License

Apache 2.0 &copy; 2015 Convox, Inc.
