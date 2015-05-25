package models

import (
	"os"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws/credentials"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/cloudformation"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/cloudwatch"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/dynamodb"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ecs"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/kinesis"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/rds"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/s3"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/sqs"
)

var SortableTime = "20060102.150405.000000000"

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

func CloudFormation() *cloudformation.CloudFormation {
	return cloudformation.New(&aws.Config{
		Credentials: credentials.NewCredentials(&AwsCredentials{}),
		Region:      os.Getenv("AWS_REGION"),
	})
}

func CloudWatch() *cloudwatch.CloudWatch {
	return cloudwatch.New(&aws.Config{
		Credentials: credentials.NewCredentials(&AwsCredentials{}),
		Region:      os.Getenv("AWS_REGION"),
	})
}

func DynamoDB() *dynamodb.DynamoDB {
	return dynamodb.New(&aws.Config{
		Credentials: credentials.NewCredentials(&AwsCredentials{}),
		Region:      os.Getenv("AWS_REGION"),
	})
}

func EC2() *ec2.EC2 {
	return ec2.New(&aws.Config{
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

func Kinesis() *kinesis.Kinesis {
	return kinesis.New(&aws.Config{
		Credentials: credentials.NewCredentials(&AwsCredentials{}),
		Region:      os.Getenv("AWS_REGION"),
	})
}

func RDS() *rds.RDS {
	return rds.New(&aws.Config{
		Credentials: credentials.NewCredentials(&AwsCredentials{}),
		Region:      os.Getenv("AWS_REGION"),
	})
}

func S3() *s3.S3 {
	return s3.New(&aws.Config{
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
