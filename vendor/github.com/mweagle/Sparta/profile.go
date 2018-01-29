package sparta

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/Sirupsen/logrus"
)

type profileLambdaDecorator func(stackName string, info *LambdaAWSInfo, S3Bucket string, logger *logrus.Logger) error

var profileDecorator profileLambdaDecorator

const (
	cpuProfileName = "cpu"

	// Name of the stack, published as env var
	envVarStackName = "SPARTA_STACK_NAME"
	// Stack instance id, published as env var
	envVarStackInstanceID = "SPARTA_STACK_INSTANCE_ID"
	// Bucket to use to store profile snapshots, published as env var
	envVarProfileBucketName = "SPARTA_PROFILE_BUCKET_NAME"
)

var profileTypes = []string{
	cpuProfileName,
	"goroutine",
	"threadcreate",
	"heap",
	"block",
	"mutex",
}

func profileSnapshotRootKeypath(stackName string) string {
	return path.Join("sparta", "pprof", stackName, "profiles")
}

func profileSnapshotRootKeypathForType(profileType string, stackName string) string {
	return path.Join(profileSnapshotRootKeypath(stackName), profileType)
}

func cacheDirectoryForProfileType(profileType string, stackName string) string {
	return filepath.Join(ScratchDirectory, "profiles", stackName, profileType)
}

func cachedAggregatedProfilePath(profileType string) string {
	return filepath.Join(ScratchDirectory, fmt.Sprintf("%s.consolidated.profile", profileType))
}
