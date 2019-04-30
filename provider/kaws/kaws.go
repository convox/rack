package kaws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecr/ecriface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/pkg/templater"
	"github.com/convox/rack/provider/k8s"
	"github.com/gobuffalo/packr"
	"k8s.io/apimachinery/pkg/util/runtime"
)

type Provider struct {
	*k8s.Provider

	AccountId      string
	AdminUser      string
	AutoscalerRole string
	// BalancerSecurity     string
	BaseDomain    string
	Bucket        string
	Cluster       string
	EventQueue    string
	EventTopic    string
	NodesRole     string
	Region        string
	RackRole      string
	RouterCache   string
	RouterHosts   string
	RouterRole    string
	RouterTargets string
	// RouterTargetGroup80  string
	// RouterTargetGroup443 string
	StackId string
	// SubnetsPublic        []string
	// SubnetsPrivate       []string
	// Vpc                  string

	CloudFormation cloudformationiface.CloudFormationAPI
	CloudWatchLogs cloudwatchlogsiface.CloudWatchLogsAPI
	ECR            ecriface.ECRAPI
	S3             s3iface.S3API
	SQS            sqsiface.SQSAPI

	templater *templater.Templater
}

func FromEnv() (*Provider, error) {
	kp, err := k8s.FromEnv()
	if err != nil {
		return nil, err
	}

	p := &Provider{
		Provider:   kp,
		Region:     os.Getenv("AWS_REGION"),
		BaseDomain: "example.org",
	}

	p.templater = templater.New(packr.NewBox("../kaws/template"), p.templateHelpers())

	kp.Engine = p

	return p, nil
}

func (p *Provider) Initialize(opts structs.ProviderOptions) error {
	if err := p.initializeAwsServices(); err != nil {
		return err
	}

	os, err := p.stackOutputs(p.Rack)
	if err != nil {
		return err
	}

	p.applyOutputs(os)

	if err := p.initializeIamRoles(); err != nil {
		return err
	}

	if err := p.Provider.Initialize(opts); err != nil {
		return err
	}

	runtime.ErrorHandlers = []func(error){}

	// nc, err := NewNodeController(p)
	// if err != nil {
	//   return err
	// }

	stc, err := NewStackController(p)
	if err != nil {
		return err
	}

	// go nc.Run()
	go stc.Run()

	go p.workerEvents()

	return nil
}

func (p *Provider) WithContext(ctx context.Context) structs.Provider {
	pp := *p
	pp.Provider = pp.Provider.WithContext(ctx).(*k8s.Provider)
	return &pp
}

func (p *Provider) applyOutputs(outputs map[string]string) {
	p.Provider.Socket = "/var/run/docker.sock"
	p.Provider.Version = outputs["Version"]

	p.AccountId = outputs["AccountId"]
	p.AdminUser = outputs["AdminUser"]
	p.AutoscalerRole = outputs["AutoscalerRole"]
	// p.BalancerSecurity = outputs["BalancerSecurity"]
	p.BaseDomain = outputs["BaseDomain"]
	p.Bucket = outputs["RackBucket"]
	p.Cluster = outputs["Cluster"]
	p.EventQueue = outputs["EventQueue"]
	p.EventTopic = outputs["EventTopic"]
	p.NodesRole = outputs["NodesRole"]
	p.RackRole = outputs["RackRole"]
	p.RouterCache = outputs["RouterCache"]
	p.RouterHosts = outputs["RouterHosts"]
	p.RouterRole = outputs["RouterRole"]
	p.RouterTargets = outputs["RouterTargets"]
	// p.RouterTargetGroup80 = outputs["RouterTargetGroup80"]
	// p.RouterTargetGroup443 = outputs["RouterTargetGroup443"]
	p.StackId = outputs["StackId"]
	// p.SubnetsPublic = strings.Split(outputs["VpcPublicSubnets"], ",")
	// p.SubnetsPrivate = strings.Split(outputs["VpcPrivateSubnets"], ",")
	// p.Vpc = outputs["Vpc"]
}

func (p *Provider) initializeAwsServices() error {
	s, err := session.NewSession()
	if err != nil {
		return err
	}

	p.CloudFormation = cloudformation.New(s)
	p.CloudWatchLogs = cloudwatchlogs.New(s)
	p.ECR = ecr.New(s)
	p.S3 = s3.New(s)
	p.SQS = sqs.New(s)

	return nil
}

func (p *Provider) initializeIamRoles() error {
	if err := kubectl("patch", "deployment/api", "-n", p.Rack, "-p", fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"iam.amazonaws.com/role":%q}}}}}`, p.RackRole)); err != nil {
		return err
	}

	if err := kubectl("patch", "deployment/router", "-n", "convox-system", "-p", fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"iam.amazonaws.com/role":%q}}}}}`, p.RouterRole)); err != nil {
		return err
	}

	return nil
}
