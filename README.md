# convox/app

<a href="https://travis-ci.org/convox/app">
  <img align="right" src="https://travis-ci.org/convox/app.svg?branch=master">
</a>

Create a CloudFormation template from an app manifest.

This is a guide to developing the convox/app project. For detailed
installation and usage instructions, see [http://docs.convox.com/](http://docs.convox.com/).

## Development

```bash
$ go get github.com/convox/app
$ cd $GOPATH/src/github.com/convox/app
$ make test
$ make build

$ cat fixtures/web_postgis.yml | docker run -i convox/app
{
  "AWSTemplateFormatVersion": "2010-09-09",
  ...
}
```

## Contributing

* Open a [GitHub Issue](https://github.com/convox/app/issues/new) for bugs and feature requests
* Initiate a [GitHub Pull Request](https://help.github.com/articles/using-pull-requests/) for patches

## See Also

* [convox/app](https://github.com/convox/app)
* [convox/build](https://github.com/convox/build)
* [convox/cli](https://github.com/convox/cli)
* [convox/kernel](https://github.com/convox/kernel)

## License

Apache 2.0 &copy; 2015 Convox, Inc.
