// +build !lambdabinary

package sparta

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/lambda"
	spartaAWS "github.com/mweagle/Sparta/aws"
	spartaCF "github.com/mweagle/Sparta/aws/cloudformation"
	spartaS3 "github.com/mweagle/Sparta/aws/s3"
	spartaZip "github.com/mweagle/Sparta/zip"
	gocf "github.com/mweagle/go-cloudformation"
)

////////////////////////////////////////////////////////////////////////////////
// CONSTANTS
////////////////////////////////////////////////////////////////////////////////
func spartaTagName(baseKey string) string {
	return fmt.Sprintf("io:gosparta:%s", baseKey)
}

// SpartaTagHomeKey is the keyname used in the CloudFormation Output
// that stores the Sparta home URL.
// @enum OutputKey
var SpartaTagHomeKey = spartaTagName("home")

// SpartaTagVersionKey is the keyname used in the CloudFormation Output
// that stores the Sparta version used to provision/update the service.
// @enum OutputKey
var SpartaTagVersionKey = spartaTagName("version")

// SpartaTagHashKey is the keyname used in the CloudFormation Output
// that stores the Sparta commit ID used to provision/update the service
var SpartaTagHashKey = spartaTagName("sha")

// SpartaTagBuildIDKey is the keyname used in the CloudFormation Output
// that stores the user-supplied or automatically generated BuildID
// for this run
var SpartaTagBuildIDKey = spartaTagName("buildId")

// SpartaTagBuildTagsKey is the keyname used in the CloudFormation Output
// that stores the optional user-supplied golang build tags
var SpartaTagBuildTagsKey = spartaTagName("buildTags")

// finalizerFunction is the type of function pushed onto the cleanup stack
type finalizerFunction func(logger *logrus.Logger)

////////////////////////////////////////////////////////////////////////////////
// Type that encapsulates an S3 URL with accessors to return either the
// full URL or just the valid S3 Keyname
type s3UploadURL struct {
	location string
	version  string
}

func (s3URL *s3UploadURL) keyName() string {
	// Find the hostname in the URL, then strip it out
	urlParts, _ := url.Parse(s3URL.location)
	return strings.TrimPrefix(urlParts.Path, "/")
}

func newS3UploadURL(s3URL string) *s3UploadURL {
	urlParts, urlPartsErr := url.Parse(s3URL)
	if nil != urlPartsErr {
		return nil
	}
	queryParams, queryParamsErr := url.ParseQuery(urlParts.RawQuery)
	if nil != queryParamsErr {
		return nil
	}
	versionIDValues := queryParams["versionId"]
	version := ""
	if len(versionIDValues) == 1 {
		version = versionIDValues[0]
	}
	return &s3UploadURL{location: s3URL, version: version}
}

////////////////////////////////////////////////////////////////////////////////

// eitherResult is an "Either" result returned by a worker pool
type eitherResult struct {
	error  error
	result interface{}
}

////////////////////////////////////////////////////////////////////////////////
// Represents data associated with provisioning the S3 Site iff defined
type s3SiteContext struct {
	s3Site      *S3Site
	s3UploadURL *s3UploadURL
}

// Type of a workflow step.  Each step is responsible
// for returning the next step or an error if the overall
// workflow should stop.
type workflowStep func(ctx *workflowContext) (workflowStep, error)

// workflowStepDuration represents a discrete step in the provisioning
// workflow.
type workflowStepDuration struct {
	name     string
	duration time.Duration
}

// userdata is user-supplied, code related values
type userdata struct {
	// Is this is a -dry-run?
	noop bool
	// Is this a CGO enabled build?
	useCGO bool
	// Are in-place updates enabled?
	inPlace bool
	// The user-supplied or automatically generated BuildID
	buildID string
	// Optional user-supplied build tags
	buildTags string
	// Optional link flags
	linkFlags string
	// Canonical basename of the service.  Also used as the CloudFormation
	// stack name
	serviceName string
	// Service description
	serviceDescription string
	// The slice of Lambda functions that constitute the service
	lambdaAWSInfos []*LambdaAWSInfo
	// User supplied workflow hooks
	workflowHooks *WorkflowHooks
	// Code pipeline S3 trigger keyname
	codePipelineTrigger string
	// Optional APIGateway definition to associate with this service
	api *API
	// Optional S3 site data to provision together with this service
	s3SiteContext *s3SiteContext
	// The user-supplied S3 bucket where service artifacts should be posted.
	s3Bucket string
}

// context is data that is mutated during the provisioning workflow
type provisionContext struct {
	// Information about the ZIP archive that contains the LambdaCode source
	s3CodeZipURL *s3UploadURL
	// AWS Session to be used for all API calls made in the process of provisioning
	// this service.
	awsSession *session.Session
	// Cached IAM role name map.  Used to support dynamic and static IAM role
	// names.  Static ARN role names are checked for existence via AWS APIs
	// prior to CloudFormation provisioning.
	lambdaIAMRoleNameMap map[string]*gocf.StringExpr
	// IO writer for autogenerated template results
	templateWriter io.Writer
	// CloudFormation Template
	cfTemplate *gocf.Template
	// Is versioning enabled for s3 Bucket?
	s3BucketVersioningEnabled bool
	// Context to pass between workflow operations
	workflowHooksContext map[string]interface{}
}

// similar to context, transaction scopes values that span the entire
// provisioning step
type transaction struct {
	startTime time.Time
	// Optional rollback functions that workflow steps may append to if they
	// have made mutations during provisioning.
	rollbackFunctions []spartaS3.RollbackFunction
	// Optional finalizer functions that are unconditionally executed following
	// workflow completion, success or failure
	finalizerFunctions []finalizerFunction
	// Timings that measure how long things actually took
	stepDurations []*workflowStepDuration
}

////////////////////////////////////////////////////////////////////////////////
// Workflow context
// The workflow context is created by `provision` and provided to all
// functions that constitute the provisioning workflow.
type workflowContext struct {
	// User supplied data that's Lambda specific
	userdata userdata
	// Context that's mutated across the workflow steps
	context provisionContext
	// Transaction-scoped information thats mutated across the workflow
	// steps
	transaction transaction
	// Preconfigured logger
	logger *logrus.Logger
}

// recordDuration is a utility function to record how long
func recordDuration(start time.Time, name string, ctx *workflowContext) {
	elapsed := time.Since(start)
	ctx.transaction.stepDurations = append(ctx.transaction.stepDurations,
		&workflowStepDuration{
			name:     name,
			duration: elapsed,
		})
}

// Register a rollback function in the event that the provisioning
// function failed.
func (ctx *workflowContext) registerRollback(userFunction spartaS3.RollbackFunction) {
	if nil == ctx.transaction.rollbackFunctions || len(ctx.transaction.rollbackFunctions) <= 0 {
		ctx.transaction.rollbackFunctions = make([]spartaS3.RollbackFunction, 0)
	}
	ctx.transaction.rollbackFunctions = append(ctx.transaction.rollbackFunctions, userFunction)
}

// Register a rollback function in the event that the provisioning
// function failed.
func (ctx *workflowContext) registerFinalizer(userFunction finalizerFunction) {
	if nil == ctx.transaction.finalizerFunctions || len(ctx.transaction.finalizerFunctions) <= 0 {
		ctx.transaction.finalizerFunctions = make([]finalizerFunction, 0)
	}
	ctx.transaction.finalizerFunctions = append(ctx.transaction.finalizerFunctions, userFunction)
}

