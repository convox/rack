// +build lambdabinary,noop

package cgo

import (
	"net/http"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	sparta "github.com/mweagle/Sparta"
)

////////////////////////////////////////////////////////////////////////////////
// cgoMain is the primary entrypoint for the library version
func cgoMain(callerFile string,
	serviceName string,
	serviceDescription string,
	lambdaAWSInfos []*sparta.LambdaAWSInfo,
	api *sparta.API,
	site *sparta.S3Site,
	workflowHooks *sparta.WorkflowHooks) error {
	// NOOP
	return nil
}

// LambdaHandler is the public handler that's called by the transformed
// CGO compliant userinput. Users should not need to call this function
// directly
func LambdaHandler(functionName string,
	logLevel string,
	eventJSON string,
	awsCredentials *credentials.Credentials) ([]byte, http.Header, error) {
	// NOOP
	return nil, nil, nil
}

// NewSession returns a CGO-aware AWS session that uses the Python
// credentials provided by the CGO interface.
func NewSession() *session.Session {
	// NOOP
	return nil
}
