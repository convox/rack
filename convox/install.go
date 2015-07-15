package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/cli/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/convox/cli/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	stdcli.RegisterCommand(cli.Command{
		Name:        "install",
		Description: "install convox into an aws account",
		Usage:       "",
		Action:      cmdInstall,
	})
}

func cmdInstall(c *cli.Context) {
	fmt.Println(`

     ___    ___     ___   __  __    ___   __  _  
    /'___\ / __'\ /' _ '\/\ \/\ \  / __'\/\ \/'\
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
	fmt.Println("https://console.aws.amazon.com/iam/home?region=us-east-1#security_credential")
	fmt.Println("")

	reader := bufio.NewReader(os.Stdin)

	access := os.Getenv("AWS_ACCESS_KEY_ID")
	secret := os.Getenv("AWS_SECRET_ACCESS_KEY")

	if access == "" || secret == "" {
		fmt.Print("AWS Access Key: ")

		access, err := reader.ReadString('\n')

		if err != nil {
			stdcli.Error(err)
		}

		fmt.Print("AWS Secret Access Key: ")

		secret, err := reader.ReadString('\n')

		if err != nil {
			stdcli.Error(err)
		}

		access = strings.TrimSpace(access)
		secret = strings.TrimSpace(secret)
	}

	fmt.Println("")

	fmt.Println("Installing Convox...")

	access = strings.TrimSpace(access)
	secret = strings.TrimSpace(secret)

	password := randomString(30)

	CloudFormation := cloudformation.New(&aws.Config{
		Region:      "us-east-1",
		Credentials: credentials.NewStaticCredentials(access, secret, ""),
	})

	res, err := CloudFormation.CreateStack(&cloudformation.CreateStackInput{
		Capabilities: []*string{aws.String("CAPABILITY_IAM")},
		Parameters: []*cloudformation.Parameter{
			&cloudformation.Parameter{ParameterKey: aws.String("Development"), ParameterValue: aws.String("No")},
			&cloudformation.Parameter{ParameterKey: aws.String("InstanceCount"), ParameterValue: aws.String("3")},
			&cloudformation.Parameter{ParameterKey: aws.String("InstanceType"), ParameterValue: aws.String("t2.small")},
			&cloudformation.Parameter{ParameterKey: aws.String("Password"), ParameterValue: aws.String(password)},
			&cloudformation.Parameter{ParameterKey: aws.String("Version"), ParameterValue: aws.String("latest")},
		},
		StackName:   aws.String(randomString(10)),
		TemplateURL: aws.String("http://convox.s3.amazonaws.com/release/latest/formation.json"),
	})

	if err != nil {
		stdcli.Error(err)
	}

	host, err := waitForCompletion(*res.StackID, CloudFormation)

	if err != nil {
		stdcli.Error(err)
	}

	stdcli.Spinner.Prefix = "Waiting for load balancer: "
	stdcli.Spinner.Start()

	waitForAvailability(host)

	stdcli.Spinner.Stop()
	fmt.Printf("\x08\x08OK\n")

	addLogin(host, password)
	switchHost(host)

	fmt.Println("Success, try `convox apps`")
}

func waitForCompletion(stack string, CloudFormation *cloudformation.CloudFormation) (string, error) {
	for {
		dres, err := CloudFormation.DescribeStacks(&cloudformation.DescribeStacksInput{
			StackName: aws.String(stack),
		})

		if err != nil {
			stdcli.Error(err)
		}

		err = displayProgress(stack, CloudFormation)

		if err != nil {
			stdcli.Error(err)
		}

		if len(dres.Stacks) != 1 {
			stdcli.Error(fmt.Errorf("could not read stack status"))
		}

		switch *dres.Stacks[0].StackStatus {
		case "CREATE_COMPLETE":
			for _, o := range dres.Stacks[0].Outputs {
				if *o.OutputKey == "Dashboard" {
					return *o.OutputValue, nil
				}
			}

			return "", fmt.Errorf("could not install stack")
		case "CREATE_FAILED":
			return "", fmt.Errorf("stack creation failed")
		case "ROLLBACK_COMPLETE":
			return "", fmt.Errorf("stack creation failed")
		}

		time.Sleep(2 * time.Second)
	}
}

var events = map[string]bool{}

func displayProgress(stack string, CloudFormation *cloudformation.CloudFormation) error {
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
			id := *event.PhysicalResourceID

			if strings.HasPrefix(id, "arn:") {
				id = *event.LogicalResourceID
			}

			fmt.Printf("Created %s: %s\n", name, id)
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
	}

	return fmt.Sprintf("UNKNOWN UNKNOWN: %s", t)
}

func waitForAvailability(host string) {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	for {
		_, err := client.Get(fmt.Sprintf("http://%s/", host))

		if err == nil {
			return
		}
	}
}

var randomAlphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")

func randomString(size int) string {
	b := make([]rune, size)
	for i := range b {
		b[i] = randomAlphabet[rand.Intn(len(randomAlphabet))]
	}
	return string(b)
}
