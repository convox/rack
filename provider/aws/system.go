package aws

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/api/structs"
	"github.com/xtgo/uuid"
)

func (p *AWSProvider) SystemGet() (*structs.System, error) {
	stacks, err := p.describeStacks(&cloudformation.DescribeStacksInput{
		StackName: aws.String(p.Rack),
	})
	if ae, ok := err.(awserr.Error); ok && ae.Code() == "ValidationError" {
		return nil, errorNotFound(fmt.Sprintf("%s not found", p.Rack))
	}
	if err != nil {
		return nil, err
	}
	if len(stacks) != 1 {
		return nil, fmt.Errorf("could not load stack for app: %s", p.Rack)
	}

	stack := stacks[0]
	status := humanStatus(*stack.StackStatus)
	params := stackParameters(stack)

	count, err := strconv.Atoi(params["InstanceCount"])
	if err != nil {
		return nil, err
	}

	// status precedence: (all other stack statues) > converging > running
	// check if the autoscale group is shuffling instances
	if status == "running" {

		rres, err := p.cloudformation().DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
			StackName: aws.String(p.Rack),
		})
		if err != nil {
			return nil, err
		}

		var asgName string
		for _, r := range rres.StackResources {
			if *r.LogicalResourceId == "Instances" {
				asgName = *r.PhysicalResourceId
				break
			}
		}

		asgres, err := p.autoscaling().DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{
				aws.String(asgName),
			},
		})
		if err != nil {
			return nil, err
		}

		if len(asgres.AutoScalingGroups) <= 0 {
			return nil, fmt.Errorf("scaling group %s was not found", asgName)
		}

		for _, instance := range asgres.AutoScalingGroups[0].Instances {
			if *instance.LifecycleState != "InService" {
				status = "converging"
				break
			}
		}
	}

	// Check if ECS is rescheduling services
	if status == "running" {
		lreq := &ecs.ListServicesInput{
			Cluster:    aws.String(p.Cluster),
			MaxResults: aws.Int64(10),
		}
	Loop:
		for {
			lres, err := p.ecs().ListServices(lreq)
			if err != nil {
				return nil, err
			}

			dres, err := p.ecs().DescribeServices(&ecs.DescribeServicesInput{
				Cluster:  aws.String(p.Cluster),
				Services: lres.ServiceArns,
			})
			if err != nil {
				return nil, err
			}

			for _, s := range dres.Services {
				for _, d := range s.Deployments {
					if *d.RunningCount != *d.DesiredCount {
						status = "converging"
						break Loop
					}
				}
			}

			if lres.NextToken == nil {
				break
			}

			lreq.NextToken = lres.NextToken
		}
	}

	r := &structs.System{
		Count:   count,
		Name:    p.Rack,
		Region:  p.Region,
		Status:  status,
		Type:    params["InstanceType"],
		Version: params["Version"],
	}

	return r, nil
}

// SystemLogs streams logs for the Rack
func (p *AWSProvider) SystemLogs(w io.Writer, opts structs.LogStreamOptions) error {
	system, err := p.describeStack(p.Rack)
	if err != nil {
		return err
	}

	return p.subscribeLogs(w, stackOutputs(system)["LogGroup"], opts)
}

func (p *AWSProvider) SystemProcesses(opts structs.SystemProcessesOptions) (structs.Processes, error) {
	var tasks []string
	var err error

	if opts.All {
		err := p.ecs().ListTasksPages(&ecs.ListTasksInput{
			Cluster: aws.String(p.Cluster),
		}, func(page *ecs.ListTasksOutput, lastPage bool) bool {
			for _, arn := range page.TaskArns {
				tasks = append(tasks, *arn)
			}
			return true
		})
		if err != nil {
			return nil, err
		}
	} else {
		tasks, err = p.stackTasks(p.Rack)
		if err != nil {
			return nil, err
		}
	}

	ps, err := p.taskProcesses(tasks)
	if err != nil {
		return nil, err
	}

	for i := range ps {
		if ps[i].App == "" {
			ps[i].App = p.Rack
		}
	}

	return ps, nil
}

// SystemReleases lists the latest releases of the rack
func (p *AWSProvider) SystemReleases() (structs.Releases, error) {
	req := &dynamodb.QueryInput{
		KeyConditions: map[string]*dynamodb.Condition{
			"app": {
				AttributeValueList: []*dynamodb.AttributeValue{
					{S: aws.String(p.Rack)},
				},
				ComparisonOperator: aws.String("EQ"),
			},
		},
		IndexName:        aws.String("app.created"),
		Limit:            aws.Int64(20),
		ScanIndexForward: aws.Bool(false),
		TableName:        aws.String(p.DynamoReleases),
	}

	res, err := p.dynamodb().Query(req)
	if err != nil {
		return nil, err
	}

	releases := make(structs.Releases, len(res.Items))

	for i, item := range res.Items {
		r, err := releaseFromItem(item)
		if err != nil {
			return nil, err
		}

		releases[i] = *r
	}

	return releases, nil
}

