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
	Apps      []Stack
	Rack      []Stack
	Resources []Stack
}

type Stack struct {
	Name      string
	Outputs   map[string]string
	StackName string
	Status    string
	Type      string
}

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "uninstall",
		Description: "uninstall a convox rack",
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
	stdcli.NeedHelp(c)

	ep := stdcli.QOSEventProperties{Start: time.Now()}

	distinctId, err := currentId()
	if err != nil {
		return stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
	}

	if len(c.Args()) != 2 && len(c.Args()) != 3 {
		stdcli.Usage(c)
		return nil
	}

	rackName := c.Args()[0]
	region := c.Args()[1]

	credentialsFile := ""
	if len(c.Args()) == 3 {
		credentialsFile = c.Args()[2]
	}

	fmt.Println(Banner)

	creds, err := readCredentials(credentialsFile)
	if err != nil {
		return stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
	}
	if creds == nil {
		return stdcli.Error(fmt.Errorf("error reading credentials"))
	}

	CF := cloudformation.New(session.New(), awsConfig(region, creds))
	S3 := s3.New(session.New(), awsConfig(region, creds))

	stacks, err := describeRackStacks(rackName, CF)
	if err != nil {
		return stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
	}

	// verify that rack was detected
	if len(stacks.Rack) == 0 || stacks.Rack[0].StackName != rackName {
		return stdcli.Error(fmt.Errorf("can not find rack named %s\nAre you authenticating with the correct AWS account?\nSee AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, and ~/.aws/credentials", rackName))
	}

	fmt.Println("Resources to delete:")

	// display all the resources, apps, then rack
	t := stdcli.NewTable("STACK", "TYPE", "STATUS")

	for _, s := range stacks.Resources {
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
			return stdcli.Error(fmt.Errorf("Can not uninstall while %s is updating.", s.StackName))
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
				return stdcli.Error(fmt.Errorf("Aborting uninstall."))
			}
		} else {
			return stdcli.Error(fmt.Errorf("Aborting uninstall. Use the --force for non-interactive uninstall."))
		}
	}

	fmt.Println()
	fmt.Println("Uninstalling Convox...")

	// collect buckets before deleting the stacks
	buckets := []string{}
	for _, s := range stacks.all() {
		stack := rackName
		if s.Name != rackName {
			stack = fmt.Sprintf("%s-%s", rackName, s.Name)
		}

		bs, err := describeStackBuckets(stack, CF)
		if err != nil {
			stdcli.Error(fmt.Errorf("Unable to gather buckets for %s, skipping", stack))
			stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
			continue
		}

		buckets = append(buckets, bs...)
		time.Sleep(2 * time.Second)
	}

	success := true
	var deleteErr error

	// Delete all Service, App and Rack stacks
	err = deleteStacks("resource", rackName, CF)
	if err != nil {
		stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
		success = false
		deleteErr = err
	}

	err = deleteStacks("app", rackName, CF)
	if err != nil {
		stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
		success = false
		deleteErr = err
	}

	err = deleteStacks("rack", rackName, CF)
	if err != nil {
		stdcli.QOSEventSend("cli-uninstall", distinctId, stdcli.QOSEventProperties{Error: err})
		success = false
		deleteErr = err
	}

	// Delete all S3 buckets
	wg := new(sync.WaitGroup)

	for _, b := range buckets {
		wg.Add(1)
		go deleteBucket(b, wg, S3)
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
		stdcli.Error(deleteErr)
		return stdcli.Error(fmt.Errorf("Uninstall encountered some errors, contact support@convox.com for assistance"))
	}

	return stdcli.QOSEventSend("cli-uninstall", distinctId, ep)
}

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

	fmt.Printf("Emptying S3 Bucket %s...\n", bucket)

	owg := new(sync.WaitGroup)
	owg.Add(len(keyMarkers))
	owg.Add(len(versionIdMarkers))
	go deleteObjects(bucket, keyMarkers, owg, S3)
	go deleteObjects(bucket, versionIdMarkers, owg, S3)
	owg.Wait()

	fmt.Printf("Deleting S3 Bucket %s...\n", bucket)

	_, err = S3.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		fmt.Printf("Failed: %s\n", err)
	}

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

		_, err := S3.DeleteObjects(&s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &s3.Delete{
				Objects: objects,
			},
		})
		if err != nil {
			fmt.Printf("Failed: %s\n", err)
		}

		wg.Add(-len(objects))
	}

	return
}

