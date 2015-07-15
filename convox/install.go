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
	reader := bufio.NewReader(os.Stdin)

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

	password := randomString(30)

	stdcli.Spinner.Prefix = "Installing: "
	stdcli.Spinner.Start()

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

	stdcli.Spinner.Stop()
	fmt.Printf("\x08\x08OK\n")

	if err != nil {
		stdcli.Error(err)
	}

	stdcli.Spinner.Prefix = "Booting: "
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
