package sparta

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Sirupsen/logrus"
)

// NOTE: your application MUST use `package main` and define a `main()` function.  The
// example text is to make the documentation compatible with godoc.
// Should be main() in your application

func mainHelloWorld(event *json.RawMessage, context *LambdaContext, w http.ResponseWriter, logger *logrus.Logger) {
	fmt.Fprintf(w, "Hello World!")
}

func ExampleMain_basic() {
	var lambdaFunctions []*LambdaAWSInfo
	helloWorldLambda := NewLambda("PreexistingAWSLambdaRoleName", mainHelloWorld, nil)
	lambdaFunctions = append(lambdaFunctions, helloWorldLambda)
	Main("HelloWorldLambdaService", "Description for Hello World Lambda", lambdaFunctions, nil, nil)
}
