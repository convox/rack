package sparta

import (
	"encoding/json"
	"net/http"

	"github.com/Sirupsen/logrus"
)

// NOTE: your application MUST use `package main` and define a `main()` function.  The
// example text is to make the documentation compatible with godoc.

func echoAPIGatewayHTTPEvent(w http.ResponseWriter, r *http.Request) {
	logger, _ := r.Context().Value(ContextKeyLogger).(*logrus.Logger)
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	var lambdaEvent APIGatewayLambdaJSONEvent
	err := decoder.Decode(&lambdaEvent)
	if err != nil {
		logger.Error("Failed to unmarshal event data: ", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	responseBody, err := json.Marshal(lambdaEvent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Header().Add("Content-Type", "application/json")
		w.Write(responseBody)
	}
}

// Should be main() in your application
func ExampleMain_apiGatewayHTTPSEvent() {

	// Create the MyEchoAPI API Gateway, with stagename /test.  The associated
	// Stage reesource will cause the API to be deployed.
	stage := NewStage("v1")
	apiGateway := NewAPIGateway("MyEchoHTTPAPI", stage)

	// Create a lambda function
	echoAPIGatewayLambdaFn := HandleAWSLambda(LambdaName(echoAPIGatewayHTTPEvent),
		http.HandlerFunc(echoAPIGatewayHTTPEvent),
		IAMRoleDefinition{})

	// Associate a URL path component with the Lambda function
	apiGatewayResource, _ := apiGateway.NewResource("/echoHelloWorld", echoAPIGatewayLambdaFn)

	// Associate 1 or more HTTP methods with the Resource.
	method, err := apiGatewayResource.NewMethod("GET", http.StatusOK)
	if err != nil {
		panic("Failed to create NewMethod")
	}
	// Whitelist query parameters that should be passed to lambda function
	method.Parameters["method.request.querystring.myKey"] = true
	method.Parameters["method.request.querystring.myOtherKey"] = true

	// Start
	Main("HelloWorldLambdaHTTPSService", "Description for Hello World HTTPS Lambda", []*LambdaAWSInfo{echoAPIGatewayLambdaFn}, apiGateway, nil)
}
