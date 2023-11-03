package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
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
	"github.com/aws/aws-sdk-go/service/eventbridge"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/convox/logger"
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/metrics"
	"github.com/convox/rack/pkg/structs"
	"github.com/pkg/errors"
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
	AvailabilityZones   string
	BuildCluster        string
	ClientId            string
	CloudformationTopic string
	Cluster             string
	CustomEncryptionKey string
	Development         bool
	DynamoBuilds        string
	DynamoReleases      string
	DockerTLS           *structs.TLSPemCertBytes
	EcsPollInterval     int
	EncryptionKey       string
	Fargate             bool
	HighAvailability    bool
	Internal            bool
	InternalOnly        bool
	ELBLogBucket        string
	LogBucket           string
	LogDriver           string
	MaintainTimerState  bool
	NotificationTopic   string
	OnDemandMinCount    int
	Password            string
	Private             bool
	PrivateBuild        bool
	Rack                string
	RackApiServiceName  string
	SecurityGroup       string
	SettingsBucket      string
	SshKey              string
	SpotInstances       bool
	Subnets             string
	SubnetsPrivate      string
	StackId             string
	SyslogDestination   string
	SyslogFormat        string
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
	p.AvailabilityZones = labels["rack.AvailabilityZones"]
	p.BuildCluster = labels["rack.BuildCluster"]
	p.CloudformationTopic = labels["rack.CloudformationTopic"]
	p.Cluster = labels["rack.Cluster"]
	p.CustomEncryptionKey = labels["rack.CustomEncryptionKey"]
	p.DynamoBuilds = labels["rack.DynamoBuilds"]
	p.DynamoReleases = labels["rack.DynamoReleases"]
	p.EcsPollInterval = intParam(labels["rack.EcsPollInterval"], 1)
	p.EncryptionKey = labels["rack.EncryptionKey"]
	p.Fargate = labels["rack.Fargate"] == "Yes"
	p.HighAvailability = labels["rack.HighAvailability"] == "true"
	p.Internal = labels["rack.Internal"] == "Yes"
	p.InternalOnly = labels["rack.InternalOnly"] == "Yes"
	p.LogBucket = labels["rack.LogBucket"]
	p.LogDriver = labels["rack.LogDriver"]
	p.MaintainTimerState = labels["rack.MaintainTimerState"] == "Yes"
	p.NotificationTopic = labels["rack.NotificationTopic"]
	p.OnDemandMinCount = intParam(labels["rack.OnDemandMinCount"], 2)
	p.Private = labels["rack.Private"] == "Yes"
	p.PrivateBuild = labels["rack.PrivateBuild"] == "Yes"
	p.RackApiServiceName = strings.ReplaceAll(labels["rack.RackApiServiceName"], ",", "|")
	p.SecurityGroup = labels["rack.SecurityGroup"]
	p.SettingsBucket = labels["rack.SettingsBucket"]
	p.SpotInstances = labels["rack.SpotInstances"] == "Yes"
	p.SshKey = labels["rack.SshKey"]
	p.Subnets = labels["rack.Subnets"]
	p.SubnetsPrivate = labels["rack.SubnetsPrivate"]
	p.SyslogDestination = labels["rack.SyslogDestination"]
	p.SyslogFormat = labels["rack.SyslogFormat"]
	p.Version = labels["rack.Version"]
	p.Vpc = labels["rack.Vpc"]
	p.VpcCidr = labels["rack.VpcCidr"]

	if v, has := labels["rack.DockerTlsCA"]; has && len(v) > 0 {
		cacert, err := base64.StdEncoding.DecodeString(labels["rack.DockerTlsCA"])
		if err != nil {
			return err
		}
		cakey, err := base64.StdEncoding.DecodeString(labels["rack.DockerTlsCAKey"])
		if err != nil {
			return err
		}
		cert, err := base64.StdEncoding.DecodeString(labels["rack.DockerTlsCert"])
		if err != nil {
			return err
		}
		key, err := base64.StdEncoding.DecodeString(labels["rack.DockerTlsKey"])
		if err != nil {
			return err
		}

		p.DockerTLS = &structs.TLSPemCertBytes{
			CACert: cacert,
			CAKey:  cakey,
			Cert:   cert,
			Key:    key,
		}
	}
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

	s, err := helpers.NewSession()
	if err != nil {
		return errors.WithStack(err)
	}

	p.CloudWatch = cloudwatch.New(s, p.config())

	return nil
}

