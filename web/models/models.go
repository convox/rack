package models

import (
	"os"

	aaws "github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/cloudwatch"

	caws "github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/aws"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/dynamodb"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/ec2"

	gaws "github.com/convox/kernel/web/Godeps/_workspace/src/github.com/goamz/goamz/aws"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/goamz/goamz/cloudformation"
)

var SortableTime = "20060102.150405.000000000"

var (
	aauth = aaws.Creds(os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"), "")
	cauth = caws.Auth{AccessKey: os.Getenv("AWS_ACCESS"), SecretKey: os.Getenv("AWS_SECRET")}
	gauth = gaws.Auth{AccessKey: os.Getenv("AWS_ACCESS"), SecretKey: os.Getenv("AWS_SECRET")}
)

var (
	CloudFormation = cloudformation.New(gauth, gaws.Regions[os.Getenv("AWS_REGION")])
	Cloudwatch     = cloudwatch.New(aauth, os.Getenv("AWS_REGION"), nil)
	DynamoDB       = dynamodb.New(cauth, caws.Regions[os.Getenv("AWS_REGION")])
	EC2            = ec2.New(cauth, caws.Regions[os.Getenv("AWS_REGION")])
)

type Cluster struct {
	Name   string
	Status string

	CpuUsed     int
	CpuTotal    int
	MemoryUsed  int
	MemoryTotal int
	DiskUsed    int
	DiskTotal   int

	Apps Apps
}

type Clusters []Cluster

type Container struct {
	Name string

	CpuUsed     int
	CpuTotal    int
	MemoryUsed  int
	MemoryTotal int
	DiskUsed    int
	DiskTotal   int
}

type Containers []Container
