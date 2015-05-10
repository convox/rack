# convox/env

Encrypt and decrypt environments.

## Usage

    $ KEY=arn:aws:kms:us-east-1:000000000000:key/00000000-0000-0000-0000-000000000000

    $ cat .env | docker run convox/env encrypt $KEY | docker run convox/env decrypt $KEY

## License

Apache 2.0 &copy; 2015 Convox, Inc.
