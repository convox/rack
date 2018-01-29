package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/version"
	"gopkg.in/urfave/cli.v1"
)

type AwsCredentials struct {
	Access     string `json:"AccessKeyId"`
	Secret     string `json:"SecretAccessKey"`
	Session    string `json:"SessionToken"`
	Expiration time.Time
}

var Banner = `

     ___    ___     ___   __  __    ___   __  _
    / ___\ / __ \ /  _  \/\ \/\ \  / __ \/\ \/ \
   /\ \__//\ \_\ \/\ \/\ \ \ \_/ |/\ \_\ \/>  </
   \ \____\ \____/\ \_\ \_\ \___/ \ \____//\_/\_\
    \/____/\/___/  \/_/\/_/\/__/   \/___/ \//\/_/

`

// CredentialsMessage is displayed to the user when no AWS credentials have been found.
const CredentialsMessage = `This installer needs AWS credentials to install/uninstall the Convox platform into
your AWS account. These credentials will only be used to communicate between this
installer running on your computer and the AWS API.

We recommend that you create a new set of credentials exclusively for this
install/uninstall process and then delete them once the installer has completed.`

var (
	distinctID         = ""
	formationURL       = "https://convox.s3.amazonaws.com/release/%s/formation.json"
	defaultSubnetCIDRs = "10.0.1.0/24,10.0.2.0/24,10.0.3.0/24"
	defaultVPCCIDR     = "10.0.0.0/16"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	stdcli.RegisterCommand(cli.Command{
		Name:        "install",
		Description: "install convox into an aws account",
		Usage:       "[credentials.csv]",
		Action:      cmdInstall,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "email",
				EnvVar: "CONVOX_EMAIL",
				Usage:  "email address to receive project updates",
			},
			cli.StringFlag{
				Name:   "password",
				EnvVar: "PASSWORD",
				Value:  "",
				Usage:  "custom rack password",
			},
			cli.StringFlag{
				Name:  "ami",
				Value: "",
				Usage: "custom AMI for rack instances",
			},
			cli.StringFlag{
				Name:   "build-instance",
				Value:  "",
				Usage:  "instance type for a dedicated build cluster",
				EnvVar: "RACK_BUILD_INSTANCE",
			},
			cli.BoolFlag{
				Name:  "dedicated",
				Usage: "create EC2 instances on dedicated hardware",
			},
			cli.StringFlag{
				Name:  "existing-vpc",
				Value: "",
				Usage: "existing vpc id into which to install rack",
			},
			cli.StringFlag{
				Name:  "http-proxy",
				Value: "",
				Usage: "an HTTP proxy URL that the rack should use for all outbound requests",
			},
			cli.IntFlag{
				Name:  "instance-count",
				Value: 3,
				Usage: "number of instances in the rack",
			},
			cli.StringFlag{
				Name:  "instance-type",
				Value: "t2.small",
				Usage: "type of instances in the rack",
			},
			cli.StringFlag{
				Name:  "internet-gateway",
				Value: "",
				Usage: "internet gateway id to use in existing vpc",
			},
			cli.BoolFlag{
				Name:  "no-autoscale",
				Usage: "use to disable autoscale during install (which is enabled by default)",
			},
			cli.BoolFlag{
				Name:   "private",
				Usage:  "use private subnets and NAT gateways to shield instances",
				EnvVar: "RACK_PRIVATE",
			},
			cli.StringFlag{
				Name:  "private-cidrs",
				Value: "10.0.4.0/24,10.0.5.0/24,10.0.6.0/24",
				Usage: "private subnet CIDRs",
			},
			cli.StringFlag{
				Name:   "region",
				Value:  "us-east-1",
				Usage:  "aws region",
				EnvVar: "AWS_REGION,AWS_DEFAULT_REGION",
			},
			cli.StringFlag{
				Name:   "stack-name",
				EnvVar: "STACK_NAME",
				Value:  "convox",
				Usage:  "custom rack name",
			},
			cli.StringFlag{
				Name:   "version",
				EnvVar: "VERSION",
				Value:  "latest",
				Usage:  "install a specific version",
			},
			cli.StringFlag{
				Name:  "vpc-cidr",
				Value: defaultVPCCIDR,
				Usage: "custom VPC CIDR",
			},
			cli.StringFlag{
				Name:  "subnet-cidrs",
				Value: defaultSubnetCIDRs,
				Usage: "subnet CIDRs",
			},
		},
	})
}

