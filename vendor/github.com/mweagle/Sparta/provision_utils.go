package sparta

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"text/template"

	"github.com/Sirupsen/logrus"
	spartaIAM "github.com/mweagle/Sparta/aws/iam"
	"github.com/mweagle/cloudformationresources"
	gocf "github.com/mweagle/go-cloudformation"
)

const (
	// ScratchDirectory is the cwd relative path component
	// where intermediate build artifacts are created
	ScratchDirectory = ".sparta"

	// The relative path of the custom scripts that is used
	// to create the filename relative path when creating the custom archive
	provisioningResourcesRelPath = "/resources/provision"
)

// The basename of the scripts that are embedded into CONSTANTS.go
// by `esc` during the generate phase.  In order to export these, there
// MUST be a corresponding PROXIED_MODULES entry for the base filename
// in resources/index.js
var customResourceScripts = []string{
	"sparta_utils.js",
	"proto/proxy_pb.js",
	"golang-constants.json"}

var golangCustomResourceTypes = []string{
	cloudformationresources.SESLambdaEventSource,
	cloudformationresources.S3LambdaEventSource,
	cloudformationresources.SNSLambdaEventSource,
	cloudformationresources.CloudWatchLogsLambdaEventSource,
	cloudformationresources.ZipToS3Bucket,
}

// PushSourceConfigurationActions map stores common IAM Policy Actions for Lambda
// push-source configuration management.
// The configuration is handled by CustomResources inserted into the generated
// CloudFormation template.
var PushSourceConfigurationActions = struct {
	SNSLambdaEventSource            []string
	S3LambdaEventSource             []string
	SESLambdaEventSource            []string
	CloudWatchLogsLambdaEventSource []string
}{
	SNSLambdaEventSource: []string{"sns:ConfirmSubscription",
		"sns:GetTopicAttributes",
		"sns:ListSubscriptionsByTopic",
		"sns:Subscribe",
		"sns:Unsubscribe"},
	S3LambdaEventSource: []string{"s3:GetBucketLocation",
		"s3:GetBucketNotification",
		"s3:PutBucketNotification",
		"s3:GetBucketNotificationConfiguration",
		"s3:PutBucketNotificationConfiguration"},
	SESLambdaEventSource: []string{"ses:CreateReceiptRuleSet",
		"ses:CreateReceiptRule",
		"ses:DeleteReceiptRule",
		"ses:DeleteReceiptRuleSet",
		"ses:DescribeReceiptRuleSet"},
	CloudWatchLogsLambdaEventSource: []string{"logs:DescribeSubscriptionFilters",
		"logs:DeleteSubscriptionFilter",
		"logs:PutSubscriptionFilter",
	},
}

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

func awsPrincipalToService(awsPrincipalName string) string {
	return strings.ToUpper(strings.SplitN(awsPrincipalName, ".", 2)[0])
}

func ensureCustomResourceHandler(serviceName string,
	useCGO bool,
	customResourceTypeName string,
	sourceArn *gocf.StringExpr,
	dependsOn []string,
	template *gocf.Template,
	S3Bucket string,
	S3Key string,
	logger *logrus.Logger) (string, error) {

	// AWS service basename
	awsServiceName := awsPrincipalToService(customResourceTypeName)

	// Use a stable resource CloudFormation resource name to represent
	// the single CustomResource that can configure the different
	// PushSource's for the given principal.
	keyName, err := json.Marshal(ArbitraryJSONObject{
		"Principal":   customResourceTypeName,
		"ServiceName": awsServiceName,
	})
	if err != nil {
		logger.Error("Failed to create configurator resource name: ", err.Error())
		return "", err
	}
	resourceBaseName := fmt.Sprintf("%sCustomResource", awsServiceName)
	subscriberHandlerName := CloudFormationResourceName(resourceBaseName, string(keyName))

	//////////////////////////////////////////////////////////////////////////////
	// IAM Role definition
	iamResourceName, err := ensureIAMRoleForCustomResource(customResourceTypeName,
		sourceArn,
		template,
		logger)
	if nil != err {
		return "", err
	}
	iamRoleRef := gocf.GetAtt(iamResourceName, "Arn")
	_, exists := template.Resources[subscriberHandlerName]
	if !exists {
		logger.WithFields(logrus.Fields{
			"Service": customResourceTypeName,
		}).Debug("Including Lambda CustomResource for AWS Service")

		configuratorDescription := customResourceDescription(serviceName, customResourceTypeName)

		//////////////////////////////////////////////////////////////////////////////
		// Custom Resource Lambda Handler
		// The export name MUST correspond to the createForwarder entry that is dynamically
		// written into the index.js file during compile in createNewSpartaCustomResourceEntry

		handlerName := lambdaExportNameForCustomResourceType(customResourceTypeName)
		logger.WithFields(logrus.Fields{
			"CustomResourceType": customResourceTypeName,
			"ScriptExport":       handlerName,
		}).Debug("Sparta CloudFormation custom resource handler info")

		customResourceHandlerDef := gocf.LambdaFunction{
			Code: &gocf.LambdaFunctionCode{
				S3Bucket: gocf.String(S3Bucket),
				S3Key:    gocf.String(S3Key),
			},
			Description: gocf.String(configuratorDescription),
			Handler:     gocf.String(handlerName),
			Role:        iamRoleRef,
			Timeout:     gocf.Integer(30),
		}
		if useCGO {
			customResourceHandlerDef.Runtime = gocf.String(PythonVersion)
		} else {
			customResourceHandlerDef.Runtime = gocf.String(NodeJSVersion)
		}
		cfResource := template.AddResource(subscriberHandlerName, customResourceHandlerDef)
		if nil != dependsOn && (len(dependsOn) > 0) {
			cfResource.DependsOn = append(cfResource.DependsOn, dependsOn...)
		}
	}
	return subscriberHandlerName, nil
}

