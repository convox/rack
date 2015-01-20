package provider

import (
	"os"

	caws "github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/aws"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/dynamodb"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/ec2"

	gaws "github.com/convox/kernel/web/Godeps/_workspace/src/github.com/goamz/goamz/aws"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/goamz/goamz/cloudformation"
)

var (
	cauth = caws.Auth{AccessKey: os.Getenv("AWS_ACCESS"), SecretKey: os.Getenv("AWS_SECRET")}
	gauth = gaws.Auth{AccessKey: os.Getenv("AWS_ACCESS"), SecretKey: os.Getenv("AWS_SECRET")}
)

var (
	CloudFormation = cloudformation.New(gauth, gaws.Regions[os.Getenv("AWS_REGION")])
	DynamoDB       = dynamodb.New(cauth, caws.Regions[os.Getenv("AWS_REGION")])
	EC2            = ec2.New(cauth, caws.Regions[os.Getenv("AWS_REGION")])
)
