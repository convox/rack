# convox/kernel

<a href="https://travis-ci.org/convox/kernel">
  <img align="right" src="https://travis-ci.org/convox/kernel.svg?branch=master">
</a>

Coordinate AWS heavy lifting behind a simple API.

This is a guide to developing the convox/kernel project. For detailed
installation and usage instructions, see [http://docs.convox.com/](http://docs.convox.com/).

## Development

Pre-reqs

* [Boot2Docker](http://boot2docker.io/)
* `mkdir -p ~/.convox ; echo dev@example.com > ~/.convox/id`
* A sandbox `DEVELOPMENT=Yes STACK_NAME=convox-dev convox install`
* An .env file with all the convox-dev stack outputs, i.e. `DYNAMO_BUILDS=convox-dev-builds`

```bash
$ go get github.com/convox/kernel
$ cd $GOPATH/src/github.com/convox/kernel

$ make dev
Attaching to kernel_web_1, kernel_registry_1
registry_1 | [2015-07-16 22:20:09 +0000] [15] [INFO] Listening at: http://0.0.0.0:5000 (15)
web_1      | [negroni] listening on :3000

$ convox login $(boot2docker ip)
Password: <REGISTRY_PASSWORD>
Logged in successfully.
$ convox --version
client: dev
server: latest (192.168.59.103)
```

## Contributing

* Open a [GitHub Issue](https://github.com/convox/kernel/issues/new) for bugs and feature requests
* Initiate a [GitHub Pull Request](https://help.github.com/articles/using-pull-requests/) for patches

## See Also

* [convox/app](https://github.com/convox/app)
* [convox/build](https://github.com/convox/build)
* [convox/cli](https://github.com/convox/cli)
* [convox/kernel](https://github.com/convox/kernel)

## License

Apache 2.0 &copy; 2015 Convox, Inc.