// ensureIAMRoleForCustomResource ensures that the single IAM::Role for a single
// AWS principal (eg, s3.*.*) exists, and includes statements for the given
// sourceArn.  Sparta uses a single IAM::Role for the CustomResource configuration
// lambda, which is the union of all Arns in the application.
func ensureIAMRoleForCustomResource(awsPrincipalName string,
	sourceArn *gocf.StringExpr,
	template *gocf.Template,
	logger *logrus.Logger) (string, error) {

	var principalActions []string
	switch awsPrincipalName {
	case cloudformationresources.SNSLambdaEventSource:
		principalActions = PushSourceConfigurationActions.SNSLambdaEventSource
	case cloudformationresources.S3LambdaEventSource:
		principalActions = PushSourceConfigurationActions.S3LambdaEventSource
	case cloudformationresources.SESLambdaEventSource:
		principalActions = PushSourceConfigurationActions.SESLambdaEventSource
	case cloudformationresources.CloudWatchLogsLambdaEventSource:
		principalActions = PushSourceConfigurationActions.CloudWatchLogsLambdaEventSource
	default:
		return "", fmt.Errorf("Unsupported principal for IAM role creation: %s", awsPrincipalName)
	}

	// What's the stable IAMRoleName?
	resourceBaseName := fmt.Sprintf("CustomResource%sIAMRole", awsPrincipalToService(awsPrincipalName))
	stableRoleName := CloudFormationResourceName(resourceBaseName, awsPrincipalName)

	// Ensure it exists, then check to see if this Source ARN is already specified...
	// Checking equality with Stringable?

	// Create a new Role
	var existingIAMRole *gocf.IAMRole
	existingResource, exists := template.Resources[stableRoleName]
	logger.WithFields(logrus.Fields{
		"PrincipalActions": principalActions,
		"SourceArn":        sourceArn,
	}).Debug("Ensuring IAM Role results")

	if !exists {
		// Insert the IAM role here.  We'll walk the policies data in the next section
		// to make sure that the sourceARN we have is in the list
		statements := CommonIAMStatements.Core

		iamPolicyList := gocf.IAMRolePolicyList{}
		iamPolicyList = append(iamPolicyList,
			gocf.IAMRolePolicy{
				PolicyDocument: ArbitraryJSONObject{
					"Version":   "2012-10-17",
					"Statement": statements,
				},
				PolicyName: gocf.String(fmt.Sprintf("%sPolicy", stableRoleName)),
			},
		)

		existingIAMRole = &gocf.IAMRole{
			AssumeRolePolicyDocument: AssumePolicyDocument,
			Policies:                 &iamPolicyList,
		}
		template.AddResource(stableRoleName, existingIAMRole)

		// Create a new IAM Role resource
		logger.WithFields(logrus.Fields{
			"RoleName": stableRoleName,
		}).Debug("Inserting IAM Role")
	} else {
		existingIAMRole = existingResource.Properties.(*gocf.IAMRole)
	}

	// Walk the existing statements
	if nil != existingIAMRole.Policies {
		for _, eachPolicy := range *existingIAMRole.Policies {
			policyDoc := eachPolicy.PolicyDocument.(ArbitraryJSONObject)
			statements := policyDoc["Statement"]
			for _, eachStatement := range statements.([]spartaIAM.PolicyStatement) {
				if sourceArn.String() == eachStatement.Resource.String() {

					logger.WithFields(logrus.Fields{
						"RoleName":  stableRoleName,
						"SourceArn": sourceArn.String(),
					}).Debug("SourceArn already exists for IAM Policy")
					return stableRoleName, nil
				}
			}
		}

		logger.WithFields(logrus.Fields{
			"RoleName": stableRoleName,
			"Action":   principalActions,
			"Resource": sourceArn,
		}).Debug("Inserting Actions for configuration ARN")

		// Add this statement to the first policy, iff the actions are non-empty
		if len(principalActions) > 0 {
			rootPolicy := (*existingIAMRole.Policies)[0]
			rootPolicyDoc := rootPolicy.PolicyDocument.(ArbitraryJSONObject)
			rootPolicyStatements := rootPolicyDoc["Statement"].([]spartaIAM.PolicyStatement)
			rootPolicyDoc["Statement"] = append(rootPolicyStatements, spartaIAM.PolicyStatement{
				Effect:   "Allow",
				Action:   principalActions,
				Resource: sourceArn,
			})
		}

		return stableRoleName, nil
	}

	return "", fmt.Errorf("Unable to find Policies entry for IAM role: %s", stableRoleName)
}

