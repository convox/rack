package sparta

import (
	"io/ioutil"
	"net/http"

	"github.com/Sirupsen/logrus"
)

const snsTopic = "arn:aws:sns:us-west-2:123412341234:mySNSTopic"

func snsProcessor(w http.ResponseWriter, r *http.Request) {
	logger, _ := r.Context().Value(ContextKeyLogger).(*logrus.Logger)
	lambdaContext, _ := r.Context().Value(ContextKeyLambdaContext).(*LambdaContext)

	logger.WithFields(logrus.Fields{
		"RequestID": lambdaContext.AWSRequestID,
	}).Info("SNSEvent")
	event, _ := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	logger.Info("Event data: ", string(event))
}

func ExampleSNSPermission() {
	var lambdaFunctions []*LambdaAWSInfo

	snsLambda := HandleAWSLambda(LambdaName(snsProcessor),
		http.HandlerFunc(snsProcessor),
		IAMRoleDefinition{})
	snsLambda.Permissions = append(snsLambda.Permissions, SNSPermission{
		BasePermission: BasePermission{
			SourceArn: snsTopic,
		},
	})
	lambdaFunctions = append(lambdaFunctions, snsLambda)
	Main("SNSLambdaApp", "Registers for SNS events", lambdaFunctions, nil, nil)
}
