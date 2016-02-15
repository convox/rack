package models

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
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
	DescribeStacksMutex.Lock()
	defer DescribeStacksMutex.Unlock()

	name := "<blank>"

	if input.StackName != nil {
		name = *input.StackName
	}

	s := DescribeStacksCache[name]

	// if last request was before the TTL, or if running in the test environment, make a request
	if s.RequestTime.Before(time.Now().Add(-DescribeStacksCacheTTL)) || os.Getenv("AWS_REGION") == "test" {
		fmt.Printf("fn=doDescribeStack at=miss name=%q age=%s\n", name, time.Now().Sub(s.RequestTime))

		res, err := CloudFormation().DescribeStacks(&input)

		if err == nil {
			DescribeStacksCache[name] = DescribeStacksResult{
				Name:        name,
				Output:      res,
				RequestTime: time.Now(),
			}
		}

		return res, err
	}

	fmt.Printf("fn=doDescribeStack at=hit name=%q age=%s\n", name, time.Now().Sub(s.RequestTime))

	return s.Output, nil
}

func maxAppConcurrency() (int, error) {
	apps, err := ListApps()

	if err != nil {
		return 0, err
	}

	max := 0

	for _, app := range apps {
		rel, err := app.LatestRelease()

		if err != nil {
			return 0, err
		}

		if rel == nil {
			continue
		}

		m, err := LoadManifest(rel.Manifest)

		if err != nil {
			return 0, err
		}

		f, err := ListFormation(app.Name)

		if err != nil {
			return 0, err
		}

		for _, me := range m {
			if len(me.ExternalPorts()) > 0 {
				entry := f.Entry(me.Name)

				if entry != nil && entry.Count > max {
					max = entry.Count
				}
			}
		}
	}

	return max, nil
}
