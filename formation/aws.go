package formation

import (
	"os"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws/credentials"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/ecs"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/lambda"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/sqs"
)

func Credentials() *credentials.Credentials {
	if os.Getenv("AWS_ACCESS") != "" {
		return credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"), "")
	}

	// return credentials.NewCredentials(&credentials.EC2RoleProvider{})
	return credentials.NewEnvCredentials()
}

func Lambda() *lambda.Lambda {
	return lambda.New(&aws.Config{
		Credentials: Credentials(),
		Region:      os.Getenv("AWS_REGION"),
	})
}

func ECS() *ecs.ECS {
	return ecs.New(&aws.Config{
		Credentials: Credentials(),
		Region:      os.Getenv("AWS_REGION"),
	})
}

func SQS() *sqs.SQS {
	return sqs.New(&aws.Config{
		Credentials: Credentials(),
		Region:      os.Getenv("AWS_REGION"),
	})
}