func writeCustomResources(zipWriter *zip.Writer,
	logger *logrus.Logger) error {
	for _, eachName := range customResourceScripts {
		resourceName := fmt.Sprintf("%s/%s", provisioningResourcesRelPath, eachName)
		resourceContent := _escFSMustString(false, resourceName)
		stringReader := strings.NewReader(resourceContent)
		embedWriter, errCreate := zipWriter.Create(eachName)
		if nil != errCreate {
			return errCreate
		}
		logger.WithFields(logrus.Fields{
			"Name": eachName,
		}).Debug("Script name")

		_, copyErr := io.Copy(embedWriter, stringReader)
		if nil != copyErr {
			return copyErr
		}
	}
	return nil
}

func createUserCustomResourceEntry(customResource *customResourceInfo, logger *logrus.Logger) string {
	// The resource name is a :: delimited one, so let's sanitize that
	// to make it a valid JS identifier
	logger.WithFields(logrus.Fields{
		"UserFunction":       customResource.userFunctionName,
		"NodeJSFunctionName": customResource.scriptExportHandlerName(),
	}).Debug("Registering User CustomResource function")

	primaryEntry := fmt.Sprintf("exports[\"%s\"] = createForwarder(\"/%s\");\n",
		customResource.scriptExportHandlerName(),
		customResource.userFunctionName)
	return primaryEntry
}

// Return a string representation of a JS function call that can be exposed
// to AWS Lambda
func createNewNodeJSProxyEntry(lambdaInfo *LambdaAWSInfo, logger *logrus.Logger) string {
	logger.WithFields(logrus.Fields{
		"FunctionName": lambdaInfo.lambdaFunctionName(),
		"ScriptName":   lambdaInfo.scriptExportHandlerName(),
	}).Info("Creating NodeJS/Sparta proxy function")

	// We do know the CF resource name here - could write this into
	// index.js and expose a GET localhost:9000/lambdaMetadata
	// which wraps up DescribeStackResource for the running
	// lambda function
	primaryEntry := fmt.Sprintf("exports[\"%s\"] = createForwarder(\"/%s\");\n",
		lambdaInfo.scriptExportHandlerName(),
		lambdaInfo.lambdaFunctionName())
	return primaryEntry
}

func createNewSpartaNodeJSCustomResourceEntry(resourceName string, logger *logrus.Logger) string {
	// The resource name is a :: delimited one, so let's sanitize that
	// to make it a valid JS identifier
	jsName := scriptExportNameForCustomResourceType(resourceName)
	primaryEntry := fmt.Sprintf("exports[\"%s\"] = createForwarder(\"/%s\");\n",
		jsName,
		resourceName)
	return primaryEntry
}

