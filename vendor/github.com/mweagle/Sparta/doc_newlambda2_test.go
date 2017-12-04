package sparta

import (
	"fmt"
	"net/http"
)

func lambdaHelloWorld2(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
}

func ExampleNewLambda_iAMRoleDefinition() {
	roleDefinition := IAMRoleDefinition{}
	roleDefinition.Privileges = append(roleDefinition.Privileges, IAMRolePrivilege{
		Actions: []string{"s3:GetObject",
			"s3:PutObject"},
		Resource: "arn:aws:s3:::*",
	})
	helloWorldLambda := HandleAWSLambda(LambdaName(lambdaHelloWorld2),
		http.HandlerFunc(lambdaHelloWorld2),
		IAMRoleDefinition{})
	if nil != helloWorldLambda {
		fmt.Printf("Failed to create new Lambda function")
	}
}
