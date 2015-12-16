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
	"regexp"
	"strings"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/golang.org/x/crypto/ssh/terminal"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/convox/release/version"
	"github.com/convox/rack/cmd/convox/stdcli"
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

var CredentialsMessage = `This installer needs AWS credentials to install the Convox platform into
your AWS account. These credentials will only be used to communicate
between this installer running on your computer and the AWS API.

We recommend that you create a new set of credentials exclusively for this
install process and then delete them once the installer has completed.

To generate a new set of AWS credentials go to:
https://docs.convox.com/creating-an-iam-user
`

var FormationUrl = "http://convox.s3.amazonaws.com/release/%s/formation.json"
var isDevelopment = false

// https://docs.aws.amazon.com/general/latest/gr/rande.html#lambda_region
var lambdaRegions = map[string]bool{"us-east-1": true, "us-west-2": true, "eu-west-1": true, "ap-northeast-1": true, "test": true}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	stdcli.RegisterCommand(cli.Command{
		Name:        "install",
		Description: "install convox into an aws account",
		Usage:       "[credentials.csv]",
		Action:      cmdInstall,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "ami",
				Value: "",
				Usage: "ID of the Amazon Machine Image to install",
			},
			cli.BoolFlag{
				Name:  "dedicated",
				Usage: "create EC2 instances on dedicated hardware",
			},
			cli.IntFlag{
				Name:  "instance-count",
				Value: 3,
				Usage: "number of EC2 instances",
			},
			cli.StringFlag{
				Name:  "instance-type",
				Value: "t2.small",
				Usage: "type of EC2 instances",
			},
			cli.StringFlag{
				Name:   "region",
				Value:  "us-east-1",
				Usage:  "aws region to install in",
				EnvVar: "AWS_REGION",
			},
			cli.StringFlag{
				Name:   "stack-name",
				EnvVar: "STACK_NAME",
				Value:  "convox",
				Usage:  "name of the CloudFormation stack",
			},
			cli.BoolFlag{
				Name:  "private-resources",
				Usage: "Create private rack resources",
			},
			cli.StringFlag{
				Name:  "nat-ami",
				Value: "ami-b0210ed8",
				Usage: "ID of the Amazon Machine Image for nat hosts",
			},
			cli.StringFlag{
				Name:  "nat-instance-type",
				Value: "t2.small",
				Usage: "type of EC2 instances for nat hosts",
			},
			cli.BoolFlag{
				Name:   "development",
				EnvVar: "DEVELOPMENT",
				Usage:  "create additional CloudFormation outputs to copy development .env file",
			},
			cli.BoolFlag{
				Name:  "disable-encryption",
				Usage: "disable encrypting secrets with KMS",
			},
			cli.StringFlag{
				Name:  "key",
				Usage: "name of an SSH keypair to install on EC2 instances",
			},
			cli.StringFlag{
				Name:   "email",
				EnvVar: "CONVOX_EMAIL",
				Usage:  "email address to receive project updates",
			},
			cli.StringFlag{
				Name:   "version",
				EnvVar: "VERSION",
				Value:  "latest",
				Usage:  "release version in the format of '20150810161818', or 'latest' by default",
			},
			cli.StringFlag{
				Name:  "vpc-cidr",
				Value: "10.0.0.0/16",
				Usage: "The VPC CIDR block",
			},
			cli.StringFlag{
				Name:  "subnet0-public-cidr",
				Value: "10.0.1.0/24",
				Usage: "Subnet 0 Public CIDR block",
			},
			cli.StringFlag{
				Name:  "subnet1-public-cidr",
				Value: "10.0.2.0/24",
				Usage: "Subnet 1 Public CIDR block",
			},
			cli.StringFlag{
				Name:  "subnet2-public-cidr",
				Value: "10.0.3.0/24",
				Usage: "Subnet 2 Public CIDR block",
			},
			cli.StringFlag{
				Name:  "subnet0-private-cidr",
				Value: "10.0.4.0/24",
				Usage: "Subnet 0 Private CIDR block, required when --private-resources present",
			},
			cli.StringFlag{
				Name:  "subnet1-private-cidr",
				Value: "10.0.5.0/24",
				Usage: "Subnet 1 Private CIDR block, required when --private-resources present",
			},
			cli.StringFlag{
				Name:  "subnet2-private-cidr",
				Value: "10.0.6.0/24",
				Usage: "Subnet 2 Private CIDR block, required when --private-resources present",
			},
		},
	})

	stdcli.RegisterCommand(cli.Command{
		Name:        "uninstall",
		Description: "uninstall convox from an aws account",
		Usage:       "[credentials.csv]",
		Action:      cmdUninstall,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "force",
				Usage: "uninstall even if apps exist",
			},
			cli.StringFlag{
				Name:   "region",
				Value:  "us-east-1",
				Usage:  "aws region to uninstall from",
				EnvVar: "AWS_REGION",
			},
			cli.StringFlag{
				Name:   "stack-name",
				EnvVar: "STACK_NAME",
				Value:  "convox",
				Usage:  "name of the convox stack",
			},
		},
	})
}

