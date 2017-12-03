package sparta

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Sirupsen/logrus"
)

// NOTE: your application MUST use `package main` and define a `main()` function.  The
// example text is to make the documentation compatible with godoc.
func echoS3SiteAPIGatewayEvent(event *json.RawMessage,
	context *LambdaContext,
	w http.ResponseWriter,
	logger *logrus.Logger) {

	logger.Info("Hello World: ", string(*event))
	fmt.Fprint(w, string(*event))
}

// Should be main() in your application
func ExampleMain_s3Site() {

	// Create an API Gateway
	apiStage := NewStage("v1")
	apiGateway := NewAPIGateway("SpartaS3Site", apiStage)
	apiGateway.CORSEnabled = true

	// Create a lambda function
	echoS3SiteAPIGatewayEventLambdaFn := NewLambda(IAMRoleDefinition{}, echoAPIGatewayEvent, nil)
	apiGatewayResource, _ := apiGateway.NewResource("/hello", echoS3SiteAPIGatewayEventLambdaFn)
	_, err := apiGatewayResource.NewMethod("GET")
	if nil != err {
		panic("Failed to create GET resource")
	}
	// Create an S3 site from the contents in ./site
	s3Site, _ := NewS3Site("./site")

	// Provision everything
	Main("HelloWorldS3SiteService", "Description for S3Site", []*LambdaAWSInfo{echoS3SiteAPIGatewayEventLambdaFn}, apiGateway, s3Site)
}