// Register a finalizer that cleans up local artifacts
func (ctx *workflowContext) registerFileCleanupFinalizer(localPath string) {
	cleanup := func(logger *logrus.Logger) {
		errRemove := os.Remove(localPath)
		if nil != errRemove {
			logger.WithFields(logrus.Fields{
				"Path":  localPath,
				"Error": errRemove,
			}).Warn("Failed to cleanup intermediate artifact")
		} else {
			logger.WithFields(logrus.Fields{
				"Path": relativePath(localPath),
			}).Debug("Build artifact deleted")
		}
	}
	ctx.registerFinalizer(cleanup)
}

// Run any provided rollback functions
func (ctx *workflowContext) rollback() {
	defer recordDuration(time.Now(), "Rollback", ctx)

	// Run each cleanup function concurrently.  If there's an error
	// all we're going to do is log it as a warning, since at this
	// point there's nothing to do...
	var wg sync.WaitGroup
	wg.Add(len(ctx.transaction.rollbackFunctions))

	// Include the user defined rollback if there is one...
	if ctx.userdata.workflowHooks != nil && ctx.userdata.workflowHooks.Rollback != nil {
		wg.Add(1)
		go func(hook RollbackHook, context map[string]interface{},
			serviceName string,
			awsSession *session.Session,
			noop bool,
			logger *logrus.Logger) {
			// Decrement the counter when the goroutine completes.
			defer wg.Done()
			hook(context, serviceName, awsSession, noop, logger)
		}(ctx.userdata.workflowHooks.Rollback,
			ctx.context.workflowHooksContext,
			ctx.userdata.serviceName,
			ctx.context.awsSession,
			ctx.userdata.noop,
			ctx.logger)
	}

	ctx.logger.WithFields(logrus.Fields{
		"RollbackCount": len(ctx.transaction.rollbackFunctions),
	}).Info("Invoking rollback functions")

	for _, eachCleanup := range ctx.transaction.rollbackFunctions {
		go func(cleanupFunc spartaS3.RollbackFunction, goLogger *logrus.Logger) {
			// Decrement the counter when the goroutine completes.
			defer wg.Done()
			// Fetch the URL.
			err := cleanupFunc(goLogger)
			if nil != err {
				ctx.logger.WithFields(logrus.Fields{
					"Error": err,
				}).Warning("Failed to cleanup resource")
			}
		}(eachCleanup, ctx.logger)
	}
	wg.Wait()
}

////////////////////////////////////////////////////////////////////////////////
// Private - START
//

// userGoPath returns either $GOPATH or the new $HOME/go path
// introduced with Go 1.8
func userGoPath() string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home := os.Getenv("HOME")
		gopath = filepath.Join(home, "go")
	}
	return gopath
}

// logFilesize outputs a friendly filesize for the given filepath
func logFilesize(message string, filePath string, logger *logrus.Logger) {
	// Binary size
	stat, err := os.Stat(filePath)
	if err == nil {
		logger.WithFields(logrus.Fields{
			"KB": stat.Size() / 1024,
			"MB": stat.Size() / (1024 * 1024),
		}).Info(message)
	}
}

// Encapsulate calling a workflow hook
func callWorkflowHook(hook WorkflowHook, ctx *workflowContext) error {
	if nil == hook {
		return nil
	}
	// Run the hook
	hookName := runtime.FuncForPC(reflect.ValueOf(hook).Pointer()).Name()
	ctx.logger.WithFields(logrus.Fields{
		"WorkflowHook":        hookName,
		"WorkflowHookContext": ctx.context.workflowHooksContext,
	}).Info("Calling WorkflowHook")

	return hook(ctx.context.workflowHooksContext,
		ctx.userdata.serviceName,
		ctx.userdata.s3Bucket,
		ctx.userdata.buildID,
		ctx.context.awsSession,
		ctx.userdata.noop,
		ctx.logger)
}

// versionAwareS3KeyName returns a keyname that provides the correct cache
// invalidation semantics based on whether the target bucket
// has versioning enabled
func versionAwareS3KeyName(s3DefaultKey string, s3VersioningEnabled bool, logger *logrus.Logger) (string, error) {
	versionKeyName := s3DefaultKey
	if !s3VersioningEnabled {
		var extension = path.Ext(s3DefaultKey)
		var prefixString = strings.TrimSuffix(s3DefaultKey, extension)

		hash := sha1.New()
		salt := fmt.Sprintf("%s-%d", s3DefaultKey, time.Now().UnixNano())
		hash.Write([]byte(salt))
		versionKeyName = fmt.Sprintf("%s-%s%s",
			prefixString,
			hex.EncodeToString(hash.Sum(nil)),
			extension)

		logger.WithFields(logrus.Fields{
			"Default":      s3DefaultKey,
			"Extension":    extension,
			"PrefixString": prefixString,
			"Unique":       versionKeyName,
		}).Debug("Created unique S3 keyname")
	}
	return versionKeyName, nil
}

// Upload a local file to S3.  Returns the full S3 URL to the file that was
// uploaded. If the target bucket does not have versioning enabled,
// this function will automatically make a new key to ensure uniqueness
func uploadLocalFileToS3(localPath string, s3ObjectKey string, ctx *workflowContext) (string, error) {

	// If versioning is enabled, use a stable name, otherwise use a name
	// that's dynamically created. By default assume that the bucket is
	// enabled for versioning
	if "" == s3ObjectKey {
		defaultS3KeyName := fmt.Sprintf("%s/%s", ctx.userdata.serviceName, filepath.Base(localPath))
		s3KeyName, s3KeyNameErr := versionAwareS3KeyName(defaultS3KeyName,
			ctx.context.s3BucketVersioningEnabled,
			ctx.logger)
		if nil != s3KeyNameErr {
			return "", s3KeyNameErr
		}
		s3ObjectKey = s3KeyName
	}

	s3URL := ""
	if ctx.userdata.noop {
		ctx.logger.WithFields(logrus.Fields{
			"Bucket": ctx.userdata.s3Bucket,
			"Key":    s3ObjectKey,
			"File":   filepath.Base(localPath),
		}).Info("Bypassing S3 upload due to -n/-noop command line argument")
		s3URL = fmt.Sprintf("https://%s-s3.amazonaws.com/%s", ctx.userdata.s3Bucket, s3ObjectKey)
	} else {
		// Make sure we mark things for cleanup in case there's a problem
		ctx.registerFileCleanupFinalizer(localPath)
		// Then upload it
		uploadLocation, uploadURLErr := spartaS3.UploadLocalFileToS3(localPath,
			ctx.context.awsSession,
			ctx.userdata.s3Bucket,
			s3ObjectKey,
			ctx.logger)
		if nil != uploadURLErr {
			return "", uploadURLErr
		}
		s3URL = uploadLocation
		ctx.registerRollback(spartaS3.CreateS3RollbackFunc(ctx.context.awsSession, uploadLocation))
	}
	return s3URL, nil
}

// Private - END
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// Workflow steps
////////////////////////////////////////////////////////////////////////////////

