package sparta

import (
	"encoding/json"
	"net/http"

	"github.com/Sirupsen/logrus"
)

const snsTopic = "arn:aws:sns:us-west-2:123412341234:mySNSTopic"

func snsProcessor(event *json.RawMessage, context *LambdaContext, w http.ResponseWriter, logger *logrus.Logger) {
	logger.WithFields(logrus.Fields{
		"RequestID": context.AWSRequestID,
	}).Info("SNSEvent")
	logger.Info("Event data: ", string(*event))
}

func ExampleSNSPermission() {
	var lambdaFunctions []*LambdaAWSInfo

	snsLambda := NewLambda(IAMRoleDefinition{}, snsProcessor, nil)
	snsLambda.Permissions = append(snsLambda.Permissions, SNSPermission{
		BasePermission: BasePermission{
			SourceArn: snsTopic,
		},
	})
	lambdaFunctions = append(lambdaFunctions, snsLambda)
	Main("SNSLambdaApp", "Registers for SNS events", lambdaFunctions, nil, nil)
}
