package cgo

import (
	"fmt"
	"runtime"

	sparta "github.com/mweagle/Sparta"
)

// Main defines the primary handler for transforming an application into a Sparta package.  The
// serviceName is used to uniquely identify your service within a region and will
// be used for subsequent updates.  For provisioning, ensure that you've
// properly configured AWS credentials for the golang SDK.
// See http://docs.aws.amazon.com/sdk-for-go/api/aws/defaults.html#DefaultChainCredentials-constant
// for more information.
func Main(serviceName string,
	serviceDescription string,
	lambdaAWSInfos []*sparta.LambdaAWSInfo,
	api *sparta.API,
	site *sparta.S3Site) error {

	_, file, _, ok := runtime.Caller(1)
	if !ok {
		return fmt.Errorf("Failed to determine caller site for CGO")
	}
	return cgoMain(file,
		serviceName,
		serviceDescription,
		lambdaAWSInfos,
		api,
		site,
		nil)
}

// MainEx is the CGO enabled signature compatible version sparta.MainEx
// function that will attempt to rewrite the input source to be a CGO
// compliant library
func MainEx(serviceName string,
	serviceDescription string,
	lambdaAWSInfos []*sparta.LambdaAWSInfo,
	api *sparta.API,
	site *sparta.S3Site,
	workflowHooks *sparta.WorkflowHooks) error {

	_, file, _, ok := runtime.Caller(1)
	if !ok {
		return fmt.Errorf("Failed to determine caller site for CGO")
	}

	// If this is a "normal" execution, let's try and
	// use the existing golang functions
	return cgoMain(file,
		serviceName,
		serviceDescription,
		lambdaAWSInfos,
		api,
		site,
		nil)
}
