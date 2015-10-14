package sqs

import "github.com/convox/rack/api/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/request"

func init() {
	initRequest = func(r *request.Request) {
		setupChecksumValidation(r)
	}
}