func cmdInstall(c *cli.Context) error {
	ep := stdcli.QOSEventProperties{Start: time.Now()}

	var err error

	distinctID, err = currentId()
	if err != nil {
		stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: fmt.Errorf("error getting versions: %s", err)})
		return stdcli.Error(err)
	}

	region := c.String("region")

	stackName := c.String("stack-name")
	awsRegexRules := []string{
		//ecr: http://docs.aws.amazon.com/AmazonECR/latest/APIReference/API_CreateRepository.html
		"(?:[a-z0-9]+(?:[._-][a-z0-9]+)*/)*[a-z0-9]+(?:[._-][a-z0-9]+)*",
		//cloud formation: https://forums.aws.amazon.com/thread.jspa?threadID=118427
		"[a-zA-Z][-a-zA-Z0-9]*",
	}

	for _, r := range awsRegexRules {
		rp := regexp.MustCompile(r)
		matchedStr := rp.FindString(stackName)
		match := len(matchedStr) == len(stackName)

		if !match {
			msg := fmt.Errorf("stack name '%s' is invalid, must match [a-z0-9-]*", stackName)
			stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{ValidationError: msg})
			return stdcli.Error(msg)
		}
	}

	tenancy := "default"
	instanceType := c.String("instance-type")

	if c.Bool("dedicated") {
		tenancy = "dedicated"
		if strings.HasPrefix(instanceType, "t2") {
			stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{ValidationError: fmt.Errorf("t2 instance types aren't supported in dedicated tenancy, please set --instance-type.")})
			return stdcli.Error(fmt.Errorf("t2 instance types aren't supported in dedicated tenancy, please set --instance-type."))
		}
	}

	numInstances := c.Int("instance-count")
	instanceCount := fmt.Sprintf("%d", numInstances)
	if numInstances <= 2 {
		stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{ValidationError: fmt.Errorf("instance-count must be greater than 2")})
		return stdcli.Error(fmt.Errorf("instance-count must be greater than 2"))
	}

	var subnet0CIDR, subnet1CIDR, subnet2CIDR string

	if cidrs := c.String("subnet-cidrs"); cidrs != "" {
		parts := strings.SplitN(cidrs, ",", 3)
		if len(parts) < 3 {
			stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{ValidationError: fmt.Errorf("subnet-cidrs must have 3 values")})
			return stdcli.Error(fmt.Errorf("subnet-cidrs must have 3 values"))
		}

		subnet0CIDR = parts[0]
		subnet1CIDR = parts[1]
		subnet2CIDR = parts[2]
	}

	var subnetPrivate0CIDR, subnetPrivate1CIDR, subnetPrivate2CIDR string

	if cidrs := c.String("private-cidrs"); cidrs != "" {
		parts := strings.SplitN(cidrs, ",", 3)
		if len(parts) < 3 {
			stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{ValidationError: fmt.Errorf("private-cidrs must have 3 values")})
			return stdcli.Error(fmt.Errorf("private-cidrs must have 3 values"))
		}

		subnetPrivate0CIDR = parts[0]
		subnetPrivate1CIDR = parts[1]
		subnetPrivate2CIDR = parts[2]
	}

	internetGateway := c.String("internet-gateway")

	vpcCIDR := c.String("vpc-cidr")

	var existingVPC string

	if vpc := c.String("existing-vpc"); vpc != "" {
		existingVPC = vpc
		// check required flags when installing into existing VPC
		if c.String("subnet-cidrs") == defaultSubnetCIDRs {
			stdcli.Warn(fmt.Sprintf("[existing vpc] using default subnet cidrs (%s); if this is incorrect, pass a custom value to --subnet-cidrs", defaultSubnetCIDRs))
		}

		if vpcCIDR == defaultVPCCIDR {
			stdcli.Warn(fmt.Sprintf("[existing vpc] using default vpc cidr (%s); if this is incorrect, pass a custom value to --vpc-cidr", defaultVPCCIDR))
		}

		if internetGateway == "" {
			return stdcli.Error(fmt.Errorf("must specify --internet-gateway for existing VPC"))
		}
	}

	private := "No"
	if c.Bool("private") {
		private = "Yes"
	}

	ami := c.String("ami")

	key := c.String("key")

	versions, err := version.All()
	if err != nil {
		stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: fmt.Errorf("error getting versions: %s", err)})
		return stdcli.Error(err)
	}

	version, err := versions.Resolve(c.String("version"))
	if err != nil {
		stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: fmt.Errorf("error resolving version: %s", err)})
		return stdcli.Error(err)
	}

	versionName := version.Version
	furl := fmt.Sprintf(formationURL, versionName)

	fmt.Println(Banner)

	if err != nil {
		stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: err})
		return stdcli.Error(err)
	}

	fmt.Printf("Installing Convox (%s)...\n", versionName)

	if private == "Yes" {
		fmt.Println("(Private Network Edition)")
	}

	reader := bufio.NewReader(os.Stdin)

	if email := c.String("email"); email != "" {
		distinctID = email
		updateID(distinctID)
	} else if distinctID != "" {
		// already has an id
	} else if terminal.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Print("Email Address (optional, to receive project updates): ")

		email, err := reader.ReadString('\n')
		if err != nil {
			stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: err})
			return stdcli.Error(err)
		}

		if strings.TrimSpace(email) != "" {
			distinctID = email
			updateID(email)
		}
	}

	credentialsFile := ""
	if len(c.Args()) >= 1 {
		credentialsFile = c.Args()[0]
	}

	creds, err := readCredentials(credentialsFile)
	if err != nil {
		stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: fmt.Errorf("error: %s", err)})
		return stdcli.Error(err)
	}
	if creds == nil {
		stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: fmt.Errorf("error reading credentials")})
		return stdcli.Error(err)
	}

	fmt.Println("Using AWS Access Key ID:", creds.Access)

	err = validateUserAccess(region, creds)
	if err != nil {
		stdcli.Error(err)
	}

	password := c.String("password")
	if password == "" {
		password = randomString(30)
	}

	httpProxy := c.String("http-proxy")

	CloudFormation := cloudformation.New(session.New(), awsConfig(region, creds))

	req := &cloudformation.CreateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		Parameters: []*cloudformation.Parameter{
			{ParameterKey: aws.String("Ami"), ParameterValue: aws.String(ami)},
			{ParameterKey: aws.String("ClientId"), ParameterValue: aws.String(distinctID)},
			{ParameterKey: aws.String("ExistingVpc"), ParameterValue: aws.String(existingVPC)},
			{ParameterKey: aws.String("HttpProxy"), ParameterValue: aws.String(httpProxy)},
			{ParameterKey: aws.String("InstanceCount"), ParameterValue: aws.String(instanceCount)},
			{ParameterKey: aws.String("InstanceType"), ParameterValue: aws.String(instanceType)},
			{ParameterKey: aws.String("InternetGateway"), ParameterValue: aws.String(internetGateway)},
			{ParameterKey: aws.String("Key"), ParameterValue: aws.String(key)},
			{ParameterKey: aws.String("Password"), ParameterValue: aws.String(password)},
			{ParameterKey: aws.String("Private"), ParameterValue: aws.String(private)},
			{ParameterKey: aws.String("Tenancy"), ParameterValue: aws.String(tenancy)},
			{ParameterKey: aws.String("Version"), ParameterValue: aws.String(versionName)},
			{ParameterKey: aws.String("Subnet0CIDR"), ParameterValue: aws.String(subnet0CIDR)},
			{ParameterKey: aws.String("Subnet1CIDR"), ParameterValue: aws.String(subnet1CIDR)},
			{ParameterKey: aws.String("Subnet2CIDR"), ParameterValue: aws.String(subnet2CIDR)},
			{ParameterKey: aws.String("SubnetPrivate0CIDR"), ParameterValue: aws.String(subnetPrivate0CIDR)},
			{ParameterKey: aws.String("SubnetPrivate1CIDR"), ParameterValue: aws.String(subnetPrivate1CIDR)},
			{ParameterKey: aws.String("SubnetPrivate2CIDR"), ParameterValue: aws.String(subnetPrivate2CIDR)},
			{ParameterKey: aws.String("VPCCIDR"), ParameterValue: aws.String(vpcCIDR)},
		},
		StackName:   aws.String(stackName),
		TemplateURL: aws.String(furl),
	}

	if c.Bool("no-autoscale") {
		p := &cloudformation.Parameter{
			ParameterKey:   aws.String("Autoscale"),
			ParameterValue: aws.String("No"),
		}
		req.Parameters = append(req.Parameters, p)
	}

	if c.String("build-instance") != "" {
		p := &cloudformation.Parameter{
			ParameterKey:   aws.String("BuildInstance"),
			ParameterValue: aws.String(c.String("build-instance")),
		}
		req.Parameters = append(req.Parameters, p)
	}

	if tf := os.Getenv("TEMPLATE_FILE"); tf != "" {
		dat, err := ioutil.ReadFile(tf)
		if err != nil {
			stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: fmt.Errorf("error reading template file: %s", tf)})
			return stdcli.Error(err)
		}

		t := new(bytes.Buffer)
		if err := json.Compact(t, dat); err != nil {
			stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: err})
			return stdcli.Error(err)
		}

		req.TemplateURL = nil
		req.TemplateBody = aws.String(t.String())
	}

	res, err := CloudFormation.CreateStack(req)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "AlreadyExistsException" {
				stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: err})
				return stdcli.Error(fmt.Errorf("Stack %q already exists. Run `convox uninstall` then try again", stackName))
			}
		}

		stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: err})
		return stdcli.Error(err)
	}

	// NOTE: we start making lots of network requests here
	//			 so we're just going to return for testability
	if os.Getenv("AWS_REGION") == "test" {
		fmt.Println(*res.StackId)
		return nil
	}

	host, err := waitForCompletion(*res.StackId, CloudFormation, false)
	if err != nil {
		stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: err})
		return stdcli.Error(err)
	}

	fmt.Printf("Waiting for load balancer...")

	if err := waitForAvailability(fmt.Sprintf("https://%s/", host)); err != nil {
		stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: err})
		return stdcli.Error(err)
	}

	err = addLogin(host, password)
	if err != nil {
		stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: err})
		return stdcli.Error(err)
	}

	fmt.Println("")
	addRackToConsoleMsg(c, host, password)

	return stdcli.QOSEventSend("cli-install", distinctID, ep)
}

