package sparta

import (
	"net/http"

	"github.com/Sirupsen/logrus"
)

func cloudWatchLogsProcessor(w http.ResponseWriter, r *http.Request) {
	logger, _ := r.Context().Value(ContextKeyLogger).(*logrus.Logger)
	lambdaContext, _ := r.Context().Value(ContextKeyLambdaContext).(*LambdaContext)
	logger.WithFields(logrus.Fields{
		"RequestID": lambdaContext.AWSRequestID,
	}).Info("CloudWatch log event")
	logger.Info("CloudWatch Log event received")
}

func ExampleCloudWatchLogsPermission() {
	var lambdaFunctions []*LambdaAWSInfo

	cloudWatchLogsLambda := HandleAWSLambda(LambdaName(cloudWatchLogsProcessor),
		http.HandlerFunc(cloudWatchLogsProcessor),
		IAMRoleDefinition{})

	cloudWatchLogsPermission := CloudWatchLogsPermission{}
	cloudWatchLogsPermission.Filters = make(map[string]CloudWatchLogsSubscriptionFilter, 1)
	cloudWatchLogsPermission.Filters["MyFilter"] = CloudWatchLogsSubscriptionFilter{
		LogGroupName: "/aws/lambda/*",
	}
	cloudWatchLogsLambda.Permissions = append(cloudWatchLogsLambda.Permissions, cloudWatchLogsPermission)

	lambdaFunctions = append(lambdaFunctions, cloudWatchLogsLambda)
	Main("CloudWatchLogs", "Registers for CloudWatch Logs", lambdaFunctions, nil, nil)
}
