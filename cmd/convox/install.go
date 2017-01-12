package main

import (
	"bufio"
	"bytes"
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
	"github.com/aws/aws-sdk-go/service/iam"
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

const CredentialsMessage = `This installer needs AWS credentials to install/uninstall the Convox platform into
your AWS account. These credentials will only be used to communicate between this
installer running on your computer and the AWS API.

We recommend that you create a new set of credentials exclusively for this
install/uninstall process and then delete them once the installer has completed.

To generate a new set of AWS credentials go to:
https://docs.convox.com/creating-an-iam-user`

var (
	formationURL = "https://convox.s3.amazonaws.com/release/%s/formation.json"
	iamUserURL   = "https://docs.convox.com/creating-an-iam-user"
	distinctID   = "nobody"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	var err error
	distinctID, err = currentId()
	if err != nil {
		distinctID = ""
	}

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
			cli.BoolFlag{
				Name:  "dedicated",
				Usage: "create EC2 instances on dedicated hardware",
			},
			cli.StringFlag{
				Name:  "existing-vpc",
				Value: "",
				Usage: "existing vpc id into which to install rack",
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
				Value: "10.0.0.0/16",
				Usage: "custom VPC CIDR",
			},
			cli.StringFlag{
				Name:  "subnet-cidrs",
				Value: "10.0.1.0/24,10.0.2.0/24,10.0.3.0/24",
				Usage: "subnet CIDRs",
			},
		},
	})
}

func cmdInstall(c *cli.Context) error {
	ep := stdcli.QOSEventProperties{Start: time.Now()}

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
			msg := fmt.Errorf("Stack name '%s' is invalid, must match [a-z0-9-]*", stackName)
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

	var existingVPC string

	if vpc := c.String("existing-vpc"); vpc != "" {
		existingVPC = vpc
	}

	internetGateway := c.String("internet-gateway")

	if (existingVPC != "") && (internetGateway == "") {
		return stdcli.Error(fmt.Errorf("must specify valid Internet Gateway for existing VPC"))
	}

	private := "No"
	if c.Bool("private") || strings.ToLower(os.Getenv("RACK_PRIVATE")) == "yes" || strings.ToLower(os.Getenv("RACK_PRIVATE")) == "true" {
		private = "Yes"
	}

	ami := c.String("ami")

	key := c.String("key")

	vpcCIDR := c.String("vpc-cidr")

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
		updateId(distinctID)
	} else if terminal.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Print("Email Address (optional, to receive project updates): ")

		email, err := reader.ReadString('\n')
		if err != nil {
			stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: err})
			return stdcli.Error(err)
		}

		if strings.TrimSpace(email) != "" {
			distinctID = email
			updateId(email)
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

	err = validateUserAccess(region, creds)
	if err != nil {
		stdcli.Error(err)
	}

	password := c.String("password")
	if password == "" {
		password = randomString(30)
	}

	CloudFormation := cloudformation.New(session.New(), awsConfig(region, creds))

	req := &cloudformation.CreateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		Parameters: []*cloudformation.Parameter{
			{ParameterKey: aws.String("Ami"), ParameterValue: aws.String(ami)},
			{ParameterKey: aws.String("ClientId"), ParameterValue: aws.String(distinctID)},
			{ParameterKey: aws.String("CustomTopicRuntime"), ParameterValue: aws.String("nodejs4.3")},
			{ParameterKey: aws.String("ExistingVpc"), ParameterValue: aws.String(existingVPC)},
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

	fmt.Println("Waiting for load balancer...")

	if err := waitForAvailability(fmt.Sprintf("http://%s/", host)); err != nil {
		stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: err})
		return stdcli.Error(err)
	}

	fmt.Println("Logging in...")

	err = addLogin(host, password)
	if err != nil {
		stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: err})
		return stdcli.Error(err)
	}

	err = switchHost(host)
	if err != nil {
		stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: err})
		return stdcli.Error(err)
	}

	fmt.Println("Success, try `convox apps`")

	return stdcli.QOSEventSend("cli-install", distinctID, ep)
}

