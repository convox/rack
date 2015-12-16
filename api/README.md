# convox/api

## Development

### Prerequisites

The Convox API coordinates AWS so we develop against an AWS environment created
by the `convox install` command.

Using the TEMPLATE_FILE variable tells the installer to use the CloudFormation
template in the local checkout instead of a published template in S3.

Passing the `--development` flag tells the installer to print out a set of ENV
vars the API needs to talk to the AWS resources it just provisioned.

Copy and paste these values to a `.env` file so that `convox start` can read them.

    $ TEMPLATE_FILE=api/dist/kernel.json install --stack-name=convox-dev --development ~/credentials.csv

    # add env vars to .env

### Running

Now that you have your AWS resources in place and your ENV configured in `.env`, all you need to do is:

    $ convox start

to boot a local API instance. You can then login to the API via the command line:

    $ convox login <host> --password $PASSWORD

Where `<host>` is going to be the hostname or ip address of your local docker install. If you're on linux
then its simply `localhost`. If you're on OS X, then it is one of:

    - `docker-machine ip <machine>`
    - `boot2docker ip`

And the `PASSWORD` is in the ENV data in your `.env` file.

### `convox build`

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
