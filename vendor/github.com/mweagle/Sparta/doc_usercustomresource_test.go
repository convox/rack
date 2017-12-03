package sparta

import (
	"encoding/json"
	"fmt"
	"net/http"

	gocf "github.com/crewjam/go-cloudformation"

	"github.com/Sirupsen/logrus"
)

// Standard AWS λ function
func helloWorld(event *json.RawMessage,
	context *LambdaContext,
	w http.ResponseWriter,
	logger *logrus.Logger) {

	configuration, _ := Discover()

	logger.WithFields(logrus.Fields{
		"Discovery": configuration,
	}).Info("Custom resource request")

	fmt.Fprint(w, "Hello World")
}

// User defined λ-backed CloudFormation CustomResource
func userDefinedCustomResource(requestType string,
	stackID string,
	properties map[string]interface{},
	logger *logrus.Logger) (map[string]interface{}, error) {

	var results = map[string]interface{}{
		"CustomResourceResult": "Victory!",
	}
	return results, nil
}

func ExampleLambdaAWSInfo_RequireCustomResource() {

	lambdaFn := NewLambda(IAMRoleDefinition{},
		helloWorld,
		nil)

	cfResName, _ := lambdaFn.RequireCustomResource(IAMRoleDefinition{},
		userDefinedCustomResource,
		nil,
		nil)

	lambdaFn.Decorator = func(serviceName string,
		lambdaResourceName string,
		lambdaResource gocf.LambdaFunction,
		resourceMetadata map[string]interface{},
		S3Bucket string,
		S3Key string,
		template *gocf.Template,
		logger *logrus.Logger) error {

		// Pass CustomResource outputs to the λ function
		resourceMetadata["CustomResource"] = gocf.GetAtt(cfResName, "CustomResourceResult")
		return nil
	}

	var lambdaFunctions []*LambdaAWSInfo
	lambdaFunctions = append(lambdaFunctions, lambdaFn)

	Main("SpartaUserCustomResource",
		"Uses a user-defined CloudFormation CustomResource",
		lambdaFunctions,
		nil,
		nil)
}
