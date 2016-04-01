# build2

Build, tag and push Docker images from an app's source.

Source is either a .tgz snapshot of an app directory or an URL to a git
repository and optional commit.

This command is designed to run in the convox/rack Docker image which is a
basic Linux environment with `docker`, `git`, and `ssh` available for extracting
or cloning the app source.

## Prerequisites

Usage:

```
$ build2 -
$ build2 github.com/convox-examples/sinatra.git
```

Docker Run Flags:

* `-v /var/run/docker.sock:/var/run/docker.sock` so it can acess the Docker daemon and execute `docker build`
* `-i`

Environment:

* APP - Name of the app we are building for
* BUILD - Id of the build
* DOCKER_AUTH - Json blob of private registry auth info
* RACK_HOST - Hostname to call back on build success or failure
* RACK_PASSWORD - Password to call back on build success or failure
* REGISTRY_EMAIL - Credentials to `docker push`
* REGISTRY_USERNAME - Credentials to `docker push`
* REGISTRY_PASSWORD - Credentials to `docker push`
* REGISTRY_ADDRESS - Credentials to `docker push`
* MANIFEST_PATH - Optional path if not docker-compose.yml
* REPOSITORY - Optional namespace that every image should use
* NO_CACHE - Option to build without reusing cache

## Development

    # rebuild the API image and build cmd

    $ cd convox/rack
    $ docker build -t rack/api .
    ...
    RUN go install ./...

    # build a tarball on stdin

    $ cd httpd
    $ tar cz . | docker run -i -v /var/run/docker.sock:/var/run/docker.sock rack/api \
      build2 httpd -