func addRackToConsoleMsg(c *cli.Context, host string, password string) {
	fmt.Println("** Success! **")
	fmt.Println("Your Rack has been installed. You should now add it as an existing Rack at console.convox.com (or your own private Console) with the following credentials (which have also been written to ~/.convox/auth):")
	fmt.Printf(" 🢂  Hostname: %s\n", host)
	fmt.Printf(" 🢂   API Key: %s\n", password)
}

/// validateUserAccess checks for the "AdministratorAccess" policy needed to create a rack.
func validateUserAccess(region string, creds *AwsCredentials) error {

	// TODO: this validation needs to actually check permissions
	return nil
}

func awsConfig(region string, creds *AwsCredentials) *aws.Config {
	config := &aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(creds.Access, creds.Secret, creds.Session),
	}

	if e := os.Getenv("AWS_ENDPOINT"); e != "" {
		config.Endpoint = aws.String(e)
	}

	return config
}

func waitForCompletion(stack string, CloudFormation *cloudformation.CloudFormation, isDeleting bool) (string, error) {
	for {
		dres, err := CloudFormation.DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: aws.String(stack),
		})
		if err != nil {
			return "", err
		}

		err = displayProgress(stack, CloudFormation, isDeleting)
		if err != nil {
			return "", err
		}

		if len(dres.Stacks) != 1 {
			return "", fmt.Errorf("could not read stack status")
		}

		switch *dres.Stacks[0].StackStatus {
		case "CREATE_COMPLETE":
			for _, o := range dres.Stacks[0].Outputs {
				if *o.OutputKey == "Dashboard" {
					return *o.OutputValue, nil
				}
			}

			stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: err})
			return "", fmt.Errorf("could not install stack, contact support@convox.com for assistance")
		case "CREATE_FAILED":
			stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: err})
			return "", fmt.Errorf("stack creation failed, contact support@convox.com for assistance")
		case "ROLLBACK_COMPLETE":
			stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: err})
			return "", fmt.Errorf("stack creation failed, contact support@convox.com for assistance")
		case "DELETE_COMPLETE":
			stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: err})
			return "", nil
		case "DELETE_FAILED":
			stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: err})
			return "", fmt.Errorf("stack deletion failed, contact support@convox.com for assistance")
		}

		time.Sleep(2 * time.Second)
	}
}

