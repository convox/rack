package aws

import (
	"context"
	"fmt"
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
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
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
	"github.com/convox/rack/pkg/metrics"
	"github.com/convox/rack/pkg/structs"
)

const (
	maxBuilds    = 50
	sortableTime = "20060102.150405.000000000"
)

// Logger is a package-wide logger
var Logger = logger.New("ns=provider.aws")

type Provider struct {
	Region   string
	Endpoint string

	AsgSpot             string
	AsgStandard         string
	BuildCluster        string
	ClientId            string
	CloudformationTopic string
	Cluster             string
	Development         bool
	DynamoBuilds        string
	DynamoReleases      string
	EcsPollInterval     int
	EncryptionKey       string
	Fargate             bool
	Internal            bool
	InternalOnly        bool
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
	StackId             string
	Version             string
	Vpc                 string
	VpcCidr             string

	Metrics   *metrics.Metrics
	SkipCache bool

	CloudWatch cloudwatchiface.CloudWatchAPI

	ctx context.Context
	log *logger.Logger
}

// NewProviderFromEnv returns a new AWS provider from env vars
func FromEnv() (*Provider, error) {
	p := &Provider{
		ClientId:    os.Getenv("CLIENT_ID"),
		Development: os.Getenv("DEVELOPMENT") == "true",
		Password:    os.Getenv("PASSWORD"),
		Rack:        os.Getenv("RACK"),
		Region:      os.Getenv("AWS_REGION"),
		StackId:     os.Getenv("STACK_ID"),
		Metrics:     metrics.New("https://metrics.convox.com/metrics/rack"),
		ctx:         context.Background(),
		log:         logger.New("ns=aws"),
	}

	if err := p.loadParams(); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *Provider) loadParams() error {
	if p.Rack == "" {
		return nil
	}

	td, err := p.stackResource(p.Rack, "ApiWebTasks")
	if err != nil {
		return err
	}

	res, err := p.ecs().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: td.PhysicalResourceId,
	})
	if err != nil {
		return err
	}

	if len(res.TaskDefinition.ContainerDefinitions) < 1 {
		return fmt.Errorf("invalid container definition")
	}

	cd := res.TaskDefinition.ContainerDefinitions[0]

	labels := map[string]string{}

	for k, v := range cd.DockerLabels {
		labels[k] = *v
	}

	p.AsgSpot = labels["rack.AsgSpot"]
	p.AsgStandard = labels["rack.AsgStandard"]
	p.BuildCluster = labels["rack.BuildCluster"]
	p.CloudformationTopic = labels["rack.CloudformationTopic"]
	p.Cluster = labels["rack.Cluster"]
	p.DynamoBuilds = labels["rack.DynamoBuilds"]
	p.DynamoReleases = labels["rack.DynamoReleases"]
	p.EcsPollInterval = intParam(labels["rack.EcsPollInterval"], 1)
	p.EncryptionKey = labels["rack.EncryptionKey"]
	p.Fargate = labels["rack.Fargate"] == "Yes"
	p.Internal = labels["rack.Internal"] == "Yes"
	p.InternalOnly = labels["rack.InternalOnly"] == "Yes"
	p.LogBucket = labels["rack.LogBucket"]
	p.NotificationTopic = labels["rack.NotificationTopic"]
	p.OnDemandMinCount = intParam(labels["rack.OnDemandMinCount"], 2)
	p.Private = labels["Private"] == "Yes"
	p.SecurityGroup = labels["rack.SecurityGroup"]
	p.SettingsBucket = labels["rack.SettingsBucket"]
	p.SpotInstances = labels["rack.SpotInstances"] == "Yes"
	p.Subnets = labels["rack.Subnets"]
	p.SubnetsPrivate = labels["rack.SubnetsPrivate"]
	p.Version = labels["rack.Version"]
	p.Vpc = labels["rack.Vpc"]
	p.VpcCidr = labels["rack.VpcCidr"]

	return nil
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func (p *Provider) Initialize(opts structs.ProviderOptions) error {
	if opts.Logs != nil {
		Logger = logger.NewWriter("ns=aws", opts.Logs)
	}

	if p.Development {
		go p.Workers()
	}

	p.CloudWatch = cloudwatch.New(session.New(), p.config())

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

func (p *Provider) config() *aws.Config {
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

func (p *Provider) logger(at string) *logger.Logger {
	log := p.log

	if id := p.ctx.Value("request.id"); id != nil {
		log = log.Prepend("id=%s", id)
	}

	return log.At(at).Start()
}

func (p *Provider) acm() *acm.ACM {
	return acm.New(session.New(), p.config())
}

func (p *Provider) autoscaling() *autoscaling.AutoScaling {
	return autoscaling.New(session.New(), p.config())
}

func (p *Provider) cloudformation() *cloudformation.CloudFormation {
	return cloudformation.New(session.New(), p.config())
}

func (p *Provider) cloudwatch() *cloudwatch.CloudWatch {
	return cloudwatch.New(session.New(), p.config())
}

func (p *Provider) cloudwatchlogs() *cloudwatchlogs.CloudWatchLogs {
	return cloudwatchlogs.New(session.New(), p.config().WithLogLevel(aws.LogOff))
}

func (p *Provider) dynamodb() *dynamodb.DynamoDB {
	return dynamodb.New(session.New(), p.config())
}

func (p *Provider) ec2() *ec2.EC2 {
	return ec2.New(session.New(), p.config())
}

func (p *Provider) ecr() *ecr.ECR {
	return ecr.New(session.New(), p.config())
}

func (p *Provider) ecs() *ecs.ECS {
	return ecs.New(session.New(), p.config())
}

func (p *Provider) kms() *kms.KMS {
	return kms.New(session.New(), p.config())
}

func (p *Provider) iam() *iam.IAM {
	return iam.New(session.New(), p.config())
}

func (p *Provider) s3() *s3.S3 {
	return s3.New(session.New(), p.config().WithS3ForcePathStyle(true))
}

func (p *Provider) sns() *sns.SNS {
	return sns.New(session.New(), p.config())
}

func (p *Provider) sqs() *sqs.SQS {
	return sqs.New(session.New(), p.config())
}

func (p *Provider) sts() *sts.STS {
	return sts.New(session.New(), p.config())
}

// IsTest returns true when we're in test mode
func (p *Provider) IsTest() bool {
	return p.Region == "us-test-1"
}
