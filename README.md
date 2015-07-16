# convox/cli 

<a href="https://travis-ci.org/convox/cli">
  <img align="right" src="https://travis-ci.org/convox/cli.svg?branch=master">
</a>

Convox command line interface.

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

* Open a GitHub Issue for bugs and detailed feature requests
* Initiate a [GitHub Pull Request](https://help.github.com/articles/using-pull-requests/) for patches

## License

Apache 2.0 &copy; 2015 Convox, Inc.