var events = map[string]bool{}

func displayProgress(stack string, CloudFormation *cloudformation.CloudFormation, isDeleting bool) error {
	res, err := CloudFormation.DescribeStackEvents(&cloudformation.DescribeStackEventsInput{
		StackName: aws.String(stack),
	})

	if err != nil {
		return err
	}

	for _, event := range res.StackEvents {
		if events[*event.EventId] {
			continue
		}

		events[*event.EventId] = true

		// Log all CREATE_FAILED to display
		if !isDeleting && *event.ResourceStatus == "CREATE_FAILED" {
			msg := fmt.Sprintf("Failed %s: %s", *event.ResourceType, *event.ResourceStatusReason)
			fmt.Println(msg)
		}

		name := FriendlyName(*event.ResourceType)

		if name == "" {
			continue
		}

		switch *event.ResourceStatus {
		case "CREATE_IN_PROGRESS":
		case "CREATE_COMPLETE":
			if !isDeleting {
				id := *event.PhysicalResourceId

				if strings.HasPrefix(id, "arn:") {
					id = *event.LogicalResourceId
				}

				fmt.Printf("Created %s: %s\n", name, id)
			}
		case "CREATE_FAILED":
		case "DELETE_IN_PROGRESS":
		case "DELETE_COMPLETE":
			id := *event.PhysicalResourceId

			if strings.HasPrefix(id, "arn:") {
				id = *event.LogicalResourceId
			}

			fmt.Printf("Deleted %s: %s\n", name, id)
		case "DELETE_SKIPPED":
			id := *event.PhysicalResourceId

			if strings.HasPrefix(id, "arn:") {
				id = *event.LogicalResourceId
			}

			fmt.Printf("Skipped %s: %s\n", name, id)
		case "DELETE_FAILED":
			id := *event.PhysicalResourceId

			if strings.HasPrefix(id, "arn:") {
				id = *event.LogicalResourceId
			}

			fmt.Printf("Failed to delete %s: %s\n", name, id)
		case "ROLLBACK_IN_PROGRESS", "ROLLBACK_COMPLETE":
		case "UPDATE_IN_PROGRESS", "UPDATE_COMPLETE", "UPDATE_COMPLETE_CLEANUP_IN_PROGRESS", "UPDATE_FAILED", "UPDATE_ROLLBACK_IN_PROGRESS", "UPDATE_ROLLBACK_COMPLETE", "UPDATE_ROLLBACK_FAILED":
		default:
			return fmt.Errorf("Unhandled status: %s\n", *event.ResourceStatus)
		}
	}

	return nil
}

