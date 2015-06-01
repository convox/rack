package formation

import (
	"fmt"
	"os"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws/credentials"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ecs"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/lambda"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/sqs"
)

func Credentials(req *Request) *credentials.Credentials {
	if req != nil {
		if access, ok := req.ResourceProperties["AccessId"].(string); ok && access != "" {
			if secret, ok := req.ResourceProperties["SecretAccessKey"].(string); ok && secret != "" {
				fmt.Printf("access = %+v\n", access)
				fmt.Printf("secret = %+v\n", secret)
				return credentials.NewStaticCredentials(access, secret, "")
			}
		}
	}

	if os.Getenv("AWS_ACCESS") != "" {
		return credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"), "")
	}

	// return credentials.NewCredentials(&credentials.EC2RoleProvider{})
	return credentials.NewEnvCredentials()
}

func Region(req *Request) string {
	if req != nil {
		if region, ok := req.ResourceProperties["Region"].(string); ok && region != "" {
			return region
		}
	}

	return os.Getenv("AWS_REGION")
}

func Lambda(req Request) *lambda.Lambda {
	return lambda.New(&aws.Config{
		Credentials: Credentials(&req),
		Region:      Region(&req),
	})
}

func ECS(req Request) *ecs.ECS {
	return ecs.New(&aws.Config{
		Credentials: Credentials(&req),
		Region:      Region(&req),
	})
}

func SQS() *sqs.SQS {
	return sqs.New(&aws.Config{
		Credentials: Credentials(nil),
		Region:      Region(nil),
	})
}