func cmdInstall(c *cli.Context) {
	region := c.String("region")

	if !lambdaRegions[region] {
		stdcli.Error(fmt.Errorf("Convox is not currently supported in %s", region))
	}

	tenancy := "default"
	instanceType := c.String("instance-type")

	if c.Bool("dedicated") {
		tenancy = "dedicated"
		if strings.HasPrefix(instanceType, "t2") {
			stdcli.Error(fmt.Errorf("t2 instance types aren't supported in dedicated tenancy, please set --instance-type."))
		}
	}

	fmt.Println(Banner)

	distinctId, err := currentId()
	creds, err := readCredentials(c)

	if err != nil {
		handleError("install", distinctId, err)
		return
	}

	if creds == nil {
		err = fmt.Errorf("error reading credentials")
		handleError("install", distinctId, err)
		return
	}

	reader := bufio.NewReader(os.Stdin)

	if email := c.String("email"); email != "" {
		distinctId = email
		updateId(distinctId)
	} else if terminal.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Print("Email Address (optional, to receive project updates): ")

		email, err := reader.ReadString('\n')

		if err != nil {
			handleError("install", distinctId, err)
			return
		}

		if strings.TrimSpace(email) != "" {
			distinctId = email
			updateId(email)
		}
	}

	development := "No"
	if c.Bool("development") {
		isDevelopment = true
		development = "Yes"
	}

	privateResources := "No"
	if c.Bool("private-resources") {
		privateResources = "Yes"
	}
	natInstanceType := c.String("nat-instance-type")
	natAmi := c.String("nat-ami")

	encryption := "Yes"
	if c.Bool("disable-encryption") {
		encryption = "No"
	}

	ami := c.String("ami")

	key := c.String("key")

	stackName := c.String("stack-name")

	vpcCIDR := c.String("vpc-cidr")

	subnet0PublicCIDR := c.String("subnet0-public-cidr")
	subnet1PublicCIDR := c.String("subnet1-public-cidr")
	subnet2PublicCIDR := c.String("subnet2-public-cidr")

	subnet0PrivateCIDR := c.String("subnet0-private-cidr")
	subnet1PrivateCIDR := c.String("subnet1-private-cidr")
	subnet2PrivateCIDR := c.String("subnet2-private-cidr")

	versions, err := version.All()

	if err != nil {
		handleError("install", distinctId, err)
		return
	}

	version, err := versions.Resolve(c.String("version"))

	if err != nil {
		handleError("install", distinctId, err)
		return
	}

	versionName := version.Version
	formationUrl := fmt.Sprintf(FormationUrl, versionName)

	instanceCount := fmt.Sprintf("%d", c.Int("instance-count"))

	fmt.Printf("Installing Convox (%s)...\n", versionName)

	if isDevelopment {
		fmt.Println("(Development Mode)")
	}

	password := randomString(30)

	CloudFormation := cloudformation.New(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(creds.Access, creds.Secret, creds.Session),
	})

	req := &cloudformation.CreateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		Parameters: []*cloudformation.Parameter{
			&cloudformation.Parameter{ParameterKey: aws.String("Ami"), ParameterValue: aws.String(ami)},
			&cloudformation.Parameter{ParameterKey: aws.String("ClientId"), ParameterValue: aws.String(distinctId)},
			&cloudformation.Parameter{ParameterKey: aws.String("Development"), ParameterValue: aws.String(development)},
			&cloudformation.Parameter{ParameterKey: aws.String("Encryption"), ParameterValue: aws.String(encryption)},
			&cloudformation.Parameter{ParameterKey: aws.String("InstanceCount"), ParameterValue: aws.String(instanceCount)},
			&cloudformation.Parameter{ParameterKey: aws.String("InstanceType"), ParameterValue: aws.String(instanceType)},
			&cloudformation.Parameter{ParameterKey: aws.String("Key"), ParameterValue: aws.String(key)},
			&cloudformation.Parameter{ParameterKey: aws.String("Password"), ParameterValue: aws.String(password)},
			&cloudformation.Parameter{ParameterKey: aws.String("PrivateResources"), ParameterValue: aws.String(privateResources)},
			&cloudformation.Parameter{ParameterKey: aws.String("NatInstanceType"), ParameterValue: aws.String(natInstanceType)},
			&cloudformation.Parameter{ParameterKey: aws.String("NatAmi"), ParameterValue: aws.String(natAmi)},
			&cloudformation.Parameter{ParameterKey: aws.String("Tenancy"), ParameterValue: aws.String(tenancy)},
			&cloudformation.Parameter{ParameterKey: aws.String("Version"), ParameterValue: aws.String(versionName)},
			&cloudformation.Parameter{ParameterKey: aws.String("Subnet0PublicCIDR"), ParameterValue: aws.String(subnet0PublicCIDR)},
			&cloudformation.Parameter{ParameterKey: aws.String("Subnet1PublicCIDR"), ParameterValue: aws.String(subnet1PublicCIDR)},
			&cloudformation.Parameter{ParameterKey: aws.String("Subnet2PublicCIDR"), ParameterValue: aws.String(subnet2PublicCIDR)},
			&cloudformation.Parameter{ParameterKey: aws.String("Subnet0PrivateCIDR"), ParameterValue: aws.String(subnet0PrivateCIDR)},
			&cloudformation.Parameter{ParameterKey: aws.String("Subnet1PrivateCIDR"), ParameterValue: aws.String(subnet1PrivateCIDR)},
			&cloudformation.Parameter{ParameterKey: aws.String("Subnet2PrivateCIDR"), ParameterValue: aws.String(subnet2PrivateCIDR)},
			&cloudformation.Parameter{ParameterKey: aws.String("VPCCIDR"), ParameterValue: aws.String(vpcCIDR)},
		},
		StackName:   aws.String(stackName),
		TemplateURL: aws.String(formationUrl),
	}

	if tf := os.Getenv("TEMPLATE_FILE"); tf != "" {
		dat, err := ioutil.ReadFile(tf)

		if err != nil {
			handleError("install", distinctId, err)
		}

		req.TemplateURL = nil
		req.TemplateBody = aws.String(string(dat))
	}

	res, err := CloudFormation.CreateStack(req)

	// NOTE: we start making lots of network requests here
	//			 so we're just going to return for testability
	if os.Getenv("AWS_REGION") == "test" {
		fmt.Println(*res.StackId)
		return
	}

	if err != nil {
		sendMixpanelEvent(fmt.Sprintf("convox-install-error"), err.Error())

		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "AlreadyExistsException" {
				stdcli.Error(fmt.Errorf("Stack %q already exists. Run `convox uninstall` then try again.", stackName))
			}
		}

		stdcli.Error(err)
	}

	sendMixpanelEvent("convox-install-start", "")

	host, err := waitForCompletion(*res.StackId, CloudFormation, false)

	if err != nil {
		handleError("install", distinctId, err)
		return
	}

	fmt.Println("Waiting for load balancer...")

	waitForAvailability(fmt.Sprintf("http://%s/", host))

	fmt.Println("Logging in...")

	addLogin(host, password)
	switchHost(host)

	fmt.Println("Success, try `convox apps`")

	sendMixpanelEvent("convox-install-success", "")
}