// Verify & cache the IAM rolename to ARN mapping
func verifyIAMRoles(ctx *workflowContext) (workflowStep, error) {
	defer recordDuration(time.Now(), "Verifying IAM roles", ctx)

	// The map is either a literal Arn from a pre-existing role name
	// or a gocf.RefFunc() value.
	// Don't verify them, just create them...
	ctx.logger.Info("Verifying IAM Lambda execution roles")
	ctx.context.lambdaIAMRoleNameMap = make(map[string]*gocf.StringExpr)
	svc := iam.New(ctx.context.awsSession)

	// Assemble all the RoleNames and validate the inline IAMRoleDefinitions
	var allRoleNames []string
	for _, eachLambdaInfo := range ctx.userdata.lambdaAWSInfos {
		if "" != eachLambdaInfo.RoleName {
			allRoleNames = append(allRoleNames, eachLambdaInfo.RoleName)
		}
		// Custom resources?
		for _, eachCustomResource := range eachLambdaInfo.customResources {
			if "" != eachCustomResource.roleName {
				allRoleNames = append(allRoleNames, eachCustomResource.roleName)
			}
		}
		// Profiling enabled?
		if profileDecorator != nil {
			profileErr := profileDecorator(ctx.userdata.serviceName,
				eachLambdaInfo,
				ctx.userdata.s3Bucket,
				ctx.logger)
			if profileErr != nil {
				return nil, profileErr
			}
		}

		// Validate the IAMRoleDefinitions associated
		if nil != eachLambdaInfo.RoleDefinition {
			logicalName := eachLambdaInfo.RoleDefinition.logicalName(ctx.userdata.serviceName, eachLambdaInfo.lambdaFunctionName())
			_, exists := ctx.context.lambdaIAMRoleNameMap[logicalName]
			if !exists {
				// Insert it into the resource creation map and add
				// the "Ref" entry to the hashmap
				ctx.context.cfTemplate.AddResource(logicalName,
					eachLambdaInfo.RoleDefinition.toResource(eachLambdaInfo.EventSourceMappings, eachLambdaInfo.Options, ctx.logger))

				ctx.context.lambdaIAMRoleNameMap[logicalName] = gocf.GetAtt(logicalName, "Arn")
			}
		}

		// And the custom resource IAMRoles as well...
		for _, eachCustomResource := range eachLambdaInfo.customResources {
			if nil != eachCustomResource.roleDefinition {
				customResourceLogicalName := eachCustomResource.roleDefinition.logicalName(ctx.userdata.serviceName,
					eachCustomResource.userFunctionName)

				_, exists := ctx.context.lambdaIAMRoleNameMap[customResourceLogicalName]
				if !exists {
					ctx.context.cfTemplate.AddResource(customResourceLogicalName,
						eachCustomResource.roleDefinition.toResource(nil, eachCustomResource.options, ctx.logger))
					ctx.context.lambdaIAMRoleNameMap[customResourceLogicalName] = gocf.GetAtt(customResourceLogicalName, "Arn")
				}
			}
		}
	}

	// Then check all the RoleName literals
	for _, eachRoleName := range allRoleNames {
		_, exists := ctx.context.lambdaIAMRoleNameMap[eachRoleName]
		if !exists {
			// Check the role
			params := &iam.GetRoleInput{
				RoleName: aws.String(eachRoleName),
			}
			ctx.logger.Debug("Checking IAM RoleName: ", eachRoleName)
			resp, err := svc.GetRole(params)
			if err != nil {
				ctx.logger.Error(err.Error())
				return nil, err
			}
			// Cache it - we'll need it later when we create the
			// CloudFormation template which needs the execution Arn (not role)
			ctx.context.lambdaIAMRoleNameMap[eachRoleName] = gocf.String(*resp.Role.Arn)
		}
	}
	ctx.logger.WithFields(logrus.Fields{
		"Count": len(ctx.context.lambdaIAMRoleNameMap),
	}).Info("IAM roles verified")

	return verifyAWSPreconditions, nil
}

// Verify that everything is setup in AWS before we start building things
func verifyAWSPreconditions(ctx *workflowContext) (workflowStep, error) {
	defer recordDuration(time.Now(), "Verifying AWS preconditions", ctx)

	// If this a NOOP, assume that versioning is not enabled
	if ctx.userdata.noop {
		ctx.logger.WithFields(logrus.Fields{
			"VersioningEnabled": false,
			"Bucket":            ctx.userdata.s3Bucket,
		}).Info("Bypassing S3 upload due to -n/-noop command line argument.")
	} else {
		// Get the S3 bucket and see if it has versioning enabled
		isEnabled, versioningPolicyErr := spartaS3.BucketVersioningEnabled(ctx.context.awsSession, ctx.userdata.s3Bucket, ctx.logger)
		if nil != versioningPolicyErr {
			return nil, versioningPolicyErr
		}
		ctx.logger.WithFields(logrus.Fields{
			"VersioningEnabled": isEnabled,
			"Bucket":            ctx.userdata.s3Bucket,
		}).Info("Checking S3 versioning")
		ctx.context.s3BucketVersioningEnabled = isEnabled
		if "" != ctx.userdata.codePipelineTrigger && !isEnabled {
			return nil, fmt.Errorf("Bucket (%s) for CodePipeline trigger doesn't have a versioning policy enabled", ctx.userdata.s3Bucket)
		}
	}

	// If there are codePipeline environments defined, warn if they don't include
	// the same keysets
	if nil != codePipelineEnvironments {
		mapKeys := func(inboundMap map[string]string) []string {
			keys := make([]string, len(inboundMap))
			i := 0
			for k := range inboundMap {
				keys[i] = k
				i++
			}
			return keys
		}
		aggregatedKeys := make([][]string, len(codePipelineEnvironments))
		i := 0
		for _, eachEnvMap := range codePipelineEnvironments {
			aggregatedKeys[i] = mapKeys(eachEnvMap)
			i++
		}
		i = 0
		keysEqual := true
		for _, eachKeySet := range aggregatedKeys {
			j := 0
			for _, eachKeySetTest := range aggregatedKeys {
				if j != i {
					if !reflect.DeepEqual(eachKeySet, eachKeySetTest) {
						keysEqual = false
					}
				}
				j++
			}
			i++
		}
		if !keysEqual {
			// Setup an interface with the fields so that the log message
			fields := make(logrus.Fields, len(codePipelineEnvironments))
			for eachEnv, eachEnvMap := range codePipelineEnvironments {
				fields[eachEnv] = eachEnvMap
			}
			ctx.logger.WithFields(fields).Warn("CodePipeline environments do not define equivalent environment keys")
		}
	}

	return createPackageStep(), nil
}

func ensureMainEntrypoint(logger *logrus.Logger) error {
	// Don't do this for "go test" runs
	if flag.Lookup("test.v") != nil {
		logger.Debug("Skipping main() check for test")
		return nil
	}

	fset := token.NewFileSet()
	packageMap, parseErr := parser.ParseDir(fset, ".", nil, parser.PackageClauseOnly)
	if parseErr != nil {
		return fmt.Errorf("Failed to parse source input: %s", parseErr.Error())
	}
	logger.WithFields(logrus.Fields{
		"SourcePackages": packageMap,
	}).Debug("Checking working directory")

	// If there isn't a main defined, we're in the wrong directory..
	mainPackageCount := 0
	for eachPackage := range packageMap {
		if eachPackage == "main" {
			mainPackageCount++
		}
	}
	if mainPackageCount <= 0 {
		unlikelyBinaryErr := fmt.Errorf("It appears your application's `func main() {}` is not in the current working directory. Please run this command in the same directory as `func main() {}`")
		return unlikelyBinaryErr
	}
	return nil
}

