package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "uninstall",
		Description: "uninstall convox from an aws account",
		Usage:       "<stack-name> <region> [credentials.csv]",
		Action:      cmdUninstall,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "force",
				Usage: "uninstall without verification prompt",
			},
		},
	})
}

func cmdUninstall(c *cli.Context) error {
	ep := stdcli.QOSEventProperties{Start: time.Now()}

	distinctId, err := currentId()
	if err != nil {
		return stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
	}

	if len(c.Args()) != 2 && len(c.Args()) != 3 {
		stdcli.Usage(c, "uninstall")
		return nil
	}

	stackName := c.Args()[0]
	region := c.Args()[1]

	credentialsFile := ""
	if len(c.Args()) == 3 {
		credentialsFile = c.Args()[3]
	}

	fmt.Println(Banner)

	creds, err := readCredentials(credentialsFile)
	if err != nil {
		return stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
	}
	if creds == nil {
		return stdcli.ExitError(fmt.Errorf("error reading credentials"))
	}

	reader := bufio.NewReader(os.Stdin)

	if !c.Bool("force") && terminal.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Printf("Confirm stack name to delete: ")

		n, err := reader.ReadString('\n')
		if err != nil {
			stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
		}

		if strings.TrimSpace(n) != stackName {
			stdcli.Error(fmt.Errorf("Name does not match. Aborting uninstall."))
		}
	}

	fmt.Println("")

	fmt.Println("Uninstalling Convox...")

	// CF Stack Delete and Retry could take 30+ minutes. Periodically generate more progress output.
	go func() {
		t := time.Tick(2 * time.Minute)
		for range t {
			fmt.Println("Uninstalling Convox...")
		}
	}()

	CloudFormation := cloudformation.New(session.New(), awsConfig(region, creds))

	res, err := CloudFormation.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "ValidationError" {
				return stdcli.ExitError(fmt.Errorf("Stack %q does not exist.", stackName))
			}
		}

		return stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
	}

	stackId := *res.Stacks[0].StackId

	_, err = CloudFormation.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: aws.String(stackId),
	})
	if err != nil {
		return stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
	}

	_, err = waitForCompletion(stackId, CloudFormation, true)
	if err != nil {
		// retry deleting stack once more to automate around transient errors
		_, err = CloudFormation.DeleteStack(&cloudformation.DeleteStackInput{
			StackName: aws.String(stackId),
		})
		if err != nil {
			return stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
		}

		_, err = waitForCompletion(stackId, CloudFormation, true)
		if err != nil {
			return stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
		}
	}

	host := ""
	for _, o := range res.Stacks[0].Outputs {
		if *o.OutputKey == "Dashboard" {
			host = *o.OutputValue
			break
		}
	}

	if configuredHost, _ := currentHost(); configuredHost == host {
		err = removeHost()
		if err != nil {
			return stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
		}
	}

	err = removeLogin(host)
	if err != nil {
		return stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
	}

	fmt.Println("Successfully uninstalled.")

	return stdcli.QOSEventSend("cli-uninstall", distinctId, ep)
}
