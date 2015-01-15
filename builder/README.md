# Convox Kernel

## Running

    $ docker run \
      -p 5000:5000 \
      -e AWS_ACCESS=access_key \
      -e AWS_SECRET=secret_key \
      -e AWS_REGION=us-east-1 \
      convox/kernel

## Hacking

#### Prerequisites

* [Go](https://golang.org/doc/install)

#### Running

```
$ make dev
```