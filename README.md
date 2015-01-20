# Convox Builder

Turn a git repository with a fig.yml into an AMI.

## Usage

    $ docker run \
      -e AWS_REGION=us-east-1 \
      -e AWS_ACCESS=foo \
      -e AWS_SECRET=bar \
      convox/builder https://github.com/convox-examples/sinatra sinatra-example

## Userdata

The AMIs produced by this tool will need userdata like the following to boot:

    {
      "start": "name-of-fig-process",
      "env": [
        "FOO=bar",
        "BAZ=qux"
      ],
      "ports": [ 5000 ],
      "volumes": [
        "/ebs/data:/data",
        "/var/run/docker.dock"
      ]
    }

## License

Apache 2.0 &copy; 2015 David Dollar
