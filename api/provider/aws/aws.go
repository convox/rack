package aws

import (
	"os"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sns"
)

var (
	CustomTopic  = os.Getenv("CUSTOM_TOPIC")
	SortableTime = "20060102.150405.000000000"
	ValidAppName = regexp.MustCompile(`\A[a-zA-Z][-a-zA-Z0-9]{3,29}\z`)
)

type AWSProvider struct {
	Region   string
	Access   string
	Secret   string
	Endpoint string
}

func NewProvider(region, access, secret, endpoint string) (*AWSProvider, error) {
	p := &AWSProvider{
		Region:   region,
		Access:   access,
		Secret:   secret,
		Endpoint: endpoint,
	}

	return p, nil
}

/** services ****************************************************************************************/

func (p *AWSProvider) config() *aws.Config {
	config := &aws.Config{
		Credentials: credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"), ""),
	}

	if e := os.Getenv("AWS_ENDPOINT"); e != "" {
		config.Endpoint = aws.String(e)
	}

	if r := os.Getenv("AWS_REGION"); r != "" {
		config.Region = aws.String(r)
	}

	if os.Getenv("DEBUG") != "" {
		config.WithLogLevel(aws.LogDebugWithHTTPBody)
	}

	return config
}

func (p *AWSProvider) acm() *acm.ACM {
	return acm.New(session.New(), p.config())
}

func (p *AWSProvider) cloudformation() *cloudformation.CloudFormation {
	return cloudformation.New(session.New(), p.config())
}

func (p *AWSProvider) cloudwatch() *cloudwatch.CloudWatch {
	return cloudwatch.New(session.New(), p.config())
}

func (p *AWSProvider) cloudwatchlogs() *cloudwatchlogs.CloudWatchLogs {
	return cloudwatchlogs.New(session.New(), p.config())
}

func (p *AWSProvider) dynamodb() *dynamodb.DynamoDB {
	return dynamodb.New(session.New(), p.config())
}

func (p *AWSProvider) ec2() *ec2.EC2 {
	return ec2.New(session.New(), p.config())
}

func (p *AWSProvider) ecr() *ecr.ECR {
	return ecr.New(session.New(), p.config())
}

func (p *AWSProvider) ecs() *ecs.ECS {
	return ecs.New(session.New(), p.config())
}

func (p *AWSProvider) iam() *iam.IAM {
	return iam.New(session.New(), p.config())
}

func (p *AWSProvider) kinesis() *kinesis.Kinesis {
	return kinesis.New(session.New(), p.config())
}

// s3 returns an S3 client configured to use the path style
// (http://s3.amazonaws.com/johnsmith.net/homepage.html) vs virtual
// hosted style (http://johnsmith.net.s3.amazonaws.com/homepage.html)
// since path style is easier to test.
func (p *AWSProvider) s3() *s3.S3 {
	return s3.New(session.New(), p.config().WithS3ForcePathStyle(true))
}

func (p *AWSProvider) sns() *sns.SNS {
	return sns.New(session.New(), p.config())
}