func buildGoBinary(executableOutput string,
	useCGO bool,
	buildTags string,
	linkFlags string,
	noop bool,
	logger *logrus.Logger) error {

	// Before we do anything, let's make sure there's a `main` package in this directory.
	ensureMainPackageErr := ensureMainEntrypoint(logger)
	if ensureMainPackageErr != nil {
		return ensureMainPackageErr
	}
	// Go generate
	cmd := exec.Command("go", "generate")
	if logger.Level == logrus.DebugLevel {
		cmd = exec.Command("go", "generate", "-v", "-x")
	}
	cmd.Env = os.Environ()
	commandString := fmt.Sprintf("%s", cmd.Args)
	logger.Info(fmt.Sprintf("Running `%s`", strings.Trim(commandString, "[]")))
	goGenerateErr := runOSCommand(cmd, logger)
	if nil != goGenerateErr {
		return goGenerateErr
	}
	// TODO: Smaller binaries via linker flags
	// Ref: https://blog.filippo.io/shrink-your-go-binaries-with-this-one-weird-trick/
	noopTag := ""
	if noop {
		noopTag = "noop "
	}
	userBuildFlags := []string{"-tags",
		fmt.Sprintf("lambdabinary %s%s", noopTag, buildTags)}
	// Append all the linker flags
	if len(linkFlags) != 0 {
		userBuildFlags = append(userBuildFlags, "-ldflags", linkFlags)
	}
	// If this is CGO, do the Docker build if we're doing an actual
	// provision. Otherwise use the "normal" build to keep things
	// a bit faster.
	var cmdError error
	if useCGO {
		currentDir, currentDirErr := os.Getwd()
		if nil != currentDirErr {
			return currentDirErr
		}
		gopathVersion, gopathVersionErr := systemGoVersion(logger)
		if nil != gopathVersionErr {
			return gopathVersionErr
		}

		gopath := userGoPath()
		containerGoPath := "/usr/src/gopath"
		// Get the package path in the current directory
		// so that we can it to the container path
		packagePath := strings.TrimPrefix(currentDir, gopath)
		volumeMountMapping := fmt.Sprintf("%s:%s", gopath, containerGoPath)
		containerSourcePath := fmt.Sprintf("%s%s", containerGoPath, packagePath)

		// Pass any SPARTA_* prefixed environment variables to the docker build
		//
		goosTarget := os.Getenv("SPARTA_GOOS")
		if goosTarget == "" {
			goosTarget = "linux"
		}
		goArch := os.Getenv("SPARTA_GOARCH")
		if goArch == "" {
			goArch = "amd64"
		}
		spartaEnvVars := []string{
			"-e",
			fmt.Sprintf("GOPATH=%s", containerGoPath),
			"-e",
			fmt.Sprintf("GOOS=%s", goosTarget),
			"-e",
			fmt.Sprintf("GOARCH=%s", goArch),
		}
		// User vars
		for _, eachPair := range os.Environ() {
			if strings.HasPrefix(eachPair, "SPARTA_") {
				spartaEnvVars = append(spartaEnvVars, "-e", eachPair)
			}
		}

		dockerBuildArgs := []string{
			"run",
			"--rm",
			"-v",
			volumeMountMapping,
			"-w",
			containerSourcePath}
		dockerBuildArgs = append(dockerBuildArgs, spartaEnvVars...)
		dockerBuildArgs = append(dockerBuildArgs,
			fmt.Sprintf("golang:%s", gopathVersion),
			"go",
			"build",
			"-o",
			executableOutput,
			"-tags",
			"lambdabinary linux ",
			"-buildmode=c-shared",
		)
		dockerBuildArgs = append(dockerBuildArgs, userBuildFlags...)
		cmd = exec.Command("docker", dockerBuildArgs...)
		cmd.Env = os.Environ()
		logger.WithFields(logrus.Fields{
			"Name": executableOutput,
			"Args": dockerBuildArgs,
		}).Info("Building `cgo` library in Docker")
		cmdError = runOSCommand(cmd, logger)

		// If this succeeded, let's find the .h file and move it into the scratch
		// Try to keep things tidy...
		if nil == cmdError {
			soExtension := filepath.Ext(executableOutput)
			headerFilepath := fmt.Sprintf("%s.h", strings.TrimSuffix(executableOutput, soExtension))
			_, headerFileErr := os.Stat(headerFilepath)
			if nil == headerFileErr {
				targetPath, targetPathErr := temporaryFile(filepath.Base(headerFilepath))
				if nil != targetPathErr {
					headerFileErr = targetPathErr
				} else {
					headerFileErr = os.Rename(headerFilepath, targetPath.Name())
				}
			}
			if nil != headerFileErr {
				logger.WithFields(logrus.Fields{
					"Path": headerFilepath,
				}).Warn("Failed to move .h file to scratch directory")
			}
		}
	} else {

		// Build the NodeJS version
		buildArgs := []string{
			"build",
			"-o",
			executableOutput,
		}
		// Debug flags?
		if logger.Level == logrus.DebugLevel {
			buildArgs = append(buildArgs, "-v")
		}
		buildArgs = append(buildArgs, userBuildFlags...)
		buildArgs = append(buildArgs, ".")
		cmd = exec.Command("go", buildArgs...)
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "GOOS=linux", "GOARCH=amd64")
		logger.WithFields(logrus.Fields{
			"Name": executableOutput,
		}).Info("Compiling binary")
		cmdError = runOSCommand(cmd, logger)
	}
	return cmdError
}

