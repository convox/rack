package cloudformation

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/pkg/errors"
)

var (
	maxRetry = 10
)

func DescribeStacks(input *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
	s, err := session.NewSession(&aws.Config{
		MaxRetries: aws.Int(maxRetry),
		Retryer: client.DefaultRetryer{
			NumMaxRetries:    maxRetry,
			MinRetryDelay:    1 * time.Second,
			MaxRetryDelay:    5 * time.Second,
			MinThrottleDelay: 10 * time.Second,
			MaxThrottleDelay: 60 * time.Second,
		},
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	cf := cloudformation.New(s)

	res, err := cf.DescribeStacks(input)
	if err != nil {
		return nil, err
	}

	return res, nil
}
