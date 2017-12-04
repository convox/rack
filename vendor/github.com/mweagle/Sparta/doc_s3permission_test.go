package sparta

import (
	"io/ioutil"
	"net/http"

	"github.com/Sirupsen/logrus"
)

const s3Bucket = "arn:aws:sns:us-west-2:123412341234:myBucket"

func s3LambdaProcessor(w http.ResponseWriter, r *http.Request) {
	logger, _ := r.Context().Value(ContextKeyLogger).(*logrus.Logger)
	lambdaContext, _ := r.Context().Value(ContextKeyLambdaContext).(*LambdaContext)

	logger.WithFields(logrus.Fields{
		"RequestID": lambdaContext.AWSRequestID,
	}).Info("S3Event")
	event, _ := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	logger.Info("Event data: ", string(event))
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
	s3Lambda := HandleAWSLambda(LambdaName(s3LambdaProcessor),
		http.HandlerFunc(s3LambdaProcessor),
		IAMRoleDefinition{})

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