// FriendlyName turns an AWS resource type into a friendly name
func FriendlyName(t string) string {
	switch t {
	case "AWS::AutoScaling::AutoScalingGroup":
		return "AutoScalingGroup"
	case "AWS::AutoScaling::LaunchConfiguration":
		return ""
	case "AWS::AutoScaling::LifecycleHook":
		return ""
	case "AWS::CertificateManager::Certificate":
		return "SSL Certificate"
	case "AWS::CloudFormation::Stack":
		return "CloudFormation Stack"
	case "AWS::DynamoDB::Table":
		return "DynamoDB Table"
	case "AWS::EC2::EIP":
		return "NAT Elastic IP"
	case "AWS::EC2::InternetGateway":
		return "VPC Internet Gateway"
	case "AWS::EC2::NatGateway":
		return "NAT Gateway"
	case "AWS::EC2::Route":
		return ""
	case "AWS::EC2::RouteTable":
		return "Routing Table"
	case "AWS::EC2::SecurityGroup":
		return "Security Group"
	case "AWS::EC2::Subnet":
		return "VPC Subnet"
	case "AWS::EC2::SubnetRouteTableAssociation":
		return ""
	case "AWS::EC2::VPC":
		return "VPC"
	case "AWS::EC2::VPCGatewayAttachment":
		return ""
	case "AWS::ECS::Cluster":
		return "ECS Cluster"
	case "AWS::ECS::Service":
		return "ECS Service"
	case "AWS::ECS::TaskDefinition":
		return "ECS TaskDefinition"
	case "AWS::EFS::FileSystem":
		return "EFS Filesystem"
	case "AWS::EFS::MountTarget":
		return ""
	case "AWS::ElasticLoadBalancing::LoadBalancer":
		return "Elastic Load Balancer"
	case "AWS::ElasticLoadBalancingV2::Listener":
		return ""
	case "AWS::ElasticLoadBalancingV2::LoadBalancer":
		return "Application Load Balancer"
	case "AWS::ElasticLoadBalancingV2::TargetGroup":
		return ""
	case "AWS::Events::Rule":
		return ""
	case "AWS::IAM::AccessKey":
		return "Access Key"
	case "AWS::IAM::InstanceProfile":
		return ""
	case "AWS::IAM::ManagedPolicy":
		return "IAM Managed Policy"
	case "AWS::IAM::Role":
		return ""
	case "AWS::IAM::User":
		return "IAM User"
	case "AWS::Kinesis::Stream":
		return "Kinesis Stream"
	case "AWS::KMS::Alias":
		return "KMS Alias"
	case "AWS::Lambda::Function":
		return "Lambda Function"
	case "AWS::Lambda::Permission":
		return ""
	case "AWS::Logs::LogGroup":
		return "CloudWatch Log Group"
	case "AWS::Logs::SubscriptionFilter":
		return ""
	case "AWS::Route53::HostedZone":
		return "Hosted Zone"
	case "AWS::Route53::RecordSet":
		return ""
	case "AWS::S3::Bucket":
		return "S3 Bucket"
	case "AWS::S3::BucketPolicy":
		return "S3 Bucket Policy"
	case "AWS::SNS::Topic":
		return ""
	case "AWS::SQS::Queue":
		return "SQS Queue"
	case "AWS::SQS::QueuePolicy":
		return ""
	case "Custom::EC2AvailabilityZones":
		return ""
	case "Custom::ECSService":
		return "ECS Service"
	case "Custom::ECSTaskDefinition":
		return "ECS TaskDefinition"
	case "Custom::KMSKey":
		return "KMS Key"
	}

	return fmt.Sprintf("Unknown: %s", t)
}

