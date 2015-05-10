# convox/env

Encrypt and decrypt environments.

## Usage

    $ cat .env | docker run convox/env encrypt arn:aws:kms:us-east-1:000000000000:key/00000000-0000-0000-0000-000000000000 > env.kms

    $ cat env.kms | docker run convox/env decrypt arn:aws:kms:us-east-1:000000000000:key/00000000-0000-0000-0000-000000000000

## License

Apache 2.0 &copy; 2015 Convox, Inc.
