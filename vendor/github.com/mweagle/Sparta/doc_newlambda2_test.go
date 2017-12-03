package sparta

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Sirupsen/logrus"
)

func lambdaHelloWorld2(event *json.RawMessage, context *LambdaContext, w http.ResponseWriter, logger *logrus.Logger) {
	fmt.Fprintf(w, "Hello World!")
}

func ExampleNewLambda_iAMRoleDefinition() {
	roleDefinition := IAMRoleDefinition{}
	roleDefinition.Privileges = append(roleDefinition.Privileges, IAMRolePrivilege{
		Actions: []string{"s3:GetObject",
			"s3:PutObject"},
		Resource: "arn:aws:s3:::*",
	})
	helloWorldLambda := NewLambda(IAMRoleDefinition{}, lambdaHelloWorld2, nil)
	if nil != helloWorldLambda {
		fmt.Printf("Failed to create new Lambda function")
	}
}