func waitForAvailability(url string) error {
	tick := time.Tick(10 * time.Second)
	timeout := time.After(20 * time.Minute)

	for {
		select {
		case <-tick:
			fmt.Print(".")

			client := &http.Client{
				Timeout: 2 * time.Second,
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}

			_, err := client.Get(url)

			if err == nil {
				return nil
			}
		case <-timeout:
			stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: fmt.Errorf("timeout")})
			return fmt.Errorf("timeout")
		}
	}

	return fmt.Errorf("unknown error")
}

var randomAlphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")

func randomString(size int) string {
	b := make([]rune, size)
	for i := range b {
		b[i] = randomAlphabet[rand.Intn(len(randomAlphabet))]
	}
	return string(b)
}

func readCredentials(fileName string) (creds *AwsCredentials, err error) {
	// read credentials from ENV
	creds = &AwsCredentials{
		Access:  os.Getenv("AWS_ACCESS_KEY_ID"),
		Secret:  os.Getenv("AWS_SECRET_ACCESS_KEY"),
		Session: os.Getenv("AWS_SESSION_TOKEN"),
	}

	// if filename argument provided, prefer these credentials over any found in the environment
	var inputCreds *AwsCredentials
	if fileName != "" {
		inputCreds, err = readCredentialsFromFile(fileName)
	} else if !terminal.IsTerminal(int(os.Stdin.Fd())) {
		inputCreds, err = readCredentialsFromSTDIN()
	}

	if err != nil {
		return nil, err
	}

	if inputCreds != nil {
		creds = inputCreds
	}

	if creds.Access == "" || creds.Secret == "" {
		awsCLICreds, err := awsCLICredentials()
		if err != nil {
			return nil, err
		}

		if awsCLICreds != nil {
			return awsCLICreds, err
		}

		fmt.Println(CredentialsMessage)

		reader := bufio.NewReader(os.Stdin)

		if creds.Access == "" {
			fmt.Print("AWS Access Key ID: ")
			creds.Access, err = reader.ReadString('\n')
			if err != nil {
				return creds, err
			}
		}

		if creds.Secret == "" {
			fmt.Print("AWS Secret Access Key: ")
			creds.Secret, err = reader.ReadString('\n')
			if err != nil {
				return creds, err
			}
		}

		fmt.Println("")
	}

	creds.Access = strings.TrimSpace(creds.Access)
	creds.Secret = strings.TrimSpace(creds.Secret)
	creds.Session = strings.TrimSpace(creds.Session)

	return
}

