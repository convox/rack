package sparta

import (
	"net/http"

	"github.com/Sirupsen/logrus"
)

func cloudWatchEventProcessor(w http.ResponseWriter, r *http.Request) {
	logger, _ := r.Context().Value(ContextKeyLogger).(*logrus.Logger)
	lambdaContext, _ := r.Context().Value(ContextKeyLambdaContext).(*LambdaContext)
	logger.WithFields(logrus.Fields{
		"RequestID": lambdaContext.AWSRequestID,
	}).Info("Request received")

	logger.Info("CloudWatch Event received")
}

func ExampleCloudWatchEventsPermission() {
	cloudWatchEventsLambda := HandleAWSLambda(LambdaName(cloudWatchEventProcessor),
		http.HandlerFunc(cloudWatchEventProcessor),
		IAMRoleDefinition{})

	cloudWatchEventsPermission := CloudWatchEventsPermission{}
	cloudWatchEventsPermission.Rules = make(map[string]CloudWatchEventsRule)
	cloudWatchEventsPermission.Rules["Rate5Mins"] = CloudWatchEventsRule{
		ScheduleExpression: "rate(5 minutes)",
	}
	cloudWatchEventsPermission.Rules["EC2Activity"] = CloudWatchEventsRule{
		EventPattern: map[string]interface{}{
			"source":      []string{"aws.ec2"},
			"detail-type": []string{"EC2 Instance State-change Notification"},
		},
	}
	cloudWatchEventsLambda.Permissions = append(cloudWatchEventsLambda.Permissions, cloudWatchEventsPermission)
	var lambdaFunctions []*LambdaAWSInfo
	lambdaFunctions = append(lambdaFunctions, cloudWatchEventsLambda)
	Main("CloudWatchLogs", "Registers for CloudWatch Logs", lambdaFunctions, nil, nil)
}
