package main

import (
	"fmt"
	"time"

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

func cmdUninstall(c *cli.Context) error {
	ep := stdcli.QOSEventProperties{Start: time.Now()}

	distinctId, err := currentId()
	if err != nil {
		return stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
	}

	if !c.Bool("force") {
		apps, err := rackClient(c).GetApps()
		if err != nil {
			return stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
		}

		if len(apps) != 0 {
			return stdcli.ExitError(fmt.Errorf("Please delete all apps before uninstalling."))
		}

		services, err := rackClient(c).GetServices()
		if err != nil {
			return stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
		}

		if len(services) != 0 {
			return stdcli.ExitError(fmt.Errorf("Please delete all services before uninstalling."))
		}
	}

	fmt.Println(Banner)

	creds, err := readCredentials(c)
	if err != nil {
		return stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
	}
	if creds == nil {
		return stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: fmt.Errorf("error reading credentials")})
	}

	region := c.String("region")
	stackName := c.String("stack-name")

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
