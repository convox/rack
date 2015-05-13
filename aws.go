package main

import (
	"github.com/convox/build/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/build/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws/credentials"
	"github.com/convox/build/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ec2"
)

type Credentials struct {
	Builder *Builder
}

func (cc *Credentials) IsExpired() bool {
	return false
}

func (cc *Credentials) Retrieve() (credentials.Value, error) {
	creds := credentials.Value{
		AccessKeyID:     cc.Builder.AwsAccess,
		SecretAccessKey: cc.Builder.AwsSecret,
	}

	return creds, nil
}

func EC2(b *Builder) *ec2.EC2 {
	return ec2.New(&aws.Config{
		Credentials: credentials.NewCredentials(&Credentials{Builder: b}),
		Region:      b.AwsRegion,
	})
}
