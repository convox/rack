# convox/env

Encrypt and decrypt environments with [AWS KMS](http://aws.amazon.com/kms/).

## Usage

#### CLI

```shell
# set up an .env file
$ cat .env
AWS_REGION=...
AWS_ACCESS=...
AWS_SECRET=...

# get the ARN of a key in KMS 
$ KEY=arn:aws:kms:us-east-1:000000000000:key/00000000-0000-0000-0000-000000000000

# encrypt credentials
$ cat .env | docker run --env-file .env -i convox/env encrypt $KEY > env.encrypted

# decrypt credentials
$ cat env.encrypted | docker run --env-file .env -i convox/env decrypt $KEY > .env
```

#### Golang

```go
import "github.com/convox/env/crypt"

const Key = "arn:aws:kms:us-east-1:000000000000:key/00000000-0000-0000-0000-000000000000"

// specify aws credentials
cr := crypt.New("region", "access", "secret")

// use iam role on an instance (not implemented yet)
cr := crypt.NewIamRole()

// encrypt a secret
enc, err := cr.Encrypt(Key, []byte("some sensitive data"))

// decrypt a secret
dec, err := cr.Decrypt(Key, enc)
```

## License

Apache 2.0 &copy; 2015 Convox, Inc.
