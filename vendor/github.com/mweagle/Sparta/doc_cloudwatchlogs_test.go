package sparta

import (
	"encoding/json"
	"net/http"

	"github.com/Sirupsen/logrus"
)

func cloudWatchLogsProcessor(event *json.RawMessage, context *LambdaContext, w http.ResponseWriter, logger *logrus.Logger) {
	logger.WithFields(logrus.Fields{
		"RequestID": context.AWSRequestID,
	}).Info("CloudWatch log event")
	logger.Info("CloudWatch Log event data: ", string(*event))
}

func ExampleCloudWatchLogsPermission() {
	var lambdaFunctions []*LambdaAWSInfo

	cloudWatchLogsLambda := NewLambda(IAMRoleDefinition{}, cloudWatchLogsProcessor, nil)

	cloudWatchLogsPermission := CloudWatchLogsPermission{}
	cloudWatchLogsPermission.Filters = make(map[string]CloudWatchLogsSubscriptionFilter, 1)
	cloudWatchLogsPermission.Filters["MyFilter"] = CloudWatchLogsSubscriptionFilter{
		LogGroupName: "/aws/lambda/*",
	}
	cloudWatchLogsLambda.Permissions = append(cloudWatchLogsLambda.Permissions, cloudWatchLogsPermission)

	lambdaFunctions = append(lambdaFunctions, cloudWatchLogsLambda)
	Main("CloudWatchLogs", "Registers for CloudWatch Logs", lambdaFunctions, nil, nil)
}
