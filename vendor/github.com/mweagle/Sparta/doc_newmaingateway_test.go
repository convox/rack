package sparta

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Sirupsen/logrus"
)

// NOTE: your application MUST use `package main` and define a `main()` function.  The
// example text is to make the documentation compatible with godoc.

func echoAPIGatewayEvent(w http.ResponseWriter, r *http.Request) {
	logger, _ := r.Context().Value(ContextKeyLogger).(*logrus.Logger)
	lambdaContext, _ := r.Context().Value(ContextKeyLambdaContext).(*LambdaContext)
	bodyData, _ := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	logger.WithFields(logrus.Fields{
		"RequestID": lambdaContext.AWSRequestID,
		"Event":     string(bodyData),
	}).Debug("Request received")
	fmt.Fprintf(w, "Hello World!")
}

// Should be main() in your application
func ExampleMain_apiGateway() {

	// Create the MyEchoAPI API Gateway, with stagename /test.  The associated
	// Stage reesource will cause the API to be deployed.
	stage := NewStage("test")
	apiGateway := NewAPIGateway("MyEchoAPI", stage)

	// Create a lambda function
	echoAPIGatewayLambdaFn := HandleAWSLambda(LambdaName(echoAPIGatewayEvent),
		http.HandlerFunc(echoAPIGatewayEvent),
		IAMRoleDefinition{})

	// Associate a URL path component with the Lambda function
	apiGatewayResource, _ := apiGateway.NewResource("/echoHelloWorld", echoAPIGatewayLambdaFn)

	// Associate 1 or more HTTP methods with the Resource.
	apiGatewayResource.NewMethod("GET", http.StatusOK)

	// After the stack is deployed, the
	// echoAPIGatewayEvent lambda function will be available at:
	// https://{RestApiID}.execute-api.{AWSRegion}.amazonaws.com/test
	//
	// The dynamically generated URL will be written to STDOUT as part of stack provisioning as in:
	//
	//	Outputs: [{
	//      Description: "API Gateway URL",
	//      OutputKey: "URL",
	//      OutputValue: "https://zdjfwrcao7.execute-api.us-west-2.amazonaws.com/test"
	//    }]
	// eg:
	// 	curl -vs https://zdjfwrcao7.execute-api.us-west-2.amazonaws.com/test/echoHelloWorld

	// Start
	Main("HelloWorldLambdaService", "Description for Hello World Lambda", []*LambdaAWSInfo{echoAPIGatewayLambdaFn}, apiGateway, nil)
}
