# convox/cli

<a href="https://travis-ci.org/convox/cli">
  <img align="right" src="https://travis-ci.org/convox/cli.svg?branch=master">
</a>

Launch a private cloud and deploy apps from the command line.

This is a guide to developing the convox/cli project. For detailed
installation and usage instructions, see [http://docs.convox.com/](http://docs.convox.com/).

## Development

    $ go get github.com/convox/cli/convox
    $ cd $GOPATH/src/github.com/convox/cli
    $ make test
    $ make install

    $ convox help
    convox: private cloud and application management

    Usage:
      convox <command> [args...]
    ...

## Contributing

* Open a [GitHub Issue](https://github.com/convox/cli/issues/new) for bugs and feature requests
* Initiate a [GitHub Pull Request](https://help.github.com/articles/using-pull-requests/) for patches

## License

Apache 2.0 &copy; 2015 Convox, Inc.
