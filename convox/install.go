package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/cli/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/cli/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/convox/cli/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/cli/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/s3"
	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

type Event struct {
	Id   string
	Name string

	Reason string
	Status string
	Type   string

	Time time.Time
}

type Events []Event

var FormationUrl = "http://convox.s3.amazonaws.com/release/latest/formation.json"

// https://docs.aws.amazon.com/general/latest/gr/rande.html#lambda_region
var lambdaRegions = map[string]bool{"us-east-1": true, "us-west-2": true, "eu-west-1": true, "ap-northeast-1": true}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	stdcli.RegisterCommand(cli.Command{
		Name:        "install",
		Description: "install convox into an aws account",
		Usage:       "",
		Action:      cmdInstall,
		Flags: []cli.Flag{
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
		},
	})

	stdcli.RegisterCommand(cli.Command{
		Name:        "uninstall",
		Description: "uninstall convox from an aws account",
		Usage:       "",
		Action:      cmdUninstall,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "region",
				Value:  "us-east-1",
				Usage:  "aws region to uninstall from",
				EnvVar: "AWS_REGION",
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

	fmt.Println(`

     ___    ___     ___   __  __    ___   __  _
    / ___\ / __ \ /  _  \/\ \/\ \  / __ \/\ \/ \
   /\ \__//\ \_\ \/\ \/\ \ \ \_/ |/\ \_\ \/>  </
   \ \____\ \____/\ \_\ \_\ \___/ \ \____//\_/\_\
    \/____/\/___/  \/_/\/_/\/__/   \/___/ \//\/_/

 `)

	fmt.Println("This installer needs AWS credentials to install the Convox platform into")
	fmt.Println("your AWS account. These credentials will only be used to communicate")
	fmt.Println("between this installer running on your computer and the AWS API.")
	fmt.Println("")
	fmt.Println("We recommend that you create a new set of credentials exclusively for this")
	fmt.Println("install process and then delete them once the installer has completed.")
	fmt.Println("")
	fmt.Println("To generate a new set of AWS credentials go to:")
	fmt.Println("https://console.aws.amazon.com/iam/home#security_credential")
	fmt.Println("")

	distinctId, err := currentId()

	if err != nil {
		handleError("install", distinctId, err)
		return
	}

	reader := bufio.NewReader(os.Stdin)

	access := os.Getenv("AWS_ACCESS_KEY_ID")
	secret := os.Getenv("AWS_SECRET_ACCESS_KEY")

	if access == "" || secret == "" {
		var err error

		fmt.Print("AWS Access Key ID: ")

		access, err = reader.ReadString('\n')

		if err != nil {
			stdcli.Error(err)
		}

		fmt.Print("AWS Secret Access Key: ")

		secret, err = reader.ReadString('\n')

		if err != nil {
			stdcli.Error(err)
		}

		fmt.Println("")
	}

	development := os.Getenv("DEVELOPMENT")

	if development != "Yes" {
		development = "No"
	}

	key := os.Getenv("KEY")

	stackName := os.Getenv("STACK_NAME")

	if stackName == "" {
		stackName = "convox"
	}

	version := os.Getenv("VERSION")

	if version == "" {
		version = "latest"
	}

	instanceCount := fmt.Sprintf("%d", c.Int("instance-count"))

	fmt.Println("Installing Convox...")

	access = strings.TrimSpace(access)
	secret = strings.TrimSpace(secret)

	password := randomString(30)

	CloudFormation := cloudformation.New(&aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentials(access, secret, ""),
	})

	res, err := CloudFormation.CreateStack(&cloudformation.CreateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		Parameters: []*cloudformation.Parameter{
			&cloudformation.Parameter{ParameterKey: aws.String("ClientId"), ParameterValue: aws.String(distinctId)},
			&cloudformation.Parameter{ParameterKey: aws.String("Development"), ParameterValue: aws.String(development)},
			&cloudformation.Parameter{ParameterKey: aws.String("InstanceCount"), ParameterValue: aws.String(instanceCount)},
			&cloudformation.Parameter{ParameterKey: aws.String("InstanceType"), ParameterValue: aws.String(instanceType)},
			&cloudformation.Parameter{ParameterKey: aws.String("Key"), ParameterValue: aws.String(key)},
			&cloudformation.Parameter{ParameterKey: aws.String("Password"), ParameterValue: aws.String(password)},
			&cloudformation.Parameter{ParameterKey: aws.String("Tenancy"), ParameterValue: aws.String(tenancy)},
			&cloudformation.Parameter{ParameterKey: aws.String("Version"), ParameterValue: aws.String(version)},
		},
		StackName:   aws.String(stackName),
		TemplateURL: aws.String(FormationUrl),
	})

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

	host, err := waitForCompletion(*res.StackID, CloudFormation, false)

	if err != nil {
		logEvents("install", region, access, secret, stackName)
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
	fmt.Println(`

     ___    ___     ___   __  __    ___   __  _
    /'___\ / __'\ /' _ '\/\ \/\ \  / __'\/\ \/'\
   /\ \__//\ \_\ \/\ \/\ \ \ \_/ |/\ \_\ \/>  </
   \ \____\ \____/\ \_\ \_\ \___/ \ \____//\_/\_\
    \/____/\/___/  \/_/\/_/\/__/   \/___/ \//\/_/

 `)

	fmt.Println("This uninstaller needs AWS credentials to uninstall the Convox platform from")
	fmt.Println("your AWS account. These credentials will only be used to communicate")
	fmt.Println("between this uninstaller running on your computer and the AWS API.")
	fmt.Println("")
	fmt.Println("We recommend that you create a new set of credentials exclusively for this")
	fmt.Println("uninstall process and then delete them once the uninstaller has completed.")
	fmt.Println("")
	fmt.Println("To generate a new set of AWS credentials go to:")
	fmt.Println("https://console.aws.amazon.com/iam/home#security_credential")
	fmt.Println("")

	reader := bufio.NewReader(os.Stdin)

	access := os.Getenv("AWS_ACCESS_KEY_ID")
	secret := os.Getenv("AWS_SECRET_ACCESS_KEY")
	region := c.String("region")

	if access == "" || secret == "" {
		var err error

		fmt.Print("AWS Access Key: ")

		access, err = reader.ReadString('\n')

		if err != nil {
			stdcli.Error(err)
		}

		fmt.Print("AWS Secret Access Key: ")

		secret, err = reader.ReadString('\n')

		if err != nil {
			stdcli.Error(err)
		}
	}

	stackName := os.Getenv("STACK_NAME")

	if stackName == "" {
		stackName = "convox"
	}

	fmt.Println("")

	fmt.Println("Uninstalling Convox...")

	distinctId := randomString(10)

	access = strings.TrimSpace(access)
	secret = strings.TrimSpace(secret)

	CloudFormation := cloudformation.New(&aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentials(access, secret, ""),
	})

	res, err := CloudFormation.DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
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

	stackId := ""
	bucket := ""

	for _, r := range res.StackResources {
		stackId = *r.StackID
		if *r.LogicalResourceID == "RegistryBucket" {
			bucket = *r.PhysicalResourceID
		}
	}

	_, err = CloudFormation.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: aws.String(stackName),
	})

	if err != nil {
		logEvents("uninstall", region, access, secret, stackName)
		handleError("uninstall", distinctId, err)
		return
	}

	sendMixpanelEvent("convox-uninstall-start", "")

	fmt.Printf("Cleaning up registry...\n")

	S3 := s3.New(&aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentials(access, secret, ""),
	})

	req := &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
	}

	sres, err := S3.ListObjectVersions(req)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// Don't block uninstall NoSuchBucket
			if awsErr.Code() != "NoSuchBucket" {
				logEvents("uninstall", region, access, secret, stackName)
				handleError("uninstall", distinctId, err)
			}
		}
	}

	for _, v := range sres.Versions {
		req := &s3.DeleteObjectInput{
			Bucket:    aws.String(bucket),
			Key:       aws.String(*v.Key),
			VersionID: aws.String(*v.VersionID),
		}

		_, err := S3.DeleteObject(req)

		if err != nil {
			logEvents("uninstall", region, access, secret, stackName)
			handleError("uninstall", distinctId, err)
			return
		}
	}

	host, err := waitForCompletion(stackId, CloudFormation, true)

	if err != nil {
		handleError("uninstall", distinctId, err)
		return
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
			development := os.Getenv("DEVELOPMENT")

			if development == "Yes" {
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
			for _, o := range dres.Stacks[0].Outputs {
				if *o.OutputKey == "Dashboard" {
					return *o.OutputValue, nil
				}
			}
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
		if events[*event.EventID] == true {
			continue
		}

		events[*event.EventID] = true

		name := friendlyName(*event.ResourceType)

		if name == "" {
			continue
		}

		switch *event.ResourceStatus {
		case "CREATE_IN_PROGRESS":
		case "CREATE_COMPLETE":
			if !isDeleting {
				id := *event.PhysicalResourceID

				if strings.HasPrefix(id, "arn:") {
					id = *event.LogicalResourceID
				}

				fmt.Printf("Created %s: %s\n", name, id)
			}
		case "CREATE_FAILED":
			return fmt.Errorf("stack creation failed")
		case "DELETE_IN_PROGRESS":
		case "DELETE_COMPLETE":
			id := *event.PhysicalResourceID

			if strings.HasPrefix(id, "arn:") {
				id = *event.LogicalResourceID
			}

			fmt.Printf("Deleted %s: %s\n", name, id)
		case "DELETE_SKIPPED":
			id := *event.PhysicalResourceID

			if strings.HasPrefix(id, "arn:") {
				id = *event.LogicalResourceID
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
	}

	return fmt.Sprintf("Unknown: %s", t)
}

func logEvents(command, region, access, secret, stackName string) error {
	// collect CF events
	CloudFormation := cloudformation.New(&aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentials(access, secret, ""),
	})

	events := Events{}

	data := ""
	next := ""

	for {
		req := &cloudformation.DescribeStackEventsInput{StackName: aws.String(stackName)}

		if next != "" {
			req.NextToken = aws.String(next)
		}

		res, err := CloudFormation.DescribeStackEvents(req)

		if err != nil {
			return err
		}

		for _, event := range res.StackEvents {
			e := Event{
				Id:     cs(event.EventID, ""),
				Name:   cs(event.LogicalResourceID, ""),
				Status: cs(event.ResourceStatus, ""),
				Type:   cs(event.ResourceType, ""),
				Reason: cs(event.ResourceStatusReason, ""),
				Time:   ct(event.Timestamp),
			}

			events = append(events, e)

			data += fmt.Sprintf("%s: [CFM] (%s) %s %s\n", e.Time.Format(time.RFC3339), e.Name, e.Status, e.Reason)
		}

		if res.NextToken == nil {
			break
		}

		next = *res.NextToken
	}

	err := ioutil.WriteFile(command+".log", []byte(data), 0644)

	if err != nil {
		return err
	}

	return nil
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