// Build and package the application
func createPackageStep() workflowStep {
	return func(ctx *workflowContext) (workflowStep, error) {
		defer recordDuration(time.Now(), "Creating code bundle", ctx)

		// PreBuild Hook
		if ctx.userdata.workflowHooks != nil {
			preBuildErr := callWorkflowHook(ctx.userdata.workflowHooks.PreBuild, ctx)
			if nil != preBuildErr {
				return nil, preBuildErr
			}
		}
		binarySuffix := "lambda.amd64"
		if ctx.userdata.useCGO {
			binarySuffix = "lambda.so"
		}
		sanitizedServiceName := sanitizedName(ctx.userdata.serviceName)
		executableOutput := fmt.Sprintf("Sparta.%s", binarySuffix)
		buildErr := buildGoBinary(executableOutput,
			ctx.userdata.useCGO,
			ctx.userdata.buildTags,
			ctx.userdata.linkFlags,
			ctx.userdata.noop,
			ctx.logger)
		if nil != buildErr {
			return nil, buildErr
		}
		// Cleanup the temporary binary
		defer func() {
			errRemove := os.Remove(executableOutput)
			if nil != errRemove {
				ctx.logger.WithFields(logrus.Fields{
					"File":  executableOutput,
					"Error": errRemove,
				}).Warn("Failed to delete binary")
			}
		}()

		// Binary size
		logFilesize("Executable binary size", executableOutput, ctx.logger)

		// PostBuild Hook
		if ctx.userdata.workflowHooks != nil {
			postBuildErr := callWorkflowHook(ctx.userdata.workflowHooks.PostBuild, ctx)
			if nil != postBuildErr {
				return nil, postBuildErr
			}
		}
		tmpFile, err := temporaryFile(fmt.Sprintf("%s-code.zip", sanitizedServiceName))
		if err != nil {
			return nil, err
		}
		// Strip the local directory in case it's in there...
		ctx.logger.WithFields(logrus.Fields{
			"TempName": relativePath(tmpFile.Name()),
		}).Info("Creating code ZIP archive for upload")
		lambdaArchive := zip.NewWriter(tmpFile)

		// Archive Hook
		if ctx.userdata.workflowHooks != nil && ctx.userdata.workflowHooks.Archive != nil {
			archiveErr := ctx.userdata.workflowHooks.Archive(ctx.context.workflowHooksContext,
				ctx.userdata.serviceName,
				lambdaArchive,
				ctx.context.awsSession,
				ctx.userdata.noop,
				ctx.logger)
			if nil != archiveErr {
				return nil, archiveErr
			}
		}

		// File info for the binary executable
		readerErr := spartaZip.AddToZip(lambdaArchive,
			executableOutput,
			"bin",
			ctx.logger)
		if nil != readerErr {
			return nil, readerErr
		}

		// Based on whether this is NodeJS or CGO, pick the proper shim
		// and write the custom entries

		// Add the string literal adapter, which requires us to add exported
		// functions to the end of index.js.  These NodeJS exports will be
		// linked to the AWS Lambda NodeJS function name, and are basically
		// automatically generated pass through proxies to the golang HTTP handler.
		var shimErr error
		if ctx.userdata.useCGO {
			shimErr = insertPythonProxyResources(ctx.userdata.serviceName,
				executableOutput,
				ctx.userdata.lambdaAWSInfos,
				lambdaArchive,
				ctx.logger)
		} else {
			shimErr = insertNodeJSProxyResources(ctx.userdata.serviceName,
				executableOutput,
				ctx.userdata.lambdaAWSInfos,
				lambdaArchive,
				ctx.logger)
		}
		if nil != shimErr {
			return nil, shimErr
		}
		archiveCloseErr := lambdaArchive.Close()
		if nil != archiveCloseErr {
			return nil, archiveCloseErr
		}
		tempfileCloseErr := tmpFile.Close()
		if nil != tempfileCloseErr {
			return nil, tempfileCloseErr
		}
		return createUploadStep(tmpFile.Name()), nil
	}
}

// Given the zipped binary in packagePath, upload the primary code bundle
// and optional S3 site resources iff they're defined.
func createUploadStep(packagePath string) workflowStep {
	return func(ctx *workflowContext) (workflowStep, error) {
		defer recordDuration(time.Now(), "Uploading code", ctx)

		var uploadErrors []error
		var wg sync.WaitGroup

		// We always need to upload the primary binary
		wg.Add(1)
		go func() {
			defer wg.Done()
			logFilesize("Lambda code archive size", packagePath, ctx.logger)

			// Create the S3 key...
			zipS3URL, zipS3URLErr := uploadLocalFileToS3(packagePath, "", ctx)
			if nil != zipS3URLErr {
				uploadErrors = append(uploadErrors, zipS3URLErr)
			} else {
				ctx.context.s3CodeZipURL = newS3UploadURL(zipS3URL)
			}
		}()

		// S3 site to compress & upload
		if nil != ctx.userdata.s3SiteContext.s3Site {
			wg.Add(1)
			go func() {
				defer wg.Done()

				tempName := fmt.Sprintf("%s-S3Site.zip", ctx.userdata.serviceName)
				tmpFile, err := temporaryFile(tempName)
				if err != nil {
					uploadErrors = append(uploadErrors,
						errors.New("Failed to create temporary S3 site archive file"))
					return
				}

				// Add the contents to the Zip file
				zipArchive := zip.NewWriter(tmpFile)
				absResourcePath, err := filepath.Abs(ctx.userdata.s3SiteContext.s3Site.resources)
				if nil != err {
					uploadErrors = append(uploadErrors, err)
					return
				}

				ctx.logger.WithFields(logrus.Fields{
					"S3Key":  path.Base(tmpFile.Name()),
					"Source": absResourcePath,
				}).Info("Creating S3Site archive")

				err = spartaZip.AddToZip(zipArchive, absResourcePath, absResourcePath, ctx.logger)
				if nil != err {
					uploadErrors = append(uploadErrors, err)
					return
				}
				zipArchive.Close()

				// Upload it & save the key
				s3SiteLambdaZipURL, s3SiteLambdaZipURLErr := uploadLocalFileToS3(tmpFile.Name(), "", ctx)
				if s3SiteLambdaZipURLErr != nil {
					uploadErrors = append(uploadErrors, s3SiteLambdaZipURLErr)
				} else {
					ctx.userdata.s3SiteContext.s3UploadURL = newS3UploadURL(s3SiteLambdaZipURL)
				}
			}()
		}
		wg.Wait()

		if len(uploadErrors) > 0 {
			errorText := "Encountered multiple errors during upload:\n"
			for _, eachError := range uploadErrors {
				errorText += fmt.Sprintf("%s%s\n", errorText, eachError.Error())
				return nil, errors.New(errorText)
			}
		}
		return ensureCloudFormationStack(), nil
	}
}

func annotateDiscoveryInfo(template *gocf.Template, logger *logrus.Logger) *gocf.Template {
	for eachResourceID, eachResource := range template.Resources {
		// Only apply this to lambda functions
		if eachResource.Properties.CfnResourceType() == "AWS::Lambda::Function" {

			// Update the metdata with a reference to the output of each
			// depended on item...
			for _, eachDependsKey := range eachResource.DependsOn {
				dependencyOutputs, _ := outputsForResource(template, eachDependsKey, logger)
				if nil != dependencyOutputs && len(dependencyOutputs) != 0 {
					logger.WithFields(logrus.Fields{
						"Resource":  eachDependsKey,
						"DependsOn": eachResource.DependsOn,
						"Outputs":   dependencyOutputs,
					}).Debug("Resource metadata")
					safeMetadataInsert(eachResource, eachDependsKey, dependencyOutputs)
				}
			}
			// Also include standard AWS outputs at a resource level if a lambda
			// needs to self-discover other resources
			safeMetadataInsert(eachResource, TagLogicalResourceID, gocf.String(eachResourceID))
			safeMetadataInsert(eachResource, TagStackRegion, gocf.Ref("AWS::Region"))
			safeMetadataInsert(eachResource, TagStackID, gocf.Ref("AWS::StackId"))
			safeMetadataInsert(eachResource, TagStackName, gocf.Ref("AWS::StackName"))
		}
	}
	return template
}