func insertNodeModulesArchive(provisioningResourcesRelPath string,
	zipWriter *zip.Writer,
	logger *logrus.Logger) error {

	nodeModulesZipRelName := fmt.Sprintf("%s/node_modules.zip", provisioningResourcesRelPath)
	nodeModuleBytes, err := _escFSByte(false, nodeModulesZipRelName)
	if nil == err {
		nodeModuleReader, errReader := zip.NewReader(bytes.NewReader(nodeModuleBytes), int64(len(nodeModuleBytes)))
		if errReader != nil {
			return errReader
		}
		logger.WithFields(logrus.Fields{
			"Name": nodeModulesZipRelName,
		}).Debug("Embedding CustomResource node_modules.zip")

		for _, zipFile := range nodeModuleReader.File {
			logger.WithFields(logrus.Fields{
				"FileName": zipFile.Name,
			}).Debug("node_modules file")

			embedWriter, errCreate := zipWriter.Create(zipFile.Name)
			if nil != errCreate {
				return errCreate
			}

			sourceReader, errOpen := zipFile.Open()
			if errOpen != nil {
				return errOpen
			}
			io.Copy(embedWriter, sourceReader)
		}
	} else {
		logger.WithFields(logrus.Fields{
			"Name":  nodeModulesZipRelName,
			"Error": err,
		}).Warn("Failed to load node_modules.zip for embedding")
	}
	return nil
}

func insertNodeJSProxyResources(serviceName string,
	executableOutput string,
	lambdaAWSInfos []*LambdaAWSInfo,
	zipWriter *zip.Writer,
	logger *logrus.Logger) error {

	// Add the string literal adapter, which requires us to add exported
	// functions to the end of index.js.  These NodeJS exports will be
	// linked to the AWS Lambda NodeJS function name, and are basically
	// automatically generated pass through proxies to the golang HTTP handler.
	nodeJSWriter, err := zipWriter.Create("index.js")
	if err != nil {
		return errors.New("Failed to create ZIP entry: index.js")
	}
	nodeJSSource := _escFSMustString(false, "/resources/provision/index.js")
	nodeJSSource += "\n// DO NOT EDIT - CONTENT UNTIL EOF IS AUTOMATICALLY GENERATED\n"

	handlerNames := make(map[string]bool)
	for _, eachLambda := range lambdaAWSInfos {
		if _, exists := handlerNames[eachLambda.scriptExportHandlerName()]; !exists {
			nodeJSSource += createNewNodeJSProxyEntry(eachLambda, logger)
			handlerNames[eachLambda.scriptExportHandlerName()] = true
		}

		// USER DEFINED RESOURCES
		for _, eachCustomResource := range eachLambda.customResources {
			if _, exists := handlerNames[eachCustomResource.scriptExportHandlerName()]; !exists {
				nodeJSSource += createUserCustomResourceEntry(eachCustomResource, logger)
				handlerNames[eachCustomResource.scriptExportHandlerName()] = true
			}
		}
	}
	// SPARTA CUSTOM RESOURCES
	for _, eachCustomResourceName := range golangCustomResourceTypes {
		nodeJSSource += createNewSpartaNodeJSCustomResourceEntry(eachCustomResourceName, logger)
	}

	// Finally, replace
	// 	SPARTA_BINARY_NAME = 'Sparta.lambda.amd64';
	// with the service binary name
	//nodeJSSource += fmt.Sprintf("SPARTA_BINARY_NAME='%s';\n", executableOutput)

	// And the service name
	nodeJSSource += fmt.Sprintf("SPARTA_SERVICE_NAME='%s';\n", serviceName)
	nodeJSSource += fmt.Sprintf("SPARTA_LOG_LEVEL='%s';\n", logger.Level.String())
	logger.WithFields(logrus.Fields{
		"index.js": nodeJSSource,
	}).Debug("Dynamically generated NodeJS adapter")

	stringReader := strings.NewReader(nodeJSSource)
	_, copyErr := io.Copy(nodeJSWriter, stringReader)
	if nil != copyErr {
		return copyErr
	}
	// Embed any node_modules
	insertErr := insertNodeModulesArchive("/resources/provision",
		zipWriter,
		logger)
	if insertErr != nil {
		return insertErr
	}

	// Next embed the custom resource scripts into the package.
	logger.Debug("Embedding CustomResource scripts")
	return writeCustomResources(zipWriter, logger)
}

func pythonFunctionEntry(scriptExportName string,
	lambdaFunctionName string,
	logger *logrus.Logger) string {
	logger.WithFields(logrus.Fields{
		"ScriptName": scriptExportName,
		"LambdaName": lambdaFunctionName,
	}).Debug("Registering Sparta Python function")

	return fmt.Sprintf(`def %s(event, context):
	return lambda_handler("%s", event, context)
`,
		scriptExportName,
		lambdaFunctionName)
}

