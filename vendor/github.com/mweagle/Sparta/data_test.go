package sparta

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Sirupsen/logrus"
)

const LambdaExecuteARN = "LambdaExecutor"
const s3BucketSourceArn = "arn:aws:s3:::sampleBucket"
const snsTopicSourceArn = "arn:aws:sns:us-west-2:000000000000:someTopic"
const dynamoDBTableArn = "arn:aws:dynamodb:us-west-2:000000000000:table/sampleTable"

func mockLambda1(event *json.RawMessage, context *LambdaContext, w http.ResponseWriter, logger *logrus.Logger) {
	fmt.Fprintf(w, "mockLambda1!")
}

func mockLambda2(event *json.RawMessage, context *LambdaContext, w http.ResponseWriter, logger *logrus.Logger) {
	fmt.Fprintf(w, "mockLambda2!")
}

func mockLambda3(event *json.RawMessage, context *LambdaContext, w http.ResponseWriter, logger *logrus.Logger) {
	fmt.Fprintf(w, "mockLambda3!")
}

func testLambdaData() []*LambdaAWSInfo {
	var lambdaFunctions []*LambdaAWSInfo

	//////////////////////////////////////////////////////////////////////////////
	// Lambda function 1
	lambdaFn := NewLambda(LambdaExecuteARN, mockLambda1, nil)
	lambdaFn.Permissions = append(lambdaFn.Permissions, S3Permission{
		BasePermission: BasePermission{
			SourceArn: s3BucketSourceArn,
		},
		// Event Filters are defined at
		// http://docs.aws.amazon.com/AmazonS3/latest/dev/NotificationHowTo.html
		Events: []string{"s3:ObjectCreated:*", "s3:ObjectRemoved:*"},
	})

	lambdaFn.Permissions = append(lambdaFn.Permissions, SNSPermission{
		BasePermission: BasePermission{
			SourceArn: snsTopicSourceArn,
		},
	})

	lambdaFn.EventSourceMappings = append(lambdaFn.EventSourceMappings, &EventSourceMapping{
		StartingPosition: "TRIM_HORIZON",
		EventSourceArn:   dynamoDBTableArn,
		BatchSize:        10,
	})

	lambdaFunctions = append(lambdaFunctions, lambdaFn)

	//////////////////////////////////////////////////////////////////////////////
	// Lambda function 2
	lambdaFunctions = append(lambdaFunctions, NewLambda(LambdaExecuteARN, mockLambda2, nil))

	//////////////////////////////////////////////////////////////////////////////
	// Lambda function 3
	// https://github.com/mweagle/Sparta/pull/1
	lambdaFn3 := NewLambda(LambdaExecuteARN, mockLambda3, nil)
	lambdaFn3.Permissions = append(lambdaFn3.Permissions, SNSPermission{
		BasePermission: BasePermission{
			SourceArn: snsTopicSourceArn,
		},
	})
	lambdaFunctions = append(lambdaFunctions, lambdaFn3)

	return lambdaFunctions
}
