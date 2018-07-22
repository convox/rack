package aws

import (
	"context"
	"math/rand"
	"os"
	"strconv"
	"strings"
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
	Development         bool
	DynamoBuilds        string
	DynamoReleases      string
	EcsPollInterval     int
	EncryptionKey       string
	Fargate             bool
	Internal            bool
	LogBucket           string
	NotificationTopic   string
	OnDemandMinCount    int
	Password            string
	Private             bool
	Rack                string
	SecurityGroup       string
	SettingsBucket      string
	SpotInstances       bool
	Subnets             string
	SubnetsPrivate      string
	Version             string
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

	params := stackParameters(rack)
	resources := map[string]string{}

	res, err := p.describeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(p.Rack),
	})
	if err != nil {
		return nil, err
	}

	for _, sr := range res.StackResources {
		resources[*sr.LogicalResourceId] = *sr.PhysicalResourceId
	}

	p.BuildCluster = coalesces(resources["BuildCluster"], resources["Cluster"])
	p.CloudformationTopic = resources["CloudformationTopic"]
	p.Cluster = resources["Cluster"]
	p.DynamoBuilds = resources["DynamoBuilds"]
	p.DynamoReleases = resources["DynamoReleases"]
	p.EcsPollInterval = intParam(params["EcsPollInterval"], 1)
	p.EncryptionKey = resources["EncryptionKey"]
	p.Fargate = params["Fargate"] == "Yes"
	p.Internal = params["Internal"] == "Yes"
	p.LogBucket = coalesces(params["LogBucket"], resources["Logs"])
	p.NotificationTopic = resources["NotificationTopic"]
	p.OnDemandMinCount = intParam(params["OnDemandMinCount"], 2)
	p.Private = params["Private"] == "Yes"
	p.SecurityGroup = coalesces(params["InstanceSecurityGroup"], resources["InstancesSecurity"])
	p.SettingsBucket = resources["Settings"]
	p.SpotInstances = params["SpotInstanceBid"] != ""
	p.Subnets = sliceParam(resources["Subnet0"], resources["Subnet1"], resources["Subnet2"])
	p.Subnets = sliceParam(resources["SubnetPrivate0"], resources["SubnetPrivate1"], resources["SubnetPrivate2"])
	p.Version = coalesces(os.Getenv("VERSION"), params["Version"])
	p.Vpc = coalesces(params["ExistingVpc"], resources["Vpc"])
	p.VpcCidr = params["VPCCIDR"]

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

func intParam(param string, def int) int {
	if param == "" {
		return def
	}
	v, err := strconv.Atoi(param)
	if err != nil {
		return def
	}
	return v
}

func sliceParam(param ...string) string {
	ss := []string{}

	for _, p := range param {
		if ps := strings.TrimSpace(p); ps != "" {
			ss = append(ss, ps)
		}
	}

	return strings.Join(ss, ",")
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