/// validateUserAccess checks for the "AdministratorAccess" policy needed to create a rack.
func validateUserAccess(region string, creds *AwsCredentials) error {

	// this validation need to check for actual permissions somehow and not
	// just a policy name
	return nil

	Iam := iam.New(session.New(), awsConfig(region, creds))

	userOutput, err := Iam.GetUser(&iam.GetUserInput{})
	if err != nil {
		if ae, ok := err.(awserr.Error); ok {
			return fmt.Errorf("%s. See %s", ae.Code(), iamUserURL)
		}
		return fmt.Errorf("%s. See %s", err, iamUserURL)
	}

	policies, err := Iam.ListAttachedUserPolicies(&iam.ListAttachedUserPoliciesInput{
		UserName: userOutput.User.UserName,
	})
	if err != nil {
		if ae, ok := err.(awserr.Error); ok {
			return fmt.Errorf("%s. See %s", ae.Code(), iamUserURL)
		}
	}

	for _, policy := range policies.AttachedPolicies {
		if "AdministratorAccess" == *policy.PolicyName {
			return nil
		}
	}

	msg := fmt.Errorf("Administrator access needed. See %s", iamUserURL)
	stdcli.QOSEventSend("cli-install", distinctID, stdcli.QOSEventProperties{Error: msg})
	return stdcli.Error(msg)
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
	case "AWS::CloudFormation::Stack":
		return "CloudFormation Stack"
	case "AWS::DynamoDB::Table":
		return "DynamoDB Table"
	case "AWS::EC2::EIP":
		return "NAT Elastic IP"
	case "AWS::EC2::InternetGateway":
		return "VPC Internet Gateway"
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
	case "AWS::EFS::FileSystem":
		return "EFS Filesystem"
	case "AWS::EFS::MountTarget":
		return ""
	case "AWS::ElasticLoadBalancing::LoadBalancer":
		return "Elastic Load Balancer"
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
	case "AWS::Lambda::Function":
		return "Lambda Function"
	case "AWS::Lambda::Permission":
		return ""
	case "AWS::Logs::LogGroup":
		return "CloudWatch Log Group"
	case "AWS::Logs::SubscriptionFilter":
		return ""
	case "AWS::EC2::NatGateway":
		return "NAT Gateway"
	case "AWS::S3::Bucket":
		return "S3 Bucket"
	case "AWS::SNS::Topic":
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

	var inputCreds *AwsCredentials
	if fileName != "" {
		inputCreds, err = readCredentialsFromFile(fileName)
	} else if !terminal.IsTerminal(int(os.Stdin.Fd())) {
		inputCreds, err = readCredentialsFromSTDIN()
	}

	if inputCreds != nil {
		creds = inputCreds
	}

	if err != nil {
		return nil, err
	}

	if creds.Access == "" || creds.Secret == "" {
		creds, err = awsCLICredentials()

		if err != nil {
			return nil, err
		}

		if creds != nil {
			return creds, err
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

func readCredentialsFromFile(credentialsCsvFileName string) (creds *AwsCredentials, err error) {
	credsFile, err := ioutil.ReadFile(credentialsCsvFileName)

	if err != nil {
		return nil, err
	}

	creds = &AwsCredentials{}

	r := csv.NewReader(bytes.NewReader(credsFile))
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) == 2 && len(records[1]) == 3 {
		creds.Access = records[1][1]
		creds.Secret = records[1][2]
	}

	return
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
	data, err := awsCLI("iam", "get-user")

	if err != nil && strings.Contains(err.Error(), "executable file not found") {
		fmt.Println("Installing the AWS CLI will allow you to install a Rack without specifying credentials.")
		fmt.Println("See: http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-set-up.html")
		fmt.Println()
		return nil, nil
	}

	if strings.Contains(string(data), "Unable to locate credentials") {
		fmt.Println("You appear to have the AWS CLI installed but have not configured credentials.")
		fmt.Println("You can configure credentials by running `aws configure`.")
		fmt.Println()
		return nil, nil
	}

	access, err := awsCLI("configure", "get", "aws_access_key_id")

	if err != nil {
		return nil, err
	}

	secret, err := awsCLI("configure", "get", "aws_secret_access_key")

	if err != nil {
		return nil, err
	}

	creds := &AwsCredentials{
		Access: strings.TrimSpace(string(access)),
		Secret: strings.TrimSpace(string(secret)),
	}

	return creds, nil
}

func awsCLI(args ...string) ([]byte, error) {
	return exec.Command("aws", args...).CombinedOutput()
}