func (p *AWSProvider) SystemSave(system structs.System) error {
	typeValid := false
	// Better search method could work here if needed
	// sort.SearchString() return value doesn't indicate if string is not in slice
	for _, itype := range instanceTypes {
		if itype == system.Type {
			typeValid = true
			break
		}
	}
	if !typeValid {
		return fmt.Errorf("invalid instance type: %s", system.Type)
	}

	// FIXME
	// mac, err := maxAppConcurrency()

	// // dont scale the rack below the max concurrency plus one
	// // see formation.go for more details
	// if err == nil && r.Count < (mac+1) {
	//   return fmt.Errorf("max process concurrency is %d, can't scale rack below %d instances", mac, mac+1)
	// }

	template := fmt.Sprintf("https://convox.s3.amazonaws.com/release/%s/formation.json", system.Version)

	params := map[string]string{
		"InstanceCount": strconv.Itoa(system.Count),
		"InstanceType":  system.Type,
		"Version":       system.Version,
	}

	stack, err := p.describeStack(p.Rack)
	if err != nil {
		return err
	}

	// build a list of changes for the notification
	sp := stackParameters(stack)
	changes := map[string]string{}
	if sp["InstanceCount"] != strconv.Itoa(system.Count) {
		changes["count"] = strconv.Itoa(system.Count)
	}
	if sp["InstanceType"] != system.Type {
		changes["type"] = system.Type
	}
	if sp["Version"] != system.Version {
		changes["version"] = system.Version
	}

	// if there is a version update then record it
	if v, ok := changes["version"]; ok {
		_, err := p.dynamodb().PutItem(&dynamodb.PutItemInput{
			Item: map[string]*dynamodb.AttributeValue{
				"id":      {S: aws.String(v)},
				"app":     {S: aws.String(p.Rack)},
				"created": {S: aws.String(p.createdTime())},
			},
			TableName: aws.String(p.DynamoReleases),
		})
		if err != nil {
			return err
		}
	}

	changes["id"] = uuid.NewRandom().String()

	// notify about the update
	p.EventSend(&structs.Event{
		Action: "rack:update",
		Data:   changes,
		Status: "start",
	}, nil)

	// update the stack
	err = p.updateStack(p.Rack, template, params)
	if awsError(err) == "ValidationError" {
		switch {
		case strings.Contains(err.Error(), "No updates are to be performed"):
			return fmt.Errorf("no system updates are to be performed")
		case strings.Contains(err.Error(), "can not be updated"):
			return fmt.Errorf("system is already updating")
		}
	}

	go p.waitForSystemUpdate(p.Rack, changes)

	return err
}

// waitForSystemUpdate observes a rack stack update by verifying the rack's CF stack status
// Sends a notifcation on success or failure. This function blocks.
// TODO: this should be in provider.ReleasePromote()
func (p *AWSProvider) waitForSystemUpdate(stackName string, changes map[string]string) {
	event := &structs.Event{
		Action: "rack:update",
		Data:   changes,
	}

	waitch := make(chan error)
	go func() {
		var err error
		//we have observed stack stabalization failures take up to 3 hours
		for i := 0; i < 3; i++ {
			err = p.cloudformation().WaitUntilStackUpdateComplete(&cloudformation.DescribeStacksInput{
				StackName: aws.String(stackName),
			})
			if err != nil {
				if err.Error() == "exceeded 120 wait attempts" {
					continue
				}
			}
			break
		}
		waitch <- err
	}()

	for {
		select {
		case err := <-waitch:
			if err == nil {
				event.Status = "success"
				p.EventSend(event, nil)
				return
			}

			if err != nil && err.Error() == "exceeded 120 wait attempts" {
				p.EventSend(event, fmt.Errorf("couldn't determine rack update status, timed out"))
				fmt.Println(fmt.Errorf("couldn't determine rack update status, timed out"))
				return
			}

			resp, err := p.cloudformation().DescribeStacks(&cloudformation.DescribeStacksInput{
				StackName: aws.String(stackName),
			})
			if err != nil {
				p.EventSend(event, fmt.Errorf("unable to check stack status: %s", err))
				fmt.Println(fmt.Errorf("unable to check stack status: %s", err))
				return
			}

			if len(resp.Stacks) < 1 {
				p.EventSend(event, fmt.Errorf("rack stack was not found: %s", stackName))
				fmt.Println(fmt.Errorf("rack stack was not found: %s", stackName))
				return
			}

			se, err := p.cloudformation().DescribeStackEvents(&cloudformation.DescribeStackEventsInput{
				StackName: aws.String(stackName),
			})
			if err != nil {
				p.EventSend(event, fmt.Errorf("unable to check stack events: %s", err))
				fmt.Println(fmt.Errorf("unable to check stack events: %s", err))
				return
			}

			var lastEvent *cloudformation.StackEvent

			for _, e := range se.StackEvents {
				switch *e.ResourceStatus {
				case "UPDATE_FAILED", "DELETE_FAILED", "CREATE_FAILED":
					lastEvent = e
					break
				}
			}

			ee := fmt.Errorf("unable to determine rack update error")
			if lastEvent != nil {
				ee = fmt.Errorf(
					"[%s:%s] [%s]: %s",
					*lastEvent.ResourceType,
					*lastEvent.LogicalResourceId,
					*lastEvent.ResourceStatus,
					*lastEvent.ResourceStatusReason,
				)
			}

			p.EventSend(event, fmt.Errorf("Rack Update failed - %s", ee.Error()))
		}
	}
}
