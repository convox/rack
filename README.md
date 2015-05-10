# convox/env

Encrypt and decrypt environments with [AWS KMS](http://aws.amazon.com/kms/).

## Usage

#### CLI

```shell
$ KEY=arn:aws:kms:us-east-1:000000000000:key/00000000-0000-0000-0000-000000000000

$ cat .env | docker run convox/env encrypt $KEY | docker run convox/env decrypt $KEY
```

#### Golang

```go
import "github.com/convox/env/crypt"

const (
  Key = "arn:aws:kms:us-east-1:000000000000:key/00000000-0000-0000-0000-000000000000"
)

cr := crypt.New("region", "access", "secret")

// encrypt a secret
envelope, err := cr.Encrypt(Key, []byte("some sensitive data"))
data, err := envelope.Marshal()

// decrypt an envelope
envelope, err := crypt.UnmarshalEnvelope(data)
decrypted, err := cr.Decrypt(Key, envelope)
```

## License

Apache 2.0 &copy; 2015 Convox, Inc.
