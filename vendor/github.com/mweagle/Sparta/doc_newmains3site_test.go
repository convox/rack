package sparta

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Sirupsen/logrus"
)

// NOTE: your application MUST use `package main` and define a `main()` function.  The
// example text is to make the documentation compatible with godoc.
func echoS3SiteAPIGatewayEvent(w http.ResponseWriter, r *http.Request) {
	logger, _ := r.Context().Value(ContextKeyLogger).(*logrus.Logger)
	bytes, _ := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	logger.Info("Hello World: ", string(bytes))
	fmt.Fprint(w, string(bytes))
}

// Should be main() in your application
func ExampleMain_s3Site() {

	// Create an API Gateway
	apiStage := NewStage("v1")
	apiGateway := NewAPIGateway("SpartaS3Site", apiStage)
	apiGateway.CORSEnabled = true

	// Create a lambda function
	echoS3SiteAPIGatewayEventLambdaFn := HandleAWSLambda(LambdaName(echoS3SiteAPIGatewayEvent),
		http.HandlerFunc(echoS3SiteAPIGatewayEvent),
		IAMRoleDefinition{})
	apiGatewayResource, _ := apiGateway.NewResource("/hello", echoS3SiteAPIGatewayEventLambdaFn)
	_, err := apiGatewayResource.NewMethod("GET", http.StatusOK)
	if nil != err {
		panic("Failed to create GET resource")
	}
	// Create an S3 site from the contents in ./site
	s3Site, _ := NewS3Site("./site")

	// Provision everything
	Main("HelloWorldS3SiteService", "Description for S3Site", []*LambdaAWSInfo{echoS3SiteAPIGatewayEventLambdaFn}, apiGateway, s3Site)
}