func cmdUninstall(c *cli.Context) {
	if !c.Bool("force") {
		apps, err := rackClient(c).GetApps()

		if err != nil {
			stdcli.Error(err)
			return
		}

		if len(apps) != 0 {
			stdcli.Error(fmt.Errorf("Please delete all apps before uninstalling."))
		}
	}

	fmt.Println(Banner)

	creds, err := readCredentials(c)
	if err != nil {
		stdcli.Error(err)
		return
	}

	if creds == nil {
		stdcli.Error(fmt.Errorf("error reading credentials"))
		return
	}

	region := c.String("region")
	stackName := c.String("stack-name")

	fmt.Println("")

	fmt.Println("Uninstalling Convox...")

	distinctId := randomString(10)

	CloudFormation := cloudformation.New(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(creds.Access, creds.Secret, creds.Session),
	})

	res, err := CloudFormation.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})

	if err != nil {
		sendMixpanelEvent(fmt.Sprintf("convox-uninstall-error"), err.Error())

		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "ValidationError" {
				stdcli.Error(fmt.Errorf("Stack %q does not exist.", stackName))
			}
		}

		stdcli.Error(err)
	}

	stackId := *res.Stacks[0].StackId

	_, err = CloudFormation.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: aws.String(stackId),
	})

	if err != nil {
		handleError("uninstall", distinctId, err)
		return
	}

	sendMixpanelEvent("convox-uninstall-start", "")

	_, err = waitForCompletion(stackId, CloudFormation, true)

	if err != nil {
		handleError("uninstall", distinctId, err)
		return
	}

	host := ""
	for _, o := range res.Stacks[0].Outputs {
		if *o.OutputKey == "Dashboard" {
			host = *o.OutputValue
			break
		}
	}

	if configuredHost, _ := currentHost(); configuredHost == host {
		removeHost()
	}
	removeLogin(host)

	fmt.Println("Successfully uninstalled.")

	sendMixpanelEvent("convox-uninstall-success", "")
}

