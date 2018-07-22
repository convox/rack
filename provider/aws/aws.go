package aws

import (
	"context"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/convox/logger"
	"github.com/convox/rack/structs"
)

const (
	maxBuilds    = 50
	sortableTime = "20060102.150405.000000000"
)

// Logger is a package-wide logger
var Logger = logger.New("ns=provider.aws")

type AWSProvider struct {
	Region   string
	Endpoint string

	BuildCluster        string
	CloudformationTopic string
	Cluster             string
	CustomTopic         string
	Development         bool
	DynamoBuilds        string
	DynamoReleases      string
	EcsPollInterval     int
	EncryptionKey       string
	Fargate             bool
	Internal            bool
	LogBucket           string
	NotificationHost    string
	NotificationTopic   string
	OnDemandMinCount    int
	Password            string
	Private             bool
	Rack                string
	Release             string
	SecurityGroup       string
	SettingsBucket      string
	SpotInstances       bool
	Subnets             string
	SubnetsPrivate      string
	Vpc                 string
	VpcCidr             string

	SkipCache bool

	ctx context.Context
	log *logger.Logger
}

// NewProviderFromEnv returns a new AWS provider from env vars
func FromEnv() (*AWSProvider, error) {
	p := &AWSProvider{
		Development: os.Getenv("DEVELOPMENT") == "true",
		Password:    os.Getenv("PASSWORD"),
		Rack:        os.Getenv("RACK"),
		Region:      os.Getenv("AWS_REGION"),
		ctx:         context.Background(),
		log:         logger.New("ns=aws"),
	}

	if p.Rack == "" {
		return p, nil
	}

	rack, err := p.describeStack(p.Rack)
	if err != nil {
		return nil, err
	}

	outputs := stackOutputs(rack)

	p.BuildCluster = outputs["BuildCluster"]
	p.CloudformationTopic = outputs["CloudformationTopic"]
	p.Cluster = outputs["Cluster"]
	p.CustomTopic = outputs["CustomTopic"]
	p.DynamoBuilds = outputs["DynamoBuilds"]
	p.DynamoReleases = outputs["DynamoReleases"]
	p.EncryptionKey = outputs["EncryptionKey"]
	p.Fargate = outputs["Fargate"] == "Yes"
	p.Internal = outputs["Internal"] == "Yes"
	p.LogBucket = outputs["LogBucket"]
	p.NotificationHost = outputs["NotificationHost"]
	p.NotificationTopic = outputs["NotificationTopic"]
	p.Private = outputs["Private"] == "Yes"
	p.Release = outputs["Release"]
	p.SecurityGroup = outputs["SecurityGroup"]
	p.SettingsBucket = outputs["SettingsBucket"]
	p.SpotInstances = outputs["SpotInstances"] == "Yes"
	p.Subnets = outputs["Subnets"]
	p.SubnetsPrivate = outputs["SubnetsPrivate"]
	p.Vpc = outputs["Vpc"]
	p.VpcCidr = outputs["Vpccidr"]

	if v := os.Getenv("VERSION"); v != "" {
		p.Release = v
	}

	v, err := strconv.Atoi(outputs["EcsPollInterval"])
	if err != nil {
		return nil, err
	}
	p.EcsPollInterval = v

	v, err = strconv.Atoi(outputs["OnDemandMinCount"])
	if err != nil {
		return nil, err
	}
	p.OnDemandMinCount = v

	return p, nil
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func (p *AWSProvider) Initialize(opts structs.ProviderOptions) error {
	if opts.Logs != nil {
		Logger = logger.NewWriter("ns=aws", opts.Logs)
	}

	return nil
}

/** services ****************************************************************************************/

func (p *AWSProvider) config() *aws.Config {
	config := &aws.Config{
		Region: aws.String(p.Region),
	}

	if p.Endpoint != "" {
		config.Endpoint = aws.String(p.Endpoint)
	}

	if os.Getenv("DEBUG") == "true" {
		config.WithLogLevel(aws.LogDebugWithHTTPBody)
	}

	config.MaxRetries = aws.Int(7)

	return config
}

func (p *AWSProvider) logger(at string) *logger.Logger {
	log := p.log

	if id := p.ctx.Value("request.id"); id != nil {
		log = log.Prepend("id=%s", id)
	}

	return log.At(at).Start()
}

func (p *AWSProvider) acm() *acm.ACM {
	return acm.New(session.New(), p.config())
}

func (p *AWSProvider) autoscaling() *autoscaling.AutoScaling {
	return autoscaling.New(session.New(), p.config())
}

func (p *AWSProvider) cloudformation() *cloudformation.CloudFormation {
	return cloudformation.New(session.New(), p.config())
}

func (p *AWSProvider) cloudwatch() *cloudwatch.CloudWatch {
	return cloudwatch.New(session.New(), p.config())
}

func (p *AWSProvider) cloudwatchlogs() *cloudwatchlogs.CloudWatchLogs {
	return cloudwatchlogs.New(session.New(), p.config().WithLogLevel(aws.LogOff))
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

func (p *AWSProvider) kms() *kms.KMS {
	return kms.New(session.New(), p.config())
}

func (p *AWSProvider) iam() *iam.IAM {
	return iam.New(session.New(), p.config())
}

func (p *AWSProvider) s3() *s3.S3 {
	return s3.New(session.New(), p.config().WithS3ForcePathStyle(true))
}

func (p *AWSProvider) sns() *sns.SNS {
	return sns.New(session.New(), p.config())
}

func (p *AWSProvider) sqs() *sqs.SQS {
	return sqs.New(session.New(), p.config())
}

func (p *AWSProvider) sts() *sts.STS {
	return sts.New(session.New(), p.config())
}

// IsTest returns true when we're in test mode
func (p *AWSProvider) IsTest() bool {
	return p.Region == "us-test-1"
}