// createCodePipelineTriggerPackage handles marshaling the template, zipping
// the config files in the package, and the
func createCodePipelineTriggerPackage(cfTemplateJSON []byte, ctx *workflowContext) (string, error) {
	tmpFile, err := temporaryFile(ctx.userdata.codePipelineTrigger)
	if err != nil {
		return "", err
	}

	ctx.logger.WithFields(logrus.Fields{
		"PipelineName": tmpFile.Name(),
	}).Info("Creating pipeline archive")

	templateArchive := zip.NewWriter(tmpFile)
	ctx.logger.WithFields(logrus.Fields{
		"Path": tmpFile.Name(),
	}).Info("Creating CodePipeline archive")

	// File info for the binary executable
	zipEntryName := "cloudformation.json"
	bytesWriter, bytesWriterErr := templateArchive.Create(zipEntryName)
	if bytesWriterErr != nil {
		return "", bytesWriterErr
	}

	bytesReader := bytes.NewReader(cfTemplateJSON)
	written, writtenErr := io.Copy(bytesWriter, bytesReader)
	if nil != writtenErr {
		return "", writtenErr
	}
	ctx.logger.WithFields(logrus.Fields{
		"WrittenBytes": written,
		"ZipName":      zipEntryName,
	}).Debug("Archiving file")

	// If there is a codePipelineEnvironments defined, then we'll need to get all the
	// maps, marshal them to JSON, then add the JSON to the ZIP archive.
	if nil != codePipelineEnvironments {
		for eachEnvironment, eachMap := range codePipelineEnvironments {
			codePipelineParameters := map[string]interface{}{
				"Parameters": eachMap,
			}
			environmentJSON, environmentJSONErr := json.Marshal(codePipelineParameters)
			if nil != environmentJSONErr {
				ctx.logger.Error("Failed to Marshal CodePipeline environment: " + eachEnvironment)
				return "", environmentJSONErr
			}
			var envVarName = fmt.Sprintf("%s.json", eachEnvironment)

			// File info for the binary executable
			binaryWriter, binaryWriterErr := templateArchive.Create(envVarName)
			if binaryWriterErr != nil {
				return "", binaryWriterErr
			}
			_, writeErr := binaryWriter.Write(environmentJSON)
			if writeErr != nil {
				return "", writeErr
			}
		}
	}
	archiveCloseErr := templateArchive.Close()
	if nil != archiveCloseErr {
		return "", archiveCloseErr
	}
	tempfileCloseErr := tmpFile.Close()
	if nil != tempfileCloseErr {
		return "", tempfileCloseErr
	}
	// Leave it here...
	ctx.logger.WithFields(logrus.Fields{
		"File": filepath.Base(tmpFile.Name()),
	}).Info("Created CodePipeline archive")
	return tmpFile.Name(), nil
	// The key is the name + the pipeline name
	//return uploadLocalFileToS3(tmpFile.Name(), "", ctx)
}

// If the only detected changes to a stack are Lambda code updates,
// then update use the LAmbda API to update the code directly
// rather than waiting for CloudFormation
func applyInPlaceFunctionUpdates(ctx *workflowContext, templateURL string) (*cloudformation.Stack, error) {
	// Get the updates...
	awsCloudFormation := cloudformation.New(ctx.context.awsSession)
	changeSetRequestName := CloudFormationResourceName(fmt.Sprintf("%sInPlaceChangeSet", ctx.userdata.serviceName))
	changes, changesErr := spartaCF.CreateStackChangeSet(changeSetRequestName,
		ctx.userdata.serviceName,
		ctx.context.cfTemplate,
		templateURL,
		nil,
		awsCloudFormation,
		ctx.logger)
	if nil != changesErr {
		return nil, changesErr
	}
	if nil == changes || len(changes.Changes) <= 0 {
		return nil, fmt.Errorf("No changes detected")
	}
	updateCodeRequests := []*lambda.UpdateFunctionCodeInput{}
	invalidInPlaceRequests := []string{}
	for _, eachChange := range changes.Changes {
		resourceChange := eachChange.ResourceChange
		if *resourceChange.Action == "Modify" && *resourceChange.ResourceType == "AWS::Lambda::Function" {
			updateCodeRequest := &lambda.UpdateFunctionCodeInput{
				FunctionName: resourceChange.PhysicalResourceId,
				S3Bucket:     aws.String(ctx.userdata.s3Bucket),
				S3Key:        aws.String(ctx.context.s3CodeZipURL.keyName()),
			}
			if ctx.context.s3CodeZipURL.version != "" {
				updateCodeRequest.S3ObjectVersion = aws.String(ctx.context.s3CodeZipURL.version)
			}
			updateCodeRequests = append(updateCodeRequests, updateCodeRequest)
		} else {
			invalidInPlaceRequests = append(invalidInPlaceRequests,
				fmt.Sprintf("%s for %s (ResourceType: %s)",
					*resourceChange.Action,
					*resourceChange.LogicalResourceId,
					*resourceChange.ResourceType))
		}
	}
	if len(invalidInPlaceRequests) != 0 {
		return nil, fmt.Errorf("Unsupported in-place operations detected:\n\t%s", strings.Join(invalidInPlaceRequests, ",\n\t"))
	}

	ctx.logger.WithFields(logrus.Fields{
		"FunctionCount": len(updateCodeRequests),
	}).Info("Updating Lambda function code")
	ctx.logger.WithFields(logrus.Fields{
		"Updates": updateCodeRequests,
	}).Debug("Update requests")

	// Run the updates...
	var wg sync.WaitGroup
	// The concurrent ops include the Lambda updates as well as the
	// request to delete the changeset we created to see if updating
	// the lambda code is a safe operation
	wgCount := len(updateCodeRequests) + 1
	wg.Add(wgCount)
	resultChannel := make(chan eitherResult, wgCount)
	awsLambda := lambda.New(ctx.context.awsSession)
	for _, eachUpdateCodeRequest := range updateCodeRequests {
		go func(lambdaSvc *lambda.Lambda, input *lambda.UpdateFunctionCodeInput) {
			_, updateResultErr := lambdaSvc.UpdateFunctionCode(input)
			if nil != updateResultErr {
				resultChannel <- eitherResult{error: updateResultErr}
			} else {
				resultChannel <- eitherResult{result: fmt.Sprintf("Updated function: %s", *input.FunctionName)}
			}
			wg.Done()
		}(awsLambda, eachUpdateCodeRequest)
	}
	// Finally, add the request to delete the changeset
	go func(cloudformationSvc *cloudformation.CloudFormation) {
		_, deleteChangeSetResultErr := spartaCF.DeleteChangeSet(ctx.userdata.serviceName,
			changeSetRequestName,
			cloudformationSvc)
		if nil != deleteChangeSetResultErr {
			resultChannel <- eitherResult{error: deleteChangeSetResultErr}
		} else {
			resultChannel <- eitherResult{result: "Deleted changeset: " + changeSetRequestName}
		}
		wg.Done()
	}(awsCloudFormation)
	wg.Wait()
	close(resultChannel)

	// What happened?
	asyncErrors := []string{}
	for eachResult := range resultChannel {
		if nil != eachResult.error {
			asyncErrors = append(asyncErrors, eachResult.error.Error())
		} else {
			ctx.logger.Debug(eachResult.result)
		}
	}
	if len(asyncErrors) != 0 {
		return nil, fmt.Errorf(strings.Join(asyncErrors, ", "))
	}
	// Describe the stack so that we can satisfy the contract with the
	// normal path using CloudFormation
	describeStacksInput := &cloudformation.DescribeStacksInput{
		StackName: aws.String(ctx.userdata.serviceName),
	}
	describeStackOutput, describeStackOutputErr := awsCloudFormation.DescribeStacks(describeStacksInput)
	if nil != describeStackOutputErr {
		return nil, describeStackOutputErr
	}
	return describeStackOutput.Stacks[0], nil
}

