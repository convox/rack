package aws

import (
	"math/rand"
	"os"
	"time"

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
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/convox/logger"
)

var (
	customTopic       = os.Getenv("CUSTOM_TOPIC")
	notificationTopic = os.Getenv("NOTIFICATION_TOPIC")
	sortableTime      = "20060102.150405.000000000"
)

// Logger is a package-wide logger
var Logger = logger.New("ns=provider.aws")

type AWSProvider struct {
	Region   string
	Endpoint string
	Access   string
	Secret   string
	Token    string

	Cluster           string
	Development       bool
	DockerImageAPI    string
	DynamoBuilds      string
	DynamoReleases    string
	NotificationHost  string
	NotificationTopic string
	Password          string
	Rack              string
	RegistryHost      string
	SettingsBucket    string
	Subnets           string
	SubnetsPrivate    string
	Vpc               string
	VpcCidr           string

	SkipCache bool
}

// NewProviderFromEnv returns a new AWS provider from env vars
func NewProviderFromEnv() *AWSProvider {
	return &AWSProvider{
		Region:            os.Getenv("AWS_REGION"),
		Endpoint:          os.Getenv("AWS_ENDPOINT"),
		Access:            os.Getenv("AWS_ACCESS"),
		Secret:            os.Getenv("AWS_SECRET"),
		Token:             os.Getenv("AWS_TOKEN"),
		Cluster:           os.Getenv("CLUSTER"),
		Development:       os.Getenv("DEVELOPMENT") == "true",
		DockerImageAPI:    os.Getenv("DOCKER_IMAGE_API"),
		DynamoBuilds:      os.Getenv("DYNAMO_BUILDS"),
		DynamoReleases:    os.Getenv("DYNAMO_RELEASES"),
		NotificationHost:  os.Getenv("NOTIFICATION_HOST"),
		NotificationTopic: os.Getenv("NOTIFICATION_TOPIC"),
		Password:          os.Getenv("PASSWORD"),
		Rack:              os.Getenv("RACK"),
		RegistryHost:      os.Getenv("REGISTRY_HOST"),
		SettingsBucket:    os.Getenv("SETTINGS_BUCKET"),
		Subnets:           os.Getenv("SUBNETS"),
		SubnetsPrivate:    os.Getenv("SUBNETS_PRIVATE"),
		Vpc:               os.Getenv("VPC"),
		VpcCidr:           os.Getenv("VPCCIDR"),
	}
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

/** services ****************************************************************************************/

func (p *AWSProvider) config() *aws.Config {
	config := &aws.Config{
		Credentials: credentials.NewStaticCredentials(p.Access, p.Secret, p.Token),
	}

	if p.Region != "" {
		config.Region = aws.String(p.Region)
	}

	if p.Endpoint != "" {
		config.Endpoint = aws.String(p.Endpoint)
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

// IsTest returns true when we're in test mode
func (p *AWSProvider) IsTest() bool {
	return p.Region == "us-test-1"
}
