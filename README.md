# convox/env

Encrypt and decrypt environments.

## Usage

    $ cat .env | docker run convox/env encrypt arn:aws:kms:us-east-1:000000000000:key/00000000-0000-0000-0000-000000000000 > env.kms

    $ cat env.kms | docker run convox/env decrypt arn:aws:kms:us-east-1:000000000000:key/00000000-0000-0000-0000-000000000000

## Test

    $ cat .env
    AWS_REGION=...
    AWS_ACCESS=...
    AWS_SECRET=...
    KEY=...

    $ forego run make test

## License

Apache 2.0 &copy; 2015 Convox, Inc.
