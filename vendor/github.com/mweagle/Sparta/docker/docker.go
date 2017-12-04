package docker

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/sts"
)

////////////////////////////////////////////////////////////////////////////////
// CONSTANTS
////////////////////////////////////////////////////////////////////////////////

const (
	// BinaryNameArgument is the argument provided to docker build that
	// supplies the local statically built Go binary
	BinaryNameArgument = "SPARTA_DOCKER_BINARY"
)

func runOSCommand(cmd *exec.Cmd, logger *logrus.Logger) error {
	logger.WithFields(logrus.Fields{
		"Arguments": cmd.Args,
		"Dir":       cmd.Dir,
		"Path":      cmd.Path,
		"Env":       cmd.Env,
	}).Debug("Running Command")
	outputWriter := logger.Writer()
	defer outputWriter.Close()
	cmd.Stdout = outputWriter
	cmd.Stderr = outputWriter
	return cmd.Run()
}

// BuildDockerImage creates the smallest docker image for this Golang binary
// using the serviceName as the image name and including the supplied tags
func BuildDockerImage(serviceName string,
	dockerFilepath string,
	tags *map[string]string,
	logger *logrus.Logger) error {

	// BEGIN DOCKER PRECONDITIONS
	// Ensure that serviceName and tags are lowercase to make Docker happy
	var dockerErrors []string
	if nil != tags {
		for eachKey, eachValue := range *tags {
			if eachKey != strings.ToLower(eachKey) ||
				eachValue != strings.ToLower(eachValue) {
				dockerErrors = append(dockerErrors, fmt.Sprintf("--tag %s:%s MUST be lower case", eachKey, eachValue))
			}
		}
	}
	if len(dockerErrors) > 0 {
		return fmt.Errorf("Docker build errors: %s", strings.Join(dockerErrors[:], ", "))
	}
	// END DOCKER PRECONDITIONS

	// Compile this binary for minimal Docker size
	// https://blog.codeship.com/building-minimal-docker-containers-for-go-applications/
	executableOutput := fmt.Sprintf("%s-%d-docker.lambda.amd64", serviceName, time.Now().UnixNano())
	cmd := exec.Command("go",
		"build",
		"-a",
		"-installsuffix",
		"cgo",
		"-o",
		executableOutput,
		"-tags",
		"lambdabinary",
		".")

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "CGO_ENABLED=0", "GOOS=linux")
	logger.WithFields(logrus.Fields{
		"Name": executableOutput,
	}).Info("Compiling Docker binary")
	buildErr := runOSCommand(cmd, logger)
	if nil != buildErr {
		return buildErr
	}
	defer func() {
		removeErr := os.Remove(executableOutput)
		if nil != removeErr {
			logger.WithFields(logrus.Fields{
				"Path":  executableOutput,
				"Error": removeErr,
			}).Warn("Failed to delete temporary Docker binary")
		}
	}()

	// ARG SPARTA_DOCKER_BINARY reference s.t. we can supply the binary
	// name to the build..
	// We need to build the static binary s.t. we can add it to the Docker container...
	// Build the image...
	var dockerArgs []string
	dockerArgs = append(dockerArgs,
		"build",
		"--build-arg",
		fmt.Sprintf("%s=%s", BinaryNameArgument, executableOutput))

	if "" != dockerFilepath {
		dockerArgs = append(dockerArgs, "--file", dockerFilepath)
	}
	// Add the latest tag
	// dockerArgs = append(dockerArgs, "--tag", fmt.Sprintf("sparta/%s:latest", serviceName))

	if nil != tags {
		for eachKey, eachValue := range *tags {
			dockerArgs = append(dockerArgs, "--tag", fmt.Sprintf("%s:%s",
				strings.ToLower(eachKey),
				strings.ToLower(eachValue)))
		}
	}
	dockerArgs = append(dockerArgs, ".")
	dockerCmd := exec.Command("docker", dockerArgs...)
	return runOSCommand(dockerCmd, logger)
}

// PushDockerImageToECR pushes a local Docker image to an ECR repository
func PushDockerImageToECR(localImageTag string,
	ecrRepoName string,
	awsSession *session.Session,
	logger *logrus.Logger) (string, error) {

	stsSvc := sts.New(awsSession)
	ecrSvc := ecr.New(awsSession)

	// 1. Get the caller identity s.t. we can get the ECR URL which includes the
	// account name
	stsIdentityOutput, stsIdentityErr := stsSvc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if nil != stsIdentityErr {
		return "", stsIdentityErr
	}

	// 2. Create the URL to which we're going to do the push
	localImageTagParts := strings.Split(localImageTag, ":")
	ecrTagValue := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s",
		*stsIdentityOutput.Account,
		*awsSession.Config.Region,
		ecrRepoName,
		localImageTagParts[len(localImageTagParts)-1])

	// 3. Tag the local image with the ECR tag
	dockerTagCmd := exec.Command("docker", "tag", localImageTag, ecrTagValue)
	dockerTagCmdErr := runOSCommand(dockerTagCmd, logger)
	if nil != dockerTagCmdErr {
		return "", dockerTagCmdErr
	}

	// 4. Push the image - if that fails attempt to reauthorize with the docker
	// client and try again
	var pushError error
	dockerPushCmd := exec.Command("docker", "push", ecrTagValue)
	pushError = runOSCommand(dockerPushCmd, logger)
	if nil != pushError {
		logger.WithFields(logrus.Fields{
			"Error": pushError,
		}).Info("ECR push failed - reauthorizing")
		ecrAuthTokenResult, ecrAuthTokenResultErr := ecrSvc.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})
		if nil != ecrAuthTokenResultErr {
			pushError = ecrAuthTokenResultErr
		} else {
			authData := ecrAuthTokenResult.AuthorizationData[0]
			authToken, authTokenErr := base64.StdEncoding.DecodeString(*authData.AuthorizationToken)
			if nil != authTokenErr {
				pushError = authTokenErr
			} else {
				authTokenString := string(authToken)
				authTokenParts := strings.Split(authTokenString, ":")
				dockerURL := fmt.Sprintf("https://%s.dkr.ecr.%s.amazonaws.com",
					*stsIdentityOutput.Account,
					*awsSession.Config.Region)
				dockerLoginCmd := exec.Command("docker",
					"login",
					"-u",
					authTokenParts[0],
					"-p",
					authTokenParts[1],
					"-e",
					"none",
					dockerURL)
				dockerLoginCmdErr := runOSCommand(dockerLoginCmd, logger)
				if nil != dockerLoginCmdErr {
					pushError = dockerLoginCmdErr
				} else {
					// Try it again...
					dockerRetryPushCmd := exec.Command("docker", "push", ecrTagValue)
					dockerRetryPushCmdErr := runOSCommand(dockerRetryPushCmd, logger)
					pushError = dockerRetryPushCmdErr
				}
			}
		}
	}
	return ecrTagValue, pushError
}
