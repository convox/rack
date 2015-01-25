package models

import (
	"os"

	caws "github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/aws"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/dynamodb"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/ec2"

	gaws "github.com/convox/kernel/web/Godeps/_workspace/src/github.com/goamz/goamz/aws"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/goamz/goamz/cloudformation"
)

var SortableTime = "20060102.150405.000000000"

var (
	cauth = caws.Auth{AccessKey: os.Getenv("AWS_ACCESS"), SecretKey: os.Getenv("AWS_SECRET")}
	gauth = gaws.Auth{AccessKey: os.Getenv("AWS_ACCESS"), SecretKey: os.Getenv("AWS_SECRET")}
)

var (
	CloudFormation = cloudformation.New(gauth, gaws.Regions[os.Getenv("AWS_REGION")])
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
