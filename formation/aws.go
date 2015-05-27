package formation

import (
	"os"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws/credentials"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ecs"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/lambda"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/sqs"
)

type AwsCredentials struct {
}

func (ec *AwsCredentials) IsExpired() bool {
	return false
}

func (ec *AwsCredentials) Retrieve() (credentials.Value, error) {
	creds := credentials.Value{
		AccessKeyID:     os.Getenv("AWS_ACCESS"),
		SecretAccessKey: os.Getenv("AWS_SECRET"),
	}

	return creds, nil
}

func Lambda() *lambda.Lambda {
	return lambda.New(&aws.Config{
		Credentials: credentials.NewCredentials(&AwsCredentials{}),
		Region:      os.Getenv("AWS_REGION"),
	})
}

func ECS() *ecs.ECS {
	return ecs.New(&aws.Config{
		Credentials: credentials.NewCredentials(&AwsCredentials{}),
		Region:      os.Getenv("AWS_REGION"),
	})
}

func SQS() *sqs.SQS {
	return sqs.New(&aws.Config{
		Credentials: credentials.NewCredentials(&AwsCredentials{}),
		Region:      os.Getenv("AWS_REGION"),
	})
}
