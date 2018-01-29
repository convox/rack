// +build lambdabinary

package sparta

// Provides NOP implementations for functions that do not need to execute
// in the Lambda context

import (
	"errors"
	"io"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/zcalusic/sysinfo"
)

// Delete is not available in the AWS Lambda binary
func Delete(serviceName string, logger *logrus.Logger) error {
	logger.Error("Delete() not supported in AWS Lambda binary")
	return errors.New("Delete not supported for this binary")
}

// Provision is not available in the AWS Lambda binary
func Provision(noop bool,
	serviceName string,
	serviceDescription string,
	lambdaAWSInfos []*LambdaAWSInfo,
	api *API,
	site *S3Site,
	s3Bucket string,
	useCGO bool,
	inplace bool,
	buildID string,
	codePipelineTrigger string,
	buildTags string,
	linkerFlags string,
	writer io.Writer,
	workflowHooks *WorkflowHooks,
	logger *logrus.Logger) error {
	logger.Error("Deploy() not supported in AWS Lambda binary")
	return errors.New("Deploy not supported for this binary")
}

// Describe is not available in the AWS Lambda binary
func Describe(serviceName string,
	serviceDescription string,
	lambdaAWSInfos []*LambdaAWSInfo,
	api *API,
	site *S3Site,
	s3BucketName string,
	buildTags string,
	linkerFlags string,
	outputWriter io.Writer,
	workflowHooks *WorkflowHooks,
	logger *logrus.Logger) error {
	logger.Error("Describe() not supported in AWS Lambda binary")
	return errors.New("Describe not supported for this binary")
}

// Explore is not available in the AWS Lambda binary
func Explore(lambdaAWSInfos []*LambdaAWSInfo,
	port int,
	logger *logrus.Logger) error {
	logger.Error("Explore() not supported in AWS Lambda binary")
	return errors.New("Explore not supported for this binary")
}

// Profile is the interactive command used to pull S3 assets locally into /tmp
// and run ppro against the cached profiles
func Profile(serviceName string,
	serviceDescription string,
	s3Bucket string,
	httpPort int,
	logger *logrus.Logger) error {
	return errors.New("Profile not supported for this binary")
}

// Support Windows development, by only requiring `syscall` in the compiled
// linux binary.  THere is a NOP impl over in sparta_xplatbuild that doesn't
// include the lambdabinary flag
func platformKill(parentProcessPID int) {
	syscall.Kill(parentProcessPID, syscall.SIGUSR2)
}

func platformLogSysInfo(logger *logrus.Logger) {
	var si sysinfo.SysInfo
	si.GetSysInfo()
	logger.WithFields(logrus.Fields{
		"systemInfo": si,
	}).Info("SystemInfo")
}

// RegisterCodePipelineEnvironment is not available during lambda execution
func RegisterCodePipelineEnvironment(environmentName string, environmentVariables map[string]string) error {
	return nil
}