// applyCloudFormationOperation is responsible for taking the current template
// and applying that operation to the stack. It's where the in-place
// branch is applied, because at this point all the template
// mutations have been accumulated
func applyCloudFormationOperation(ctx *workflowContext) (workflowStep, error) {
	stackTags := map[string]string{
		SpartaTagHomeKey:    "http://gosparta.io",
		SpartaTagVersionKey: SpartaVersion,
		SpartaTagHashKey:    SpartaGitHash,
		SpartaTagBuildIDKey: ctx.userdata.buildID,
	}
	if len(ctx.userdata.buildTags) != 0 {
		stackTags[SpartaTagBuildTagsKey] = ctx.userdata.buildTags
	}
	// Generate the CF template...
	cfTemplate, err := json.Marshal(ctx.context.cfTemplate)
	if err != nil {
		ctx.logger.Error("Failed to Marshal CloudFormation template: ", err.Error())
		return nil, err
	}

	// Consistent naming of template
	sanitizedServiceName := sanitizedName(ctx.userdata.serviceName)
	templateName := fmt.Sprintf("%s-cftemplate.json", sanitizedServiceName)
	templateFile, templateFileErr := temporaryFile(templateName)
	if nil != templateFileErr {
		return nil, templateFileErr
	}
	_, writeErr := templateFile.Write(cfTemplate)
	if nil != writeErr {
		return nil, writeErr
	}
	templateFile.Close()

	// Log the template if needed
	if nil != ctx.context.templateWriter || ctx.logger.Level <= logrus.DebugLevel {
		templateBody := string(cfTemplate)
		formatted, formattedErr := json.MarshalIndent(templateBody, "", " ")
		if nil != formattedErr {
			return nil, formattedErr
		}
		ctx.logger.WithFields(logrus.Fields{
			"Body": string(formatted),
		}).Debug("CloudFormation template body")
		if nil != ctx.context.templateWriter {
			io.WriteString(ctx.context.templateWriter, string(formatted))
		}
	}

	// If this isn't a codePipelineTrigger, then do that
	if "" == ctx.userdata.codePipelineTrigger {
		if ctx.userdata.noop {
			ctx.logger.WithFields(logrus.Fields{
				"Bucket":       ctx.userdata.s3Bucket,
				"TemplateName": templateName,
			}).Info("Bypassing Stack creation due to -n/-noop command line argument")
		} else {
			// Dump the template to a file, then upload it...
			uploadURL, uploadURLErr := uploadLocalFileToS3(templateFile.Name(), "", ctx)
			if nil != uploadURLErr {
				return nil, uploadURLErr
			}

			// If we're supposed to be inplace, then go ahead and try that
			var stack *cloudformation.Stack
			var stackErr error
			if ctx.userdata.inPlace {
				stack, stackErr = applyInPlaceFunctionUpdates(ctx, uploadURL)
			} else {
				// Regular update, go ahead with the CloudFormation changes
				stack, stackErr = spartaCF.ConvergeStackState(ctx.userdata.serviceName,
					ctx.context.cfTemplate,
					uploadURL,
					stackTags,
					ctx.transaction.startTime,
					ctx.context.awsSession,
					ctx.logger)
			}
			if nil != stackErr {
				return nil, stackErr
			}
			ctx.logger.WithFields(logrus.Fields{
				"StackName":    *stack.StackName,
				"StackId":      *stack.StackId,
				"CreationTime": *stack.CreationTime,
			}).Info("Stack provisioned")
		}
	} else {
		ctx.logger.Info("Creating pipeline package")

		ctx.registerFileCleanupFinalizer(templateFile.Name())
		_, urlErr := createCodePipelineTriggerPackage(cfTemplate, ctx)
		if nil != urlErr {
			return nil, urlErr
		}
	}
	return nil, nil
}

func annotateCodePipelineEnvironments(lambdaAWSInfo *LambdaAWSInfo, logger *logrus.Logger) {
	if nil != codePipelineEnvironments {
		if nil == lambdaAWSInfo.Options {
			lambdaAWSInfo.Options = defaultLambdaFunctionOptions()
		}
		if nil == lambdaAWSInfo.Options.Environment {
			lambdaAWSInfo.Options.Environment = make(map[string]*gocf.StringExpr)
		}
		for _, eachEnvironment := range codePipelineEnvironments {

			logger.WithFields(logrus.Fields{
				"Environment":    eachEnvironment,
				"LambdaFunction": lambdaAWSInfo.lambdaFunctionName(),
			}).Debug("Annotating Lambda environment for CodePipeline")

			for eachKey := range eachEnvironment {
				lambdaAWSInfo.Options.Environment[eachKey] = gocf.Ref(eachKey).String()
			}
		}
	}
}

func ensureCloudFormationStack() workflowStep {
	return func(ctx *workflowContext) (workflowStep, error) {
		msg := "Ensuring CloudFormation stack"
		if ctx.userdata.inPlace {
			msg = "Updating Lambda function code "
		}
		defer recordDuration(time.Now(), msg, ctx)

		// PreMarshall Hook
		if ctx.userdata.workflowHooks != nil {
			preMarshallErr := callWorkflowHook(ctx.userdata.workflowHooks.PreMarshall, ctx)
			if nil != preMarshallErr {
				return nil, preMarshallErr
			}
		}

		// Add the "Parameters" to the template...
		if nil != codePipelineEnvironments {
			ctx.context.cfTemplate.Parameters = make(map[string]*gocf.Parameter)
			for _, eachEnvironment := range codePipelineEnvironments {
				for eachKey := range eachEnvironment {
					ctx.context.cfTemplate.Parameters[eachKey] = &gocf.Parameter{
						Type:    "String",
						Default: "",
					}
				}
			}
		}
		lambdaRuntime := NodeJSVersion
		if ctx.userdata.useCGO {
			lambdaRuntime = PythonVersion
		}
		for _, eachEntry := range ctx.userdata.lambdaAWSInfos {
			// If this is a legacy Sparta lambda function, let the user know
			if eachEntry.lambdaFn != nil {
				ctx.logger.WithFields(logrus.Fields{
					"Name": eachEntry.lambdaFunctionName(),
				}).Warn("DEPRECATED: sparta.LambdaFunc() signature provided. Please migrate to http.HandlerFunc()")
			}
			annotateCodePipelineEnvironments(eachEntry, ctx.logger)

			err := eachEntry.export(ctx.userdata.serviceName,
				ctx.userdata.useCGO,
				lambdaRuntime,
				ctx.userdata.s3Bucket,
				ctx.context.s3CodeZipURL.keyName(),
				ctx.context.s3CodeZipURL.version,
				ctx.userdata.buildID,
				ctx.context.lambdaIAMRoleNameMap,
				ctx.context.cfTemplate,
				ctx.context.workflowHooksContext,
				ctx.logger)
			if nil != err {
				return nil, err
			}
		}
		// If there's an API gateway definition, include the resources that provision it. Since this export will likely
		// generate outputs that the s3 site needs, we'll use a temporary outputs accumulator, pass that to the S3Site
		// if it's defined, and then merge it with the normal output map.
		apiGatewayTemplate := gocf.NewTemplate()

		if nil != ctx.userdata.api {
			err := ctx.userdata.api.export(
				ctx.userdata.serviceName,
				ctx.context.awsSession,
				ctx.userdata.s3Bucket,
				ctx.context.s3CodeZipURL.keyName(),
				ctx.context.s3CodeZipURL.version,
				ctx.context.lambdaIAMRoleNameMap,
				apiGatewayTemplate,
				ctx.userdata.noop,
				ctx.logger)
			if nil == err {
				err = safeMergeTemplates(apiGatewayTemplate, ctx.context.cfTemplate, ctx.logger)
			}
			if nil != err {
				return nil, fmt.Errorf("Failed to export APIGateway template resources")
			}
		}
		// If there's a Site defined, include the resources the provision it
		if nil != ctx.userdata.s3SiteContext.s3Site {
			ctx.userdata.s3SiteContext.s3Site.export(ctx.userdata.serviceName,
				ctx.userdata.s3Bucket,
				ctx.context.s3CodeZipURL.keyName(),
				ctx.userdata.s3SiteContext.s3UploadURL.keyName(),
				ctx.userdata.useCGO,
				apiGatewayTemplate.Outputs,
				ctx.context.lambdaIAMRoleNameMap,
				ctx.context.cfTemplate,
				ctx.logger)
		}
		// Service decorator?
		// If there's an API gateway definition, include the resources that provision it. Since this export will likely
		// generate outputs that the s3 site needs, we'll use a temporary outputs accumulator, pass that to the S3Site
		// if it's defined, and then merge it with the normal output map.-
		if nil != ctx.userdata.workflowHooks && nil != ctx.userdata.workflowHooks.ServiceDecorator {
			hookName := runtime.FuncForPC(reflect.ValueOf(ctx.userdata.workflowHooks.ServiceDecorator).Pointer()).Name()
			ctx.logger.WithFields(logrus.Fields{
				"WorkflowHook":        hookName,
				"WorkflowHookContext": ctx.context.workflowHooksContext,
			}).Info("Calling WorkflowHook")

			serviceTemplate := gocf.NewTemplate()
			decoratorError := ctx.userdata.workflowHooks.ServiceDecorator(
				ctx.context.workflowHooksContext,
				ctx.userdata.serviceName,
				serviceTemplate,
				ctx.userdata.s3Bucket,
				ctx.userdata.buildID,
				ctx.context.awsSession,
				ctx.userdata.noop,
				ctx.logger,
			)
			if nil != decoratorError {
				return nil, decoratorError
			}
			mergeErr := safeMergeTemplates(serviceTemplate, ctx.context.cfTemplate, ctx.logger)
			if nil != mergeErr {
				return nil, mergeErr
			}
		}
		ctx.context.cfTemplate = annotateDiscoveryInfo(ctx.context.cfTemplate, ctx.logger)

		// PostMarshall Hook
		if ctx.userdata.workflowHooks != nil {
			postMarshallErr := callWorkflowHook(ctx.userdata.workflowHooks.PostMarshall, ctx)
			if nil != postMarshallErr {
				return nil, postMarshallErr
			}
		}
		return applyCloudFormationOperation(ctx)
	}
}

