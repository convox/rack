package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/s3"
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

	Buckets []string
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

	CF := cloudformation.New(session.New(), awsConfig(region, creds))
	S3 := s3.New(session.New(), awsConfig(region, creds))

	stacks, err := describeRackStacks(rackName, distinctId, CF)
	if err != nil {
		return stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
	}

	// verify that rack was detected
	if len(stacks.Rack) == 0 || stacks.Rack[0].StackName != rackName {
		return stdcli.ExitError(fmt.Errorf("Can not find rack named %s.", rackName))
	}

	fmt.Println("Resources to delete:\n")

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
			return stdcli.ExitError(fmt.Errorf("Can not uninstall while %s is updating.", s.StackName))
		}
	}

	// prompt to confirm rack name
	reader := bufio.NewReader(os.Stdin)

	if !c.Bool("force") {
		if terminal.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Printf("Delete everything? y/N: ")

			confirm, err := reader.ReadString('\n')
			if err != nil {
				return stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
			}

			if strings.TrimSpace(confirm) != "y" {
				return stdcli.ExitError(fmt.Errorf("Aborting uninstall."))
			}
		} else {
			return stdcli.ExitError(fmt.Errorf("Aborting uninstall. Use the --force for non-interactive uninstall."))
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

	// Delete all Service, App and Rack stacks
	err = deleteStacks("service", rackName, distinctId, CF)
	if err != nil {
		stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
		success = false
	}

	err = deleteStacks("app", rackName, distinctId, CF)
	if err != nil {
		stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
		success = false
	}

	err = deleteStacks("rack", rackName, distinctId, CF)
	if err != nil {
		stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
		success = false
	}

	// Delete all S3 buckets
	wg := new(sync.WaitGroup)

	for _, s := range stacks.Apps {
		for _, b := range s.Buckets {
			wg.Add(1)
			go deleteBucket(b, wg, S3)
		}
	}

	for _, s := range stacks.Rack {
		for _, b := range s.Buckets {
			wg.Add(1)
			go deleteBucket(b, wg, S3)
		}
	}

	wg.Wait()

	// Clean up ~/.convox
	host := stacks.Rack[0].Outputs["Dashboard"]

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

	if success {
		fmt.Println("Successfully uninstalled.")
	} else {
		return stdcli.ExitError(fmt.Errorf("Uninstall encountered some errors, contact support@convox.com for assistance"))
	}

	return stdcli.QOSEventSend("cli-uninstall", distinctId, ep)
}

var deleteAttempts = map[string]int{}

type Obj struct {
	key, id string
}

func deleteBucket(bucket string, wg *sync.WaitGroup, S3 *s3.S3) error {
	keyMarkers := []Obj{}
	versionIdMarkers := []Obj{}

	nextKeyMarker := aws.String("")
	nextVersionIdMarker := aws.String("")

	req := &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
	}

	res, err := S3.ListObjectVersions(req)
	if err != nil {
		wg.Done()
		return err
	}

	for _, d := range res.DeleteMarkers {
		keyMarkers = append(keyMarkers, Obj{key: *d.Key, id: *d.VersionId})
	}

	for _, v := range res.Versions {
		versionIdMarkers = append(versionIdMarkers, Obj{key: *v.Key, id: *v.VersionId})
	}

	nextKeyMarker = res.NextKeyMarker
	nextVersionIdMarker = res.NextVersionIdMarker

	for nextKeyMarker != nil && nextVersionIdMarker != nil {
		req.KeyMarker = nextKeyMarker
		req.VersionIdMarker = nextVersionIdMarker

		res, err := S3.ListObjectVersions(req)
		if err != nil {
			wg.Done()
			return err
		}

		for _, d := range res.DeleteMarkers {
			keyMarkers = append(keyMarkers, Obj{key: *d.Key, id: *d.VersionId})
		}

		for _, v := range res.Versions {
			versionIdMarkers = append(versionIdMarkers, Obj{key: *v.Key, id: *v.VersionId})
		}

		nextKeyMarker = res.NextKeyMarker
		nextVersionIdMarker = res.NextVersionIdMarker
	}

	owg := new(sync.WaitGroup)
	owg.Add(len(keyMarkers))
	owg.Add(len(versionIdMarkers))
	go deleteObjects(bucket, keyMarkers, owg, S3)
	go deleteObjects(bucket, versionIdMarkers, owg, S3)
	owg.Wait()

	wg.Done()
	return nil
}

func deleteObjects(bucket string, objs []Obj, wg *sync.WaitGroup, S3 *s3.S3) {
	maxLen := 1000

	for i := 0; i < len(objs); i += maxLen {
		high := i + maxLen
		if high > len(objs) {
			high = len(objs)
		}

		objects := []*s3.ObjectIdentifier{}
		for _, obj := range objs[i:high] {
			objects = append(objects, &s3.ObjectIdentifier{Key: aws.String(obj.key), VersionId: aws.String(obj.id)})
		}

		S3.DeleteObjects(&s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &s3.Delete{
				Objects: objects,
			},
		})

		wg.Add(-len(objects))
	}

	return
}

func deleteStack(s Stack, distinctId string, CF *cloudformation.CloudFormation) error {
	deleteAttempts[s.StackName] += 1
	switch deleteAttempts[s.StackName] {
	case 1:
		fmt.Printf("Deleting %s...\n", s.Name)
	default:
		fmt.Printf("Retrying deleting %s...\n", s.Name)
	}

	_, err := CF.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: aws.String(s.StackName),
	})
	return err
}

func deleteStacks(stackType, rackName, distinctId string, CF *cloudformation.CloudFormation) error {
	for {
		stacks, err := describeRackStacks(rackName, distinctId, CF)
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
					deleteStack(s, distinctId, CF)
				case "CREATE_FAILED", "DELETE_FAILED", "ROLLBACK_FAILED", "UPDATE_ROLLBACK_FAILED":
					deleteStack(s, distinctId, CF)
					// TODO: report event?
				case "DELETE_IN_PROGRESS":
					displayProgress(s.StackName, CF, true)
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
func describeRackStacks(rackName, distinctId string, CF *cloudformation.CloudFormation) (Stacks, error) {
	res, err := CF.DescribeStacks(&cloudformation.DescribeStacksInput{})
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

		buckets := []string{}

		rres, err := CF.DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
			StackName: stack.StackId,
		})
		if err != nil {
			return Stacks{}, err
		}

		for _, resource := range rres.StackResources {
			if *resource.ResourceType == "AWS::S3::Bucket" {
				buckets = append(buckets, *resource.PhysicalResourceId)
			}
		}

		s := Stack{
			Name:      name,
			StackName: *stack.StackName,
			Status:    *stack.StackStatus,
			Type:      tags["Type"],

			Buckets: buckets,
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
