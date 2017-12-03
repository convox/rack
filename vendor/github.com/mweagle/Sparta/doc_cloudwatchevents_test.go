package sparta

import (
	"encoding/json"
	"net/http"

	"github.com/Sirupsen/logrus"
)

func cloudWatchEventProcessor(event *json.RawMessage,
	context *LambdaContext,
	w http.ResponseWriter,
	logger *logrus.Logger) {

	logger.WithFields(logrus.Fields{
		"RequestID": context.AWSRequestID,
	}).Info("Request received")

	logger.Info("CloudWatch Event data: ", string(*event))
}

func ExampleCloudWatchEventsPermission() {
	cloudWatchEventsLambda := NewLambda(IAMRoleDefinition{}, cloudWatchEventProcessor, nil)

	cloudWatchEventsPermission := CloudWatchEventsPermission{}
	cloudWatchEventsPermission.Rules = make(map[string]CloudWatchEventsRule, 0)
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
