package sparta

import (
	"archive/zip"
	"fmt"
	"io"

	"net/http"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws/session"
)

const userdataResourceContents = `
{
  "Hello" : "World",
}`

func helloZipLambda(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello World")
}

func archiveHook(context map[string]interface{},
	serviceName string,
	zipWriter *zip.Writer,
	awsSession *session.Session,
	noop bool,
	logger *logrus.Logger) error {

	logger.Info("Adding userResource")
	resourceFileName := "userResource.json"
	binaryWriter, binaryWriterErr := zipWriter.Create(resourceFileName)
	if nil != binaryWriterErr {
		return binaryWriterErr
	}
	userdataReader := strings.NewReader(userdataResourceContents)
	_, copyErr := io.Copy(binaryWriter, userdataReader)
	return copyErr
}

func ExampleWorkflowHooks() {
	workflowHooks := WorkflowHooks{
		Archive: archiveHook,
	}

	var lambdaFunctions []*LambdaAWSInfo
	helloWorldLambda := HandleAWSLambda("PreexistingAWSLambdaRoleName",
		http.HandlerFunc(helloZipLambda),
		nil)
	lambdaFunctions = append(lambdaFunctions, helloWorldLambda)
	MainEx("HelloWorldArchiveHook",
		"Description for Hello World HelloWorldArchiveHook",
		lambdaFunctions,
		nil,
		nil,
		&workflowHooks,
		false)
}
