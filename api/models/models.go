package models

import (
	"bytes"
	"fmt"
	"html/template"
	"math"
	"os"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/iam"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/rds"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/s3"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/sns"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/sqs"
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

func CloudWatchLogs() *cloudwatchlogs.CloudWatchLogs {
	return cloudwatchlogs.New(awsConfig())
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

func IAM() *iam.IAM {
	return iam.New(awsConfig())
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

func SNS() *sns.SNS {
	return sns.New(awsConfig())
}

func buildTemplate(name, section string, input interface{}) (string, error) {
	data, err := Asset(fmt.Sprintf("templates/%s.tmpl", name))

	if err != nil {
		return "", err
	}

	tmpl, err := template.New(section).Funcs(templateHelpers()).Parse(string(data))

	if err != nil {
		return "", err
	}

	var formation bytes.Buffer

	err = tmpl.Execute(&formation, input)

	if err != nil {
		return "", err
	}

	return formation.String(), nil
}

// truncat a float to a given precision
// ex:  truncate(3.1459, 2) -> 3.14
func truncate(f float64, precision int) float64 {
	p := math.Pow10(precision)
	return float64(int(f*p)) / p
}
