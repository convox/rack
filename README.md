# convox/build

Turn a Compose-enabled application into an AMI.

## Usage

    $ docker run \
      -e AWS_REGION=us-east-1 \
      -e AWS_ACCESS=foo \
      -e AWS_SECRET=bar \
      convox/build sinatra-example https://github.com/convox-examples/sinatra

## Userdata

The AMIs produced by this tool will need userdata like the following to boot:

    {
      "app": "myapp",
      "process": "web",
      "command": "",
      "env": "http://convox.io/example.env",
      "logs": {
        "kinesis": "kinesis-stream",
        "cloudwatch": "cloudwatch-logs-group"
      },
      "ports": [
        "5000:3000",
        "5001:3001"
      ]
    }

## See Also

* [convox/architect](https://github.com/convox/architect)

## License

Apache 2.0 &copy; 2015 Convox, Inc.
