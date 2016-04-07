# build

Build, tag and push Docker images from an app's source.

Source is either a .tgz snapshot of an app directory or an URL to a git
repository and optional commit.

This command is designed to run in the convox/rack Docker image which is a
basic Linux environment with `docker`, `git`, and `ssh` available for extracting
or cloning the app source.

## Usage

```
$ tar cz | build -
$ build github.com/convox-examples/sinatra.git
```

## Docker Run Flags

* `-v /var/run/docker.sock:/var/run/docker.sock` so it can acess the Docker daemon and execute `docker build`
* `-i` so it can read .tgz data from stdin

## Environment

This command tags images in a way that reflects a specific app and build id. Therefore these arguments
are required:

* `APP` - Name of the app we are building for
* `BUILD` - Id of the build
* `REGISTRY_ADDRESS` - Registry host to tag and push to

And the tag namespace is optional:

* `REPOSITORY` - Optional namespace that every image should use

Without `REPOSITORY`, tags use the app name as the repository, the process name as the image name, and the build id as the tag:

```
convox-826133048.us-east-1.elb.amazonaws.com:5000/sinatra/web:BANHPORIOTL 
```

With `REPOSITORY`, tags always share the same repository, and use the process name and build id as the tag:

```
132866487567.dkr.ecr.us-east-1.amazonaws.com/convox-sinatra-soppqmvrdv:web.BDQBBSNVTZD
```

This command may pull images that docker-compose.yml references, and will push new images to a remote registry.
These arguments along with `REGISTRY_ADDRESS` offer Docker authentication to push/pull:

* `DOCKER_AUTH` - Json blob of private registry auth info
* `REGISTRY_EMAIL` - Credentials to `docker push`
* `REGISTRY_USERNAME` - Credentials to `docker push`
* `REGISTRY_PASSWORD` - Credentials to `docker push`

If this command calls back to Rack to denote build status. Any error calls back to report "failed",
otherwise it reports "complete". These arguments offer Rack authentication to call back:

* `RACK_HOST` - Hostname to call back on build success or failure
* `RACK_PASSWORD` - Password to call back on build success or failure

A few options of the build are controlled by a user. These arguments override default assumptions for `docker build`:

* `MANIFEST_PATH` - Optional path if not docker-compose.yml
* `NO_CACHE` - Option to build without reusing cache

## Examples

    # rebuild the API image and build cmd

    $ cd convox/rack
    $ docker build -t rack/api .
    ...
    RUN go install ./...

    # build a tarball on stdin

    $ cd httpd
    $ tar cz . | docker run -i -v /var/run/docker.sock:/var/run/docker.sock rack/api \
      build httpd -