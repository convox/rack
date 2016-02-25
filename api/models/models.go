package models

import (
	"bytes"
	"fmt"
	"html/template"
	"math"
	"os"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/session"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ec2"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecr"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/elb"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/iam"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/rds"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/s3"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/sns"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/sqs"
)

var SortableTime = "20060102.150405.000000000"

func awsError(err error) string {
	if ae, ok := err.(awserr.Error); ok {
		return ae.Code()
	}

	return ""
}

func awsConfig() *aws.Config {
	config := &aws.Config{
		Credentials: credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"), ""),
	}

	if e := os.Getenv("AWS_ENDPOINT"); e != "" {
		config.Endpoint = aws.String(e)
	}

	if r := os.Getenv("AWS_REGION"); r != "" {
		config.Region = aws.String(r)
	}

	return config
}

func AutoScaling() *autoscaling.AutoScaling {
	return autoscaling.New(session.New(), awsConfig())
}

func CloudFormation() *cloudformation.CloudFormation {
	return cloudformation.New(session.New(), awsConfig())
}

func CloudWatch() *cloudwatch.CloudWatch {
	return cloudwatch.New(session.New(), awsConfig())
}

func CloudWatchLogs() *cloudwatchlogs.CloudWatchLogs {
	return cloudwatchlogs.New(session.New(), awsConfig())
}

func DynamoDB() *dynamodb.DynamoDB {
	return dynamodb.New(session.New(), awsConfig())
}

func EC2() *ec2.EC2 {
	return ec2.New(session.New(), awsConfig())
}

func ECR() *ecr.ECR {
	return ecr.New(session.New(), awsConfig())
}

func ECS() *ecs.ECS {
	c := awsConfig()
	c.MaxRetries = aws.Int(10)
	return ecs.New(session.New(), c)
}

func ELB() *elb.ELB {
	return elb.New(session.New(), awsConfig())
}

func IAM() *iam.IAM {
	return iam.New(session.New(), awsConfig())
}

func Kinesis() *kinesis.Kinesis {
	return kinesis.New(session.New(), awsConfig())
}

func RDS() *rds.RDS {
	return rds.New(session.New(), awsConfig())
}

func S3() *s3.S3 {
	return s3.New(session.New(), awsConfig())
}

func SQS() *sqs.SQS {
	return sqs.New(session.New(), awsConfig())
}

func SNS() *sns.SNS {
	return sns.New(session.New(), awsConfig())
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