// Return a string representation of a JS function call that can be exposed
// to AWS Lambda
/*
func createNewPythonProxyEntry(lambdaInfo *LambdaAWSInfo, logger *logrus.Logger) string {
	logger.WithFields(logrus.Fields{
		"FunctionName": lambdaInfo.lambdaFunctionName(),
		"ScriptName":   lambdaInfo.scriptExportHandlerName(),
	}).Info("Registering Sparta Python function")

	primaryEntry := fmt.Sprintf(`def %s(event, context):
		return lambda_handler(%s, event, context)
	`,
		lambdaInfo.scriptExportHandlerName(),
		lambdaInfo.lambdaFunctionName())
	return primaryEntry
}
*/

func createNewSpartaPythonCustomResourceEntry(resourceName string, logger *logrus.Logger) string {
	// The resource name is a :: delimited one, so let's sanitize that
	// to make it a valid JS identifier
	pyName := scriptExportNameForCustomResourceType(resourceName)
	return pythonFunctionEntry(pyName, resourceName, logger)
}

func insertPythonProxyResources(serviceName string,
	executableOutput string,
	lambdaAWSInfos []*LambdaAWSInfo,
	zipWriter *zip.Writer,
	logger *logrus.Logger) error {
	pythonWriter, err := zipWriter.Create("index.py")
	if err != nil {
		return errors.New("Failed to create ZIP entry: index.py")
	}

	pythonTemplate := _escFSMustString(false, "/resources/provision/index.template.py")
	pythonSource := "\n#DO NOT EDIT - CONTENT UNTIL EOF IS AUTOMATICALLY GENERATED\n"

	// Great, let's assemble all the Python function names, then
	// supply them to the template expansion to perform the final
	// magic
	handlerNames := make(map[string]bool)
	for _, eachLambda := range lambdaAWSInfos {
		if _, exists := handlerNames[eachLambda.scriptExportHandlerName()]; !exists {
			pythonSource += pythonFunctionEntry(eachLambda.scriptExportHandlerName(),
				eachLambda.lambdaFunctionName(),
				logger)
			handlerNames[eachLambda.scriptExportHandlerName()] = true
		}

		// USER DEFINED RESOURCES
		for _, eachCustomResource := range eachLambda.customResources {
			if _, exists := handlerNames[eachCustomResource.scriptExportHandlerName()]; !exists {
				pythonSource += pythonFunctionEntry(eachCustomResource.scriptExportHandlerName(),
					eachCustomResource.userFunctionName,
					logger)

				pythonSource += createUserCustomResourceEntry(eachCustomResource, logger)
				handlerNames[eachCustomResource.scriptExportHandlerName()] = true
			}
		}
	}

	// SPARTA CUSTOM RESOURCES
	for _, eachCustomResourceName := range golangCustomResourceTypes {
		pythonSource += createNewSpartaPythonCustomResourceEntry(eachCustomResourceName, logger)
	}

	// Finally, pump the index.template.py through
	// the Go template engine so that we can substitute the
	// library name and the python functions we've built up...
	data := struct {
		LogLevel        string
		LibraryName     string
		PythonFunctions string
	}{
		logger.Level.String(),
		executableOutput,
		pythonSource,
	}
	pyTemplate, pyTemplateErr := template.New("PythonHandler").Parse(pythonTemplate)
	if nil != pyTemplateErr {
		return pyTemplateErr
	}
	var pyDoc bytes.Buffer
	pyTemplateErr = pyTemplate.Execute(&pyDoc, data)
	if nil != pyTemplateErr {
		return pyTemplateErr
	}

	// Log the Python handler...
	logger.WithFields(logrus.Fields{
		"index.py": pyDoc.String(),
	}).Debug("Dynamically generated Python ctypes adapter")

	_, copyErr := io.WriteString(pythonWriter, pyDoc.String())
	return copyErr
}

func systemGoVersion(logger *logrus.Logger) (string, error) {
	runtimeVersion := runtime.Version()
	// Get the golang version from the output:
	// Matts-MBP:Sparta mweagle$ go version
	// go version go1.8.1 darwin/amd64
	golangVersionRE := regexp.MustCompile(`go(\d+\.\d+(\.\d+)?)`)
	matches := golangVersionRE.FindStringSubmatch(runtimeVersion)
	if len(matches) > 2 {
		return matches[1], nil
	}
	logger.WithFields(logrus.Fields{
		"Output": runtimeVersion,
	}).Warn("Unable to find Golang version using RegExp - using current version")
	return runtimeVersion, nil
}