// Provision compiles, packages, and provisions (either via create or update) a Sparta application.
// The serviceName is the service's logical
// identify and is used to determine create vs update operations.  The compilation options/flags are:
//
// 	TAGS:         -tags lambdabinary
// 	ENVIRONMENT:  GOOS=linux GOARCH=amd64
//
// The compiled binary is packaged with a NodeJS proxy shim to manage AWS Lambda setup & invocation per
// http://docs.aws.amazon.com/lambda/latest/dg/authoring-function-in-nodejs.html
//
// The two files are ZIP'd, posted to S3 and used as an input to a dynamically generated CloudFormation
// template (http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/Welcome.html)
// which creates or updates the service state.
//
func Provision(noop bool,
	serviceName string,
	serviceDescription string,
	lambdaAWSInfos []*LambdaAWSInfo,
	api *API,
	site *S3Site,
	s3Bucket string,
	useCGO bool,
	inPlaceUpdates bool,
	buildID string,
	codePipelineTrigger string,
	buildTags string,
	linkerFlags string,
	templateWriter io.Writer,
	workflowHooks *WorkflowHooks,
	logger *logrus.Logger) error {

	err := validateSpartaPreconditions(lambdaAWSInfos, logger)
	if nil != err {
		return err
	}
	startTime := time.Now()

	ctx := &workflowContext{
		logger: logger,
		userdata: userdata{
			noop:               noop,
			useCGO:             useCGO,
			inPlace:            inPlaceUpdates,
			buildID:            buildID,
			buildTags:          buildTags,
			linkFlags:          linkerFlags,
			serviceName:        serviceName,
			serviceDescription: serviceDescription,
			lambdaAWSInfos:     lambdaAWSInfos,
			api:                api,
			s3Bucket:           s3Bucket,
			s3SiteContext: &s3SiteContext{
				s3Site: site,
			},
			codePipelineTrigger: codePipelineTrigger,
			workflowHooks:       workflowHooks,
		},
		context: provisionContext{
			cfTemplate:                gocf.NewTemplate(),
			s3BucketVersioningEnabled: false,
			awsSession:                spartaAWS.NewSession(logger),
			workflowHooksContext:      make(map[string]interface{}),
			templateWriter:            templateWriter,
		},
		transaction: transaction{
			startTime: time.Now(),
		},
	}
	ctx.context.cfTemplate.Description = serviceDescription

	// Update the context iff it exists
	if nil != workflowHooks && nil != workflowHooks.Context {
		for eachKey, eachValue := range workflowHooks.Context {
			ctx.context.workflowHooksContext[eachKey] = eachValue
		}
	}

	ctx.logger.WithFields(logrus.Fields{
		"BuildID":             buildID,
		"NOOP":                noop,
		"Tags":                ctx.userdata.buildTags,
		"CodePipelineTrigger": ctx.userdata.codePipelineTrigger,
		"InPlaceUpdates":      ctx.userdata.inPlace,
	}).Info("Provisioning service")

	if len(lambdaAWSInfos) <= 0 {
		return errors.New("No lambda functions provided to Sparta.Provision()")
	}

	// Start the workflow
	for step := verifyIAMRoles; step != nil; {
		next, err := step(ctx)
		if err != nil {
			ctx.rollback()
			// Workflow step?
			return err
		}

		if next == nil {
			summaryLine := fmt.Sprintf("%s Summary (%s)",
				ctx.userdata.serviceName,
				time.Now().Format(time.RFC3339))
			subheaderDivider := strings.Repeat("─", dividerLength)

			ctx.logger.Info(subheaderDivider)
			ctx.logger.Info(summaryLine)
			ctx.logger.Info(subheaderDivider)
			for _, eachEntry := range ctx.transaction.stepDurations {
				ctx.logger.WithFields(logrus.Fields{
					"Duration (s)": fmt.Sprintf("%.f", eachEntry.duration.Seconds()),
				}).Info(eachEntry.name)
			}
			elapsed := time.Since(startTime)
			ctx.logger.WithFields(logrus.Fields{
				"Duration (s)": fmt.Sprintf("%.f", elapsed.Seconds()),
			}).Info("Total elapsed time")
			ctx.logger.Info(subheaderDivider)
			break
		} else {
			step = next
		}
	}
	// When we're done, execute any finalizers
	if nil != ctx.transaction.finalizerFunctions {
		ctx.logger.WithFields(logrus.Fields{
			"FinalizerCount": len(ctx.transaction.finalizerFunctions),
		}).Debug("Invoking finalizer functions")
		for _, eachFinalizer := range ctx.transaction.finalizerFunctions {
			eachFinalizer(ctx.logger)
		}
	}
	return nil
}
