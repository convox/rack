package models

import (
	"bytes"
	"fmt"
	"html/template"
	"math"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/convox/logger"
)

// Logger is a package-wide logger
var Logger = logger.New("ns=api.models")

var SortableTime = "20060102.150405.000000000"

func awsError(err error) string {
	if ae, ok := err.(awserr.Error); ok {
		return ae.Code()
	}

	return ""
}

func awsConfig() *aws.Config {
	config := &aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	}

	if e := os.Getenv("AWS_ENDPOINT"); e != "" {
		config.Endpoint = aws.String(e)
	}

	if os.Getenv("DEBUG") == "true" {
		config.WithLogLevel(aws.LogDebugWithHTTPBody)
	}

	return config
}

func ACM() *acm.ACM {
	return acm.New(session.New(), awsConfig())
}

func AutoScaling() *autoscaling.AutoScaling {
	return autoscaling.New(session.New(), awsConfig())
}

func CloudFormation() *cloudformation.CloudFormation {
	return cloudformation.New(session.New(), awsConfig())
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

func IAM() *iam.IAM {
	return iam.New(session.New(), awsConfig())
}

func S3() *s3.S3 {
	return s3.New(session.New(), awsConfig())
}

func SNS() *sns.SNS {
	return sns.New(session.New(), awsConfig())
}

func SQS() *sqs.SQS {
	return sqs.New(session.New(), awsConfig())
}

func STS() *sts.STS {
	return sts.New(session.New(), awsConfig())
}

func assetTemplate(name, section string, input interface{}) (string, error) {
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
func truncate(f float64, precision int) float64 {
	p := math.Pow10(precision)
	return float64(int(f*p)) / p
}
