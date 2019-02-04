# convox/rack

Convox Rack is a private PaaS that runs in an AWS account.

## Usage

### Installation

Install a Rack from [Convox Console](https://console.convox.com)

### Usage

```console
$ convox apps create myapp
Creating app myapp... OK

$ convox env set SECRET=foo -a myapp
Setting SECRET... OK, RABCDEFGH

$ convox deploy ~/src/myapp -a myapp
Building myapp... OK
Creating release RHGFEFCBA... OK
Promoting release RHGFEFCBA... OK
```

## Related

* [Convox Console](https://console.convox.com)
* [Documentation](https://docs.convox.com)
* [Forums](https://community.convox.com)

## License

Apache 2.0
