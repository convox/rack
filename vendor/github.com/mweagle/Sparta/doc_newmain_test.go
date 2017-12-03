package sparta

import (
	"fmt"
	"net/http"
)

// NOTE: your application MUST use `package main` and define a `main()` function.  The
// example text is to make the documentation compatible with godoc.
// Should be main() in your application

func mainHelloWorld(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
}

func ExampleMain_basic() {
	var lambdaFunctions []*LambdaAWSInfo
	helloWorldLambda := HandleAWSLambda("PreexistingAWSLambdaRoleName",
		http.HandlerFunc(mainHelloWorld),
		IAMRoleDefinition{})

	lambdaFunctions = append(lambdaFunctions, helloWorldLambda)
	Main("HelloWorldLambdaService", "Description for Hello World Lambda", lambdaFunctions, nil, nil)
}
