package crypt

import (
	"github.com/convox/env/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/env/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/convox/env/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/kms"
)

type Credentials struct {
	Crypt *Crypt
}

func (cc *Credentials) IsExpired() bool {
	return false
}

func (cc *Credentials) Retrieve() (credentials.Value, error) {
	creds := credentials.Value{
		AccessKeyID:     cc.Crypt.AwsAccess,
		SecretAccessKey: cc.Crypt.AwsSecret,
		SessionToken:    cc.Crypt.AwsToken,
	}

	return creds, nil
}

func KMS(c *Crypt) *kms.KMS {
	return kms.New(&aws.Config{
		Credentials: credentials.NewCredentials(&Credentials{Crypt: c}),
		Region:      c.AwsRegion,
	})
}
