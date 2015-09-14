package models

import (
	"bytes"
	"fmt"
	"html/template"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
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

func awsConfig() *aws.Config {
	return &aws.Config{
		Credentials: credentials.NewCredentials(&AwsCredentials{}),
	}
}

func AutoScaling() *autoscaling.AutoScaling {
	return autoscaling.New(awsConfig())
}

func CloudFormation() *cloudformation.CloudFormation {
	return cloudformation.New(awsConfig())
}

func CloudWatch() *cloudwatch.CloudWatch {
	return cloudwatch.New(awsConfig())
}

func DynamoDB() *dynamodb.DynamoDB {
	return dynamodb.New(awsConfig())
}

func EC2() *ec2.EC2 {
	return ec2.New(awsConfig())
}

func ECS() *ecs.ECS {
	return ecs.New(awsConfig())
}

func Kinesis() *kinesis.Kinesis {
	return kinesis.New(awsConfig())
}

func RDS() *rds.RDS {
	return rds.New(awsConfig())
}

func S3() *s3.S3 {
	return s3.New(awsConfig())
}

func SQS() *sqs.SQS {
	return sqs.New(awsConfig())
}

func buildTemplate(name, section string, data interface{}) (string, error) {
	data, err := Asset(fmt.Sprintf("templates/%s.tmpl", name))

	tmpl, err := template.New(section).Funcs(templateHelpers()).Parse(string(data.([]byte)))

	if err != nil {
		return "", err
	}

	var formation bytes.Buffer

	err = tmpl.Execute(&formation, data)

	if err != nil {
		return "", err
	}

	return formation.String(), nil
}
