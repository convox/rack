package sqs

import "github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"

func init() {
	initRequest = func(r *aws.Request) {
		setupChecksumValidation(r)
	}
}
