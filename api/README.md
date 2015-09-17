# convox/api

## Development

    $ convox install --stack-name=convox-dev --development ~/credentials.csv

    # add env vars to .env

    $ convox start

### Building locally

For pushes to the convox registry to work locally, you need to configure the docker daemon to talk to the
registry that is running in Amazon.

You can get the convox registry from `REGISTRY_HOST` or the error output from docker when you try
to login. Pass this value to the `--engine-insecure-registry` flag of `docker-machine`
when you create a VM. This means you'll need to create a new machine via:

    docker-machine create --driver virtualbox --engine-insecure-registry $(cat .env | grep REGISTRY_HOST | tr -d 'REGISTRY_HOST=') convox-dev`

And then do `convox start` again against this new docker installation.

If you're running Docker yourself, just pass the `REGISTRY_HOST` as the option to  `--insecure-registry` when starting
the docker daemon.


## Contributing

* Open a [GitHub Issue](https://github.com/convox/rack/issues/new) for bugs and feature requests
* Initiate a [GitHub Pull Request](https://help.github.com/articles/using-pull-requests/) for patches

## License

Apache 2.0 &copy; 2015 Convox, Inc.
