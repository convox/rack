package aws

import (
	"fmt"
	"os"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sns"
)

var (
	NotificationTopic = os.Getenv("NOTIFICATION_TOPIC")
	CustomTopic       = os.Getenv("CUSTOM_TOPIC")
	SortableTime      = "20060102.150405.000000000"
	ValidAppName      = regexp.MustCompile(`\A[a-zA-Z][-a-zA-Z0-9]{3,29}\z`)
)

type AWSProvider struct {
	Region   string
	Endpoint string
	Access   string
	Secret   string
	Token    string

	Cache bool
}

// NewProvider returns the AWS provider
func NewProvider(region, endpoint, access, secret, token string) *AWSProvider {
	return &AWSProvider{
		Region:   region,
		Endpoint: endpoint,
		Access:   access,
		Secret:   secret,
		Token:    token,
		Cache:    true,
	}
}

/** services ****************************************************************************************/

func (p *AWSProvider) config() *aws.Config {
	config := &aws.Config{
		Credentials: credentials.NewStaticCredentials(p.Access, p.Secret, p.Token),
	}

	if p.Region != "" {
		config.Region = aws.String(p.Region)
	}

	if p.Endpoint != "" {
		config.Endpoint = aws.String(p.Endpoint)
	}

	if os.Getenv("DEBUG") != "" {
		config.WithLogLevel(aws.LogDebugWithHTTPBody)
	}

	return config
}

func (p *AWSProvider) acm() *acm.ACM {
	return acm.New(session.New(), p.config())
}

func (p *AWSProvider) cloudformation() *cloudformation.CloudFormation {
	return cloudformation.New(session.New(), p.config())
}

func (p *AWSProvider) cloudwatch() *cloudwatch.CloudWatch {
	return cloudwatch.New(session.New(), p.config())
}

func (p *AWSProvider) cloudwatchlogs() *cloudwatchlogs.CloudWatchLogs {
	return cloudwatchlogs.New(session.New(), p.config())
}

func (p *AWSProvider) dynamodb() *dynamodb.DynamoDB {
	return dynamodb.New(session.New(), p.config())
}

func (p *AWSProvider) ec2() *ec2.EC2 {
	return ec2.New(session.New(), p.config())
}

func (p *AWSProvider) ecr() *ecr.ECR {
	return ecr.New(session.New(), p.config())
}

func (p *AWSProvider) ecs() *ecs.ECS {
	return ecs.New(session.New(), p.config())
}

func (p *AWSProvider) iam() *iam.IAM {
	return iam.New(session.New(), p.config())
}

// s3 returns an S3 client configured to use the path style
// (http://s3.amazonaws.com/johnsmith.net/homepage.html) vs virtual
// hosted style (http://johnsmith.net.s3.amazonaws.com/homepage.html)
// since path style is easier to test.
func (p *AWSProvider) s3() *s3.S3 {
	return s3.New(session.New(), p.config().WithS3ForcePathStyle(true))
}

func (p *AWSProvider) sns() *sns.SNS {
	return sns.New(session.New(), p.config())
}

func (p *AWSProvider) dynamoBatchDeleteItems(wrs []*dynamodb.WriteRequest, tableName string) error {

	if len(wrs) > 0 {

		if len(wrs) <= 25 {
			_, err := p.dynamodb().BatchWriteItem(&dynamodb.BatchWriteItemInput{
				RequestItems: map[string][]*dynamodb.WriteRequest{
					tableName: wrs,
				},
			})
			if err != nil {
				return err
			}

		} else {

			// if more than 25 items to delete, we have to make multiple calls
			maxLen := 25
			for i := 0; i < len(wrs); i += maxLen {
				high := i + maxLen
				if high > len(wrs) {
					high = len(wrs)
				}

				_, err := p.dynamodb().BatchWriteItem(&dynamodb.BatchWriteItemInput{
					RequestItems: map[string][]*dynamodb.WriteRequest{
						tableName: wrs[i:high],
					},
				})
				if err != nil {
					return err
				}

			}
		}
	} else {
		fmt.Println("ns=api fn=dynamoBatchDeleteItems level=info msg=\"no builds to delete\"")
	}

	return nil
}
