package models

import (
	"os"

	aaws "github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/cloudformation"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/cloudwatch"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/dynamodb"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/ec2"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/kinesis"
)

var SortableTime = "20060102.150405.000000000"

var (
	auth = aaws.Creds(os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"), "")
)

var (
	CloudFormation = cloudformation.New(auth, os.Getenv("AWS_REGION"), nil)
	Cloudwatch     = cloudwatch.New(auth, os.Getenv("AWS_REGION"), nil)
	DynamoDB       = dynamodb.New(auth, os.Getenv("AWS_REGION"), nil)
	EC2            = ec2.New(auth, os.Getenv("AWS_REGION"), nil)
	Kinesis        = kinesis.New(auth, os.Getenv("AWS_REGION"), nil)
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