var deleteAttempts = map[string]int{}

func deleteStack(s Stack, CF *cloudformation.CloudFormation) error {
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

func deleteStacks(stackType, rackName string, CF *cloudformation.CloudFormation) error {
	for {
		stacks, err := describeRackStacks(rackName, CF)
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
			if deleteAttempts[s.StackName] >= 3 { // after the 3rd delete, stop monitoring progress
				skipped = append(skipped, s.StackName)
			} else {
				switch s.Status {
				case "CREATE_COMPLETE", "ROLLBACK_COMPLETE", "UPDATE_COMPLETE", "UPDATE_ROLLBACK_COMPLETE":
					deleteStack(s, CF)
				case "CREATE_FAILED", "DELETE_FAILED", "ROLLBACK_FAILED", "UPDATE_ROLLBACK_FAILED":
					eres, err := CF.DescribeStackEvents(&cloudformation.DescribeStackEventsInput{
						StackName: aws.String(s.Name),
					})
					if err != nil {
						return err
					}

					for _, event := range eres.StackEvents {
						if strings.HasSuffix(*event.ResourceStatus, "FAILED") {
							fmt.Printf("Failed: %s: %s\n", *event.LogicalResourceId, *event.ResourceStatusReason)
						}
					}

					deleteStack(s, CF)
				case "DELETE_IN_PROGRESS":
					displayProgress(s.StackName, CF, true)
				default:
					displayProgress(s.StackName, CF, true)
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
func describeRackStacks(rackName string, CF *cloudformation.CloudFormation) (Stacks, error) {
	apps := []Stack{}
	rack := []Stack{}
	resources := []Stack{}

	err := CF.DescribeStacksPages(&cloudformation.DescribeStacksInput{},
		func(page *cloudformation.DescribeStacksOutput, lastPage bool) bool {
			for _, stack := range page.Stacks {
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
					Outputs:   outputs,
				}

				// collect stacks that are explicitly related to the rack
				if tags["Rack"] == rackName {
					switch tags["Type"] {
					case "app":
						apps = append(apps, s)
					case "service":
						s.Type = "resource"
						fallthrough
					case "resource":
						resources = append(resources, s)
					}
				}

				// collect stack that is explicitly the rack
				if *stack.StackName == rackName && outputs["Dashboard"] != "" {
					s.Type = "rack"
					rack = append(rack, s)
				}
			}

			return true
		})

	if err != nil {
		return Stacks{}, err
	}

	return Stacks{
		Apps:      apps,
		Rack:      rack,
		Resources: resources,
	}, nil
}

func describeStackBuckets(stack string, CF *cloudformation.CloudFormation) ([]string, error) {
	buckets := []string{}

	rres, err := CF.DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(stack),
	})
	if err != nil {
		return nil, err
	}

	for _, resource := range rres.StackResources {
		if *resource.ResourceType == "AWS::S3::Bucket" {
			if resource.PhysicalResourceId != nil {
				buckets = append(buckets, *resource.PhysicalResourceId)
			}
		}
	}

	return buckets, nil
}

func (stacks Stacks) all() []Stack {
	s := []Stack{}
	s = append(s, stacks.Resources...)
	s = append(s, stacks.Apps...)
	s = append(s, stacks.Rack...)
	return s
}

func (stacks Stacks) byType(t string) []Stack {
	switch t {
	case "app":
		return stacks.Apps
	case "rack":
		return stacks.Rack
	case "resource", "service":
		return stacks.Resources
	default:
		return []Stack{}
	}
}
