package models

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

var DescribeStacksCache = map[string]DescribeStacksResult{}

var DescribeStacksCacheTTL = 5 * time.Second

var DescribeStacksMutex = &sync.Mutex{}

type DescribeStacksResult struct {
	Name        string
	Output      *cloudformation.DescribeStacksOutput
	RequestTime time.Time
}

func DescribeStacks() (*cloudformation.DescribeStacksOutput, error) {
	return doDescribeStack(cloudformation.DescribeStacksInput{})
}

func DescribeStack(name string) (*cloudformation.DescribeStacksOutput, error) {
	return doDescribeStack(cloudformation.DescribeStacksInput{
		StackName: aws.String(name),
	})
}

func UpdateStack(req *cloudformation.UpdateStackInput) (*cloudformation.UpdateStackOutput, error) {
	if req.StackName != nil {
		name := *req.StackName
		fmt.Printf("fn=UpdateStack at=delete name=%q\n", name)

		delete(DescribeStacksCache, name)
	}

	return CloudFormation().UpdateStack(req)
}

func doDescribeStack(input cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
	log := Logger.At("doDescribeStack").Start()

	DescribeStacksMutex.Lock()
	defer DescribeStacksMutex.Unlock()

	name := "<blank>"

	if input.StackName != nil {
		name = *input.StackName
	}

	s := DescribeStacksCache[name]

	// if last request was before the TTL, or if running in the test environment, make a request
	if s.RequestTime.Before(time.Now().Add(-DescribeStacksCacheTTL)) || os.Getenv("PROVIDER") == "test" {
		log.Logf("name=%q age=%s status=miss", name, time.Since(s.RequestTime))

		var err error
		var res *cloudformation.DescribeStacksOutput
		var stacks []*cloudformation.Stack

		for {
			res, err = CloudFormation().DescribeStacks(&input)
			if err != nil {
				log.Namespace("name=%q", name).Error(err)
				return nil, err
			}

			stacks = append(stacks, res.Stacks...)

			if res.NextToken == nil {
				break
			}

			input.NextToken = res.NextToken
		}

		res.Stacks = stacks
		DescribeStacksCache[name] = DescribeStacksResult{
			Name:        name,
			Output:      res,
			RequestTime: time.Now(),
		}

		return res, err
	}

	log.Logf("name=%q age=%s status=hit", name, time.Since(s.RequestTime))
	return s.Output, nil
}