func waitForCompletion(stack string, CloudFormation *cloudformation.CloudFormation, isDeleting bool) (string, error) {
	for {
		dres, err := CloudFormation.DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: aws.String(stack),
		})

		if err != nil {
			stdcli.Error(err)
		}

		err = displayProgress(stack, CloudFormation, isDeleting)

		if err != nil {
			stdcli.Error(err)
		}

		if len(dres.Stacks) != 1 {
			stdcli.Error(fmt.Errorf("could not read stack status"))
		}

		switch *dres.Stacks[0].StackStatus {
		case "CREATE_COMPLETE":
			// Dump .env if DEVELOPMENT
			if isDevelopment {
				fmt.Printf("Development .env:\n")

				// convert Port5432TcpAddr to PORT_5432_TCP_ADDR
				re := regexp.MustCompile("([a-z])([A-Z0-9])") // lower case letter followed by upper case or number, i.e. Port5432
				re2 := regexp.MustCompile("([0-9])([A-Z])")   // number followed by upper case letter, i.e. 5432Tcp

				for _, o := range dres.Stacks[0].Outputs {
					k := re.ReplaceAllString(*o.OutputKey, "${1}_${2}")
					k = re2.ReplaceAllString(k, "${1}_${2}")
					k = strings.ToUpper(k)

					fmt.Printf("%v=%v\n", k, *o.OutputValue)
				}
			}

			for _, o := range dres.Stacks[0].Outputs {
				if *o.OutputKey == "Dashboard" {
					return *o.OutputValue, nil
				}
			}

			return "", fmt.Errorf("could not install stack, contact support@convox.com for assistance")
		case "CREATE_FAILED":
			return "", fmt.Errorf("stack creation failed, contact support@convox.com for assistance")
		case "ROLLBACK_COMPLETE":
			return "", fmt.Errorf("stack creation failed, contact support@convox.com for assistance")
		case "DELETE_COMPLETE":
			return "", nil
		case "DELETE_FAILED":
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
		if events[*event.EventId] == true {
			continue
		}

		events[*event.EventId] = true

		// Log all CREATE_FAILED to display and MixPanel
		if !isDeleting && *event.ResourceStatus == "CREATE_FAILED" {
			msg := fmt.Sprintf("Failed %s: %s", *event.ResourceType, *event.ResourceStatusReason)
			fmt.Println(msg)
			sendMixpanelEvent("convox-install-error", msg)
		}

		name := friendlyName(*event.ResourceType)

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
			return fmt.Errorf("stack deletion failed")
		case "ROLLBACK_IN_PROGRESS", "ROLLBACK_COMPLETE":
		case "UPDATE_IN_PROGRESS", "UPDATE_COMPLETE", "UPDATE_COMPLETE_CLEANUP_IN_PROGRESS", "UPDATE_FAILED", "UPDATE_ROLLBACK_IN_PROGRESS", "UPDATE_ROLLBACK_COMPLETE", "UPDATE_ROLLBACK_FAILED":
		default:
			return fmt.Errorf("Unhandled status: %s\n", *event.ResourceStatus)
		}
	}

	return nil
}

