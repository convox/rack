# convox/build

<a href="https://travis-ci.org/convox/build">
  <img align="right" src="https://travis-ci.org/convox/build.svg?branch=master">
</a>

Create Docker images from an app directory, repo or tarball.

This is a guide to developing the convox/build project. For detailed
installation and usage instructions, see [http://docs.convox.com/](http://docs.convox.com/).

## Development

```bash
$ go get github.com/convox/build
$ cd $GOPATH/src/github.com/convox/build
$ make test
$ make build

$ docker run -v /var/run/docker.sock:/var/run/docker.sock \
convox/build worker https://github.com/convox-examples/worker.git
git|Cloning into '/tmp/repo662108518/clone'...
manifest|worker:
manifest|  build: .
build|RUNNING: docker build -t oawshqivmr /tmp/repo662108518/clone
...
build|RUNNING: docker tag -f oawshqivmr example-sinatra/worker
```

## Contributing

* Open a [GitHub Issue](https://github.com/convox/build/issues/new) for bugs and feature requests
* Initiate a [GitHub Pull Request](https://help.github.com/articles/using-pull-requests/) for patches

## See Also

* [convox/app](https://github.com/convox/app)
* [convox/build](https://github.com/convox/build)
* [convox/cli](https://github.com/convox/cli)
* [convox/kernel](https://github.com/convox/kernel)

## License

Apache 2.0 &copy; 2015 Convox, Inc.
