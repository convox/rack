# builder

Turn a Compose-enabled application into an AMI.

## Usage

    $ docker run \
      -e AWS_REGION=us-east-1 \
      -e AWS_ACCESS=foo \
      -e AWS_SECRET=bar \
      convox/builder https://github.com/convox-examples/sinatra sinatra-example

## Userdata

The AMIs produced by this tool will need userdata like the following to boot:

    {
      "start": "name-of-compose-process",
      "env": [
        "FOO=bar",
        "BAZ=qux"
      ],
      "ports": [ 5000 ]
    }

## See Also

* [convox/architect](https://github.com/convox/architect)

## License

Apache 2.0 &copy; 2015 Convox, Inc.