func friendlyName(t string) string {
	switch t {
	case "AWS::AutoScaling::AutoScalingGroup":
		return "AutoScalingGroup"
	case "AWS::AutoScaling::LaunchConfiguration":
		return ""
	case "AWS::CloudFormation::Stack":
		return "CloudFormation Stack"
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
	case "AWS::ElasticLoadBalancing::LoadBalancer":
		return "Elastic Load Balancer"
	case "AWS::Lambda::Function":
		return "Lambda Function"
	case "AWS::IAM::AccessKey":
		return "Access Key"
	case "AWS::IAM::InstanceProfile":
		return ""
	case "AWS::IAM::Role":
		return ""
	case "AWS::IAM::User":
		return "IAM User"
	case "AWS::Kinesis::Stream":
		return "Kinesis Stream"
	case "AWS::S3::Bucket":
		return "S3 Bucket"
	case "AWS::DynamoDB::Table":
		return "DynamoDB Table"
	case "Custom::EC2AvailabilityZones":
		return ""
	case "Custom::ECSTaskDefinition":
		return "ECS TaskDefinition"
	case "Custom::ECSService":
		return "ECS Service"
	case "Custom::S3BucketCleanup":
		return ""
	case "Custom::KMSKey":
		return "KMS Key"
	}

	return fmt.Sprintf("Unknown: %s", t)
}

func waitForAvailability(url string) {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	for {
		_, err := client.Get(url)

		if err == nil {
			return
		}
	}
}

func handleError(command string, distinctId string, err error) {
	sendMixpanelEvent(fmt.Sprintf("convox-%s-error", command), err.Error())
	stdcli.Error(err)
}

var randomAlphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")

func randomString(size int) string {
	b := make([]rune, size)
	for i := range b {
		b[i] = randomAlphabet[rand.Intn(len(randomAlphabet))]
	}
	return string(b)
}

func readCredentials(c *cli.Context) (creds *AwsCredentials, err error) {
	// read credentials from ENV
	creds = &AwsCredentials{
		Access:  os.Getenv("AWS_ACCESS_KEY_ID"),
		Secret:  os.Getenv("AWS_SECRET_ACCESS_KEY"),
		Session: os.Getenv("AWS_SESSION_TOKEN"),
	}

	if os.Getenv("AWS_ENDPOINT_URL") != "" {
		url := os.Getenv("AWS_ENDPOINT_URL")
		defaults.DefaultConfig.Endpoint = &url
	}

	var inputCreds *AwsCredentials
	if len(c.Args()) > 0 {
		fileName := c.Args()[0]
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
		reader := bufio.NewReader(os.Stdin)

		fmt.Println(CredentialsMessage)

		fmt.Print("AWS Access Key ID: ")

		creds.Access, err = reader.ReadString('\n')

		if err != nil {
			return creds, err
		}

		fmt.Print("AWS Secret Access Key: ")

		creds.Secret, err = reader.ReadString('\n')

		if err != nil {
			return creds, err
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
