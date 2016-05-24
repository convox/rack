package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

type Stacks struct {
	Apps     []Stack
	Rack     []Stack
	Services []Stack
}

type Stack struct {
	Name      string
	StackName string
	Status    string
	Type      string

	Outputs map[string]string
}

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

	rackName := c.Args()[0]
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

	stacks, err := describeRackStacks(rackName, region, creds, distinctId)
	if err != nil {
		stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
	}

	// verify that rack was detected
	if len(stacks.Rack) == 0 || stacks.Rack[0].StackName != rackName {
		stdcli.Error(fmt.Errorf("Can not find rack named %s.", rackName))
	}

	fmt.Println("Resources to uninstall:\n")

	// display all the services, apps, then rack
	t := stdcli.NewTable("STACK", "TYPE", "STATUS")

	for _, s := range stacks.Services {
		t.AddRow(s.Name, s.Type, s.Status)
	}

	for _, s := range stacks.Apps {
		t.AddRow(s.Name, s.Type, s.Status)
	}

	t.AddRow(stacks.Rack[0].Name, "rack", stacks.Rack[0].Status)

	t.Print()
	fmt.Println()

	// verify that no stack is being updated
	for _, s := range stacks.all() {
		if strings.HasSuffix(s.Status, "IN_PROGRESS") {
			stdcli.Error(fmt.Errorf("Can not uninstall while %s is updating.", s.StackName))
		}
	}

	// prompt to confirm rack name
	reader := bufio.NewReader(os.Stdin)

	if !c.Bool("force") && terminal.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Printf("Confirm rack name to delete all the stacks, apps and rack. WARNING, this is irreversable: ")

		n, err := reader.ReadString('\n')
		if err != nil {
			stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
		}

		if strings.TrimSpace(n) != rackName {
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

	success := true

	err = deleteStacks("service", rackName, region, creds, distinctId)
	if err != nil {
		success = false
	}

	err = deleteStacks("app", rackName, region, creds, distinctId)
	if err != nil {
		success = false
	}

	err = deleteStacks("rack", rackName, region, creds, distinctId)
	if err != nil {
		success = false
	}

	if success {
		fmt.Println("Successfully uninstalled.")
	} else {
		stdcli.Error(fmt.Errorf("stack deletion failed, contact support@convox.com for assistance"))
	}

	host := stacks.Rack[0].Outputs["Dashboard"]

	if configuredHost, _ := currentHost(); configuredHost == host {
		err = removeHost()
		if err != nil {
			stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
		}
	}

	err = removeLogin(host)
	if err != nil {
		stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
	}

	fmt.Println("Successfully uninstalled.")

	return stdcli.QOSEventSend("cli-uninstall", distinctId, ep)
}

func deleteStack(stackName, region string, creds *AwsCredentials, distinctId string) error {
	fmt.Printf("Deleting %s...\n", stackName)

	CloudFormation := cloudformation.New(session.New(), awsConfig(region, creds))
	_, err := CloudFormation.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: aws.String(stackName),
	})
	return err
}

func deleteStacks(stackType, rackName, region string, creds *AwsCredentials, distinctId string) error {
	deleteAttempts := map[string]int{}

	for {
		stacks, err := describeRackStacks(rackName, region, creds, distinctId)
		if err != nil {
			return err
		}

		skipped := []string{}
		toDelete := stacks.byType(stackType)

		// no more stacks exist. Success!
		if len(toDelete) == 0 {
			return nil
		}

		for _, s := range toDelete {
			if deleteAttempts[s.StackName] >= 2 {
				skipped = append(skipped, s.StackName)
			} else {
				switch s.Status {
				case "CREATE_COMPLETE", "ROLLBACK_COMPLETE", "UPDATE_COMPLETE", "UPDATE_ROLLBACK_COMPLETE":
					deleteAttempts[s.StackName] += 1
					deleteStack(s.StackName, region, creds, distinctId)
				case "CREATE_FAILED", "DELETE_FAILED", "ROLLBACK_FAILED", "UPDATE_ROLLBACK_FAILED":
					deleteAttempts[s.StackName] += 1
					deleteStack(s.StackName, region, creds, distinctId)
					// todo: report event?
				case "DELETE_IN_PROGRESS":
					// noop
				default:
					// noop
				}
			}
		}

		if len(skipped) == len(toDelete) {
			return fmt.Errorf("Failed to delete %+v", skipped)
		}

		time.Sleep(5 * time.Second)
	}

	return nil
}

// describeRackStacks uses credentials to describe CF service, app and rack stacks that belong to the rack name and region
func describeRackStacks(rackName, region string, creds *AwsCredentials, distinctId string) (Stacks, error) {
	CloudFormation := cloudformation.New(session.New(), awsConfig(region, creds))
	res, err := CloudFormation.DescribeStacks(&cloudformation.DescribeStacksInput{})
	if err != nil {
		return Stacks{}, err
	}

	apps := []Stack{}
	rack := []Stack{}
	services := []Stack{}

	for _, stack := range res.Stacks {
		outputs := map[string]string{}
		tags := map[string]string{}

		for _, output := range stack.Outputs {
			outputs[*output.OutputKey] = *output.OutputValue
		}

		for _, tag := range stack.Tags {
			tags[*tag.Key] = *tag.Value
		}

		name := tags["Name"]
		if name == "" {
			name = *stack.StackName
		}

		s := Stack{
			Name:      name,
			StackName: *stack.StackName,
			Status:    *stack.StackStatus,
			Type:      tags["Type"],

			Outputs: outputs,
		}

		// collect stacks that are explicitly related to the rack
		if tags["Rack"] == rackName {
			switch tags["Type"] {
			case "app":
				apps = append(apps, s)
			case "service":
				services = append(services, s)
			}
		}

		// collect stack that is explicitly the rack
		if *stack.StackName == rackName && outputs["Dashboard"] != "" {
			rack = append(rack, s)
		}
	}

	return Stacks{
		Apps:     apps,
		Rack:     rack,
		Services: services,
	}, nil
}

func (stacks Stacks) all() []Stack {
	s := []Stack{}

	for _, stack := range stacks.Services {
		s = append(s, stack)
	}

	for _, stack := range stacks.Apps {
		s = append(s, stack)
	}

	for _, stack := range stacks.Rack {
		s = append(s, stack)
	}

	return s
}

func (stacks Stacks) byType(t string) []Stack {
	switch t {
	case "app":
		return stacks.Apps
	case "rack":
		return stacks.Rack
	case "service":
		return stacks.Services
	default:
		return []Stack{}
	}
}
