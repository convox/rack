# convox/crypt

Encrypt/decrypt sensitive information

## Usage

```shell
# create a key in KMS
KEY=arn:aws:kms:us-east-1:000000000000:key/00000000-0000-0000-0000-000000000000

# set up IAM credentials with access to Decrypt and GenerateDataKey on that key
$ cat <<EOF >.env
AWS_REGION=...
AWS_ACCESS=...
AWS_SECRET=...
EOF

# encrypt
$ cat .env | docker run --env-file .env -i convox/crypt encrypt $KEY > env.encrypted

# decrypt
$ cat env.encrypted | docker run --env-file .env -i convox/crypt decrypt $KEY > .env
```

## License

Apache 2.0 &copy; 2015 Convox, Inc.
