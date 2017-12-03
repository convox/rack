package sparta

import (
	"encoding/json"
	"net/http"

	"github.com/Sirupsen/logrus"
)

const s3Bucket = "arn:aws:sns:us-west-2:123412341234:myBucket"

func s3LambdaProcessor(event *json.RawMessage, context *LambdaContext, w http.ResponseWriter, logger *logrus.Logger) {
	logger.WithFields(logrus.Fields{
		"RequestID": context.AWSRequestID,
	}).Info("S3Event")

	logger.Info("Event data: ", string(*event))
}

func ExampleS3Permission() {
	var lambdaFunctions []*LambdaAWSInfo
	// Define the IAM role
	roleDefinition := IAMRoleDefinition{}
	roleDefinition.Privileges = append(roleDefinition.Privileges, IAMRolePrivilege{
		Actions: []string{"s3:GetObject",
			"s3:PutObject"},
		Resource: s3Bucket,
	})
	// Create the Lambda
	s3Lambda := NewLambda(IAMRoleDefinition{}, s3LambdaProcessor, nil)

	// Add a Permission s.t. the Lambda function automatically registers for S3 events
	s3Lambda.Permissions = append(s3Lambda.Permissions, S3Permission{
		BasePermission: BasePermission{
			SourceArn: s3Bucket,
		},
		Events: []string{"s3:ObjectCreated:*", "s3:ObjectRemoved:*"},
	})

	lambdaFunctions = append(lambdaFunctions, s3Lambda)
	Main("S3LambdaApp", "Registers for S3 events", lambdaFunctions, nil, nil)
}
