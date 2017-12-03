package sparta

import (
	"fmt"
	"net/http"
)

func lambdaHelloWorld(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
}

func ExampleNewLambda_preexistingIAMRoleName() {
	helloWorldLambda := HandleAWSLambda(LambdaName(lambdaHelloWorld),
		http.HandlerFunc(lambdaHelloWorld),
		IAMRoleDefinition{})
	if nil != helloWorldLambda {
		fmt.Printf("Failed to create new Lambda function")
	}
}