func readCredentialsFromFile(credentialsCsvFileName string) (*AwsCredentials, error) {
	fmt.Printf("Reading credentials from file %s\n", credentialsCsvFileName)
	credsFile, err := ioutil.ReadFile(credentialsCsvFileName)
	if err != nil {
		return nil, err
	}

	r := csv.NewReader(bytes.NewReader(credsFile))

	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	creds := &AwsCredentials{}
	if len(records) == 2 {
		switch len(records[0]) {

		case 2:
			// Access key ID,Secret access key
			creds.Access = records[1][0]
			creds.Secret = records[1][1]

		case 3:
			// User name,Access key ID,Secret access key
			creds.Access = records[1][1]
			creds.Secret = records[1][2]

		case 5:
			// User name,Password,Access key ID,Secret access key,Console login link
			creds.Access = records[1][2]
			creds.Secret = records[1][3]

		default:
			return creds, fmt.Errorf("credentials secrets is of unknown length")
		}
	} else {
		return creds, fmt.Errorf("credentials file is of unknown length")
	}

	return creds, nil
}

func readCredentialsFromSTDIN() (creds *AwsCredentials, err error) {
	stdin, err := ioutil.ReadAll(os.Stdin)

	if err != nil {
		return nil, err
	}

	if len(stdin) == 0 {
		return nil, nil
	}

	var input struct {
		Credentials AwsCredentials
	}
	err = json.Unmarshal(stdin, &input)

	if err != nil {
		return nil, err
	}

	return &input.Credentials, err
}

func awsCLICredentials() (*AwsCredentials, error) {
	data, err := awsCLI("help")

	if err != nil && strings.Contains(err.Error(), "executable file not found") {
		fmt.Println("Installing the AWS CLI will allow you to install a Rack without specifying credentials.")
		fmt.Println("See: http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-set-up.html")
		fmt.Println()
		return nil, nil
	}

	fmt.Println("Using AWS CLI for authentication...")

	data, err = awsCLI("iam", "get-account-summary")
	if err != nil {
		if strings.Contains(string(data), "Unable to locate credentials") {
			fmt.Println("You appear to have the AWS CLI installed but have not configured credentials.")
			fmt.Println("You can configure credentials by running `aws configure`.")
			fmt.Println()
			return nil, nil
		}

		return nil, fmt.Errorf("%s: %s", strings.TrimSpace(string(data)), err)
	}

	creds := awsCLICredentialsStatic()

	if creds == nil {
		creds = awsCLICredentialsRole()
	}

	return creds, nil
}

func awsCLICredentialsStatic() *AwsCredentials {
	accessb, err := awsCLI("configure", "get", "aws_access_key_id")
	if err != nil {
		return nil
	}

	secretb, err := awsCLI("configure", "get", "aws_secret_access_key")
	if err != nil {
		return nil
	}

	access := strings.TrimSpace(string(accessb))
	secret := strings.TrimSpace(string(secretb))

	if access != "" && secret != "" {
		return &AwsCredentials{
			Access: access,
			Secret: secret,
		}
	}

	return nil
}

type awsRoleCredentials struct {
	Credentials struct {
		AccessKeyID     string `json:"AccessKeyId"`
		SecretAccessKey string
		SessionToken    string
	}
}

func awsCLICredentialsRole() *AwsCredentials {
	roleb, err := awsCLI("configure", "get", "role_arn")
	if err != nil {
		return nil
	}

	role := strings.TrimSpace(string(roleb))

	if role != "" {
		data, err := awsCLI("sts", "assume-role", "--role-arn", role, "--role-session-name", "convox-cli")
		if err != nil {
			return nil
		}

		var arc awsRoleCredentials

		err = json.Unmarshal(data, &arc)
		if err != nil {
			return nil
		}

		return &AwsCredentials{
			Access:  arc.Credentials.AccessKeyID,
			Secret:  arc.Credentials.SecretAccessKey,
			Session: arc.Credentials.SessionToken,
		}
	}

	return nil
}

func awsCLI(args ...string) ([]byte, error) {
	return exec.Command("aws", args...).CombinedOutput()
}