func (p *Provider) Context() context.Context {
	if p.ctx == nil {
		return context.Background()
	}

	return p.ctx
}

func (p *Provider) WithContext(ctx context.Context) structs.Provider {
	cp := *p
	cp.ctx = ctx
	return &cp
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
	s, err := helpers.NewSession()
	if err != nil {
		panic(errors.WithStack(err))
	}
	return acm.New(s, p.config())
}

func (p *Provider) autoscaling() *autoscaling.AutoScaling {
	s, err := helpers.NewSession()
	if err != nil {
		panic(errors.WithStack(err))
	}
	return autoscaling.New(s, p.config())
}

func (p *Provider) cloudformation() *cloudformation.CloudFormation {
	s, err := helpers.NewSession()
	if err != nil {
		panic(errors.WithStack(err))
	}
	return cloudformation.New(s, p.config())
}

func (p *Provider) cloudwatch() *cloudwatch.CloudWatch {
	s, err := helpers.NewSession()
	if err != nil {
		panic(errors.WithStack(err))
	}
	return cloudwatch.New(s, p.config())
}

func (p *Provider) cloudwatchlogs() *cloudwatchlogs.CloudWatchLogs {
	s, err := helpers.NewSession()
	if err != nil {
		panic(errors.WithStack(err))
	}
	return cloudwatchlogs.New(s, p.config().WithLogLevel(aws.LogOff))
}

func (p *Provider) dynamodb() *dynamodb.DynamoDB {
	s, err := helpers.NewSession()
	if err != nil {
		panic(errors.WithStack(err))
	}
	return dynamodb.New(s, p.config())
}

func (p *Provider) ec2() *ec2.EC2 {
	s, err := helpers.NewSession()
	if err != nil {
		panic(errors.WithStack(err))
	}
	return ec2.New(s, p.config())
}

func (p *Provider) ecr() *ecr.ECR {
	s, err := helpers.NewSession()
	if err != nil {
		panic(errors.WithStack(err))
	}
	return ecr.New(s, p.config())
}

func (p *Provider) ecs() *ecs.ECS {
	s, err := helpers.NewSession()
	if err != nil {
		panic(errors.WithStack(err))
	}
	return ecs.New(s, p.config())
}

func (p *Provider) eventbridge() *eventbridge.EventBridge {
	s, err := helpers.NewSession()
	if err != nil {
		panic(errors.WithStack(err))
	}
	return eventbridge.New(s, p.config())
}

func (p *Provider) kms() *kms.KMS {
	s, err := helpers.NewSession()
	if err != nil {
		panic(errors.WithStack(err))
	}
	return kms.New(s, p.config())
}

func (p *Provider) iam() *iam.IAM {
	s, err := helpers.NewSession()
	if err != nil {
		panic(errors.WithStack(err))
	}
	return iam.New(s, p.config())
}

func (p *Provider) s3() *s3.S3 {
	s, err := helpers.NewSession()
	if err != nil {
		panic(errors.WithStack(err))
	}
	return s3.New(s, p.config().WithS3ForcePathStyle(true))
}

func (p *Provider) sns() *sns.SNS {
	s, err := helpers.NewSession()
	if err != nil {
		panic(errors.WithStack(err))
	}
	return sns.New(s, p.config())
}

func (p *Provider) sqs() *sqs.SQS {
	s, err := helpers.NewSession()
	if err != nil {
		panic(errors.WithStack(err))
	}
	return sqs.New(s, p.config())
}

func (p *Provider) sts() *sts.STS {
	s, err := helpers.NewSession()
	if err != nil {
		panic(errors.WithStack(err))
	}
	return sts.New(s, p.config())
}

func (p *Provider) ssm() *ssm.SSM {
	s, err := helpers.NewSession()
	if err != nil {
		panic(errors.WithStack(err))
	}
	return ssm.New(s, p.config())
}

// IsTest returns true when we're in test mode
func (p *Provider) IsTest() bool {
	return p.Region == "us-test-1"
}
