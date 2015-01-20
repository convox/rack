# Convox Builder

Turn git repos into AMIs.

## Usage

    $ cat .env
    AWS_REGION=us-east-1
    AWS_ACCESS=foo
    AWS_SECRET=bar

    $ docker run --env-file .env \
      convox/builder sinatra-example https://github.com/convox-examples/sinatra

## License

Apache 2.0 &copy; 2015 David Dollar
