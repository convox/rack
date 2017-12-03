package sparta

import (
	"archive/zip"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws/session"
	_ "github.com/aws/aws-sdk-go/service/ecr" // Ref to have Glide include depends
	spartaCF "github.com/mweagle/Sparta/aws/cloudformation"
	_ "github.com/mweagle/Sparta/aws/dynamodb" // Ref to have Glide include depends
	spartaIAM "github.com/mweagle/Sparta/aws/iam"
	gocf "github.com/mweagle/go-cloudformation"
)

////////////////////////////////////////////////////////////////////////////////
// Constants
////////////////////////////////////////////////////////////////////////////////

const (
	// SpartaVersion defines the current Sparta release
	SpartaVersion = "0.20.4"
	// NodeJSVersion is the Node JS runtime used for the shim layer
	NodeJSVersion = "nodejs6.10"
	// PythonVersion is the Python version used for CGO support
	PythonVersion = "python3.6"
	// Custom Resource typename used to create new cloudFormationUserDefinedFunctionCustomResource
	cloudFormationLambda = "Custom::SpartaLambdaCustomResource"
	// divider length is the length of a divider in the text
	// based CLI output
	dividerLength = 62
)

var (
	// internal logging header
	headerDivider = strings.Repeat("═", dividerLength)
)

// AWS Principal ARNs from http://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html
// See also
// http://docs.aws.amazon.com/general/latest/gr/rande.html
// for region specific principal names
const (
	// @enum AWSPrincipal
	APIGatewayPrincipal = "apigateway.amazonaws.com"
	// @enum AWSPrincipal
	CloudWatchEventsPrincipal = "events.amazonaws.com"
	// @enum AWSPrincipal
	SESPrincipal = "ses.amazonaws.com"
	// @enum AWSPrincipal
	SNSPrincipal = "sns.amazonaws.com"
	// @enum AWSPrincipal
	EC2Principal = "ec2.amazonaws.com"
	// @enum AWSPrincipal
	LambdaPrincipal = "lambda.amazonaws.com"
)

type cloudFormationLambdaCustomResource struct {
	gocf.CloudFormationCustomResource
	ServiceToken   *gocf.StringExpr
	UserProperties map[string]interface{} `json:",omitempty"`
}

func customResourceProvider(resourceType string) gocf.ResourceProperties {
	switch resourceType {
	case cloudFormationLambda:
		{
			return &cloudFormationLambdaCustomResource{}
		}
	default:
		return nil
	}
}

func init() {
	gocf.RegisterCustomResourceProvider(customResourceProvider)
	rand.Seed(time.Now().Unix())
}

////////////////////////////////////////////////////////////////////////////////
// Variables
////////////////////////////////////////////////////////////////////////////////

// Represents the CloudFormation Arn of this stack, referenced
// in CommonIAMStatements
var cloudFormationThisStackArn = []gocf.Stringable{gocf.String("arn:aws:cloudformation:"),
	gocf.Ref("AWS::Region").String(),
	gocf.String(":"),
	gocf.Ref("AWS::AccountId").String(),
	gocf.String(":stack/"),
	gocf.Ref("AWS::StackName").String(),
	gocf.String("/*")}

// CommonIAMStatements defines common IAM::Role Policy Statement values for different AWS
// service types.  See http://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html#genref-aws-service-namespaces
// for names.
// http://docs.aws.amazon.com/lambda/latest/dg/monitoring-functions.html
// for more information.
var CommonIAMStatements = struct {
	Core     []spartaIAM.PolicyStatement
	VPC      []spartaIAM.PolicyStatement
	DynamoDB []spartaIAM.PolicyStatement
	Kinesis  []spartaIAM.PolicyStatement
}{
	Core: []spartaIAM.PolicyStatement{
		{
			Action: []string{"logs:CreateLogGroup",
				"logs:CreateLogStream",
				"logs:PutLogEvents"},
			Effect: "Allow",
			Resource: gocf.Join("",
				gocf.String("arn:aws:logs:"),
				gocf.Ref("AWS::Region"),
				gocf.String(":"),
				gocf.Ref("AWS::AccountId"),
				gocf.String("*")),
		},
		{
			Action:   []string{"cloudwatch:PutMetricData"},
			Effect:   "Allow",
			Resource: wildcardArn,
		},
		{
			Effect: "Allow",
			Action: []string{"cloudformation:DescribeStacks",
				"cloudformation:DescribeStackResource"},
			Resource: gocf.Join("", cloudFormationThisStackArn...),
		},
		// http://docs.aws.amazon.com/lambda/latest/dg/lambda-x-ray.html#enabling-x-ray
		{
			Effect: "Allow",
			Action: []string{"xray:PutTraceSegments",
				"xray:PutTelemetryRecords"},
			Resource: gocf.String("*"),
		},
	},
	VPC: []spartaIAM.PolicyStatement{
		{
			Action: []string{"ec2:CreateNetworkInterface",
				"ec2:DescribeNetworkInterfaces",
				"ec2:DeleteNetworkInterface"},
			Effect:   "Allow",
			Resource: wildcardArn,
		},
	},
	DynamoDB: []spartaIAM.PolicyStatement{
		{
			Effect: "Allow",
			Action: []string{"dynamodb:DescribeStream",
				"dynamodb:GetRecords",
				"dynamodb:GetShardIterator",
				"dynamodb:ListStreams",
			},
		},
	},
	Kinesis: []spartaIAM.PolicyStatement{
		{
			Effect: "Allow",
			Action: []string{"kinesis:GetRecords",
				"kinesis:GetShardIterator",
				"kinesis:DescribeStream",
				"kinesis:ListStreams",
			},
		},
	},
}

// RE for sanitizing golang/JS layer
var reSanitize = regexp.MustCompile(`\W+`)

// Wildcard ARN for any AWS resource
var wildcardArn = gocf.String("*")

// AssumePolicyDocument defines common a IAM::Role PolicyDocument
// used as part of IAM::Role resource definitions
var AssumePolicyDocument = ArbitraryJSONObject{
	"Version": "2012-10-17",
	"Statement": []ArbitraryJSONObject{
		{
			"Effect": "Allow",
			"Principal": ArbitraryJSONObject{
				"Service": []string{LambdaPrincipal},
			},
			"Action": []string{"sts:AssumeRole"},
		},
		{
			"Effect": "Allow",
			"Principal": ArbitraryJSONObject{
				"Service": []string{EC2Principal},
			},
			"Action": []string{"sts:AssumeRole"},
		},
		{
			"Effect": "Allow",
			"Principal": ArbitraryJSONObject{
				"Service": []string{APIGatewayPrincipal},
			},
			"Action": []string{"sts:AssumeRole"},
		},
	},
}

////////////////////////////////////////////////////////////////////////////////
// Types
////////////////////////////////////////////////////////////////////////////////

// CustomResourceFunction represents a user-defined function that is used
// as a CloudFormation lambda backed resource target
type CustomResourceFunction func(requestType string,
	stackID string,
	properties map[string]interface{},
	logger *logrus.Logger) (map[string]interface{}, error)

// ArbitraryJSONObject represents an untyped key-value object. CloudFormation resource representations
// are aggregated as []ArbitraryJSONObject before being marsharled to JSON
// for API operations.
type ArbitraryJSONObject map[string]interface{}

// Package private type to deserialize NodeJS proxied
// Lambda Event and Context information
type lambdaRequest struct {
	Event   json.RawMessage
	Context LambdaContext
}

// LambdaContext defines the AWS Lambda Context object provided by the AWS Lambda runtime.
// See http://docs.aws.amazon.com/lambda/latest/dg/nodejs-prog-model-context.html
// for more information on field values.  Note that the golang version doesn't functions
// defined on the Context object.
type LambdaContext struct {
	FunctionName       string `json:"functionName"`
	FunctionVersion    string `json:"functionVersion"`
	InvokedFunctionARN string `json:"invokedFunctionArn"`
	MemoryLimitInMB    string `json:"memoryLimitInMB"`
	AWSRequestID       string `json:"awsRequestId"`
	LogGroupName       string `json:"logGroupName"`
	LogStreamName      string `json:"logStreamName"`
}

// LambdaFunction is the golang function signature required to support AWS Lambda execution.
// Standard HTTP response codes are used to signal AWS Lambda success/failure on the
// proxied context() object.  See http://docs.aws.amazon.com/lambda/latest/dg/nodejs-prog-model-context.html
// for more information.
//
// 	200 - 299       : Success
// 	<200 || >= 300  : Failure
//
// Content written to the ResponseWriter will be used as the
// response/Error value provided to AWS Lambda.
type LambdaFunction func(*json.RawMessage, *LambdaContext, http.ResponseWriter, *logrus.Logger)

// HTTPLambdaFunction is a more Go-friendly HTTP handler definition
type HTTPLambdaFunction func(http.ResponseWriter, *http.Request)

// LambdaFunctionOptions defines additional AWS Lambda execution params.  See the
// AWS Lambda FunctionConfiguration (http://docs.aws.amazon.com/lambda/latest/dg/API_FunctionConfiguration.html)
// docs for more information. Note that the "Runtime" field will be automatically set
// to "nodejs4.3" (at least until golang is officially supported). See
// http://docs.aws.amazon.com/lambda/latest/dg/programming-model.html
type LambdaFunctionOptions struct {
	// Additional function description
	Description string
	// Memory limit
	MemorySize int64
	// Timeout (seconds)
	Timeout int64
	// VPC Settings
	VpcConfig *gocf.LambdaFunctionVPCConfig
	// Environment Variables
	Environment map[string]*gocf.StringExpr
	// KMS Key Arn used to encrypt environment variables
	KmsKeyArn string
	// Tags to associate with the Lambda function
	Tags map[string]string
	// Tracing options for XRay
	TracingConfig *gocf.LambdaFunctionTracingConfig
	// Additional params
	SpartaOptions *SpartaOptions
}

func defaultLambdaFunctionOptions() *LambdaFunctionOptions {
	return &LambdaFunctionOptions{Description: "",
		MemorySize:    128,
		Timeout:       3,
		VpcConfig:     nil,
		Environment:   nil,
		KmsKeyArn:     "",
		SpartaOptions: nil,
	}
}

// SpartaOptions allow the passing in of additional options during the creation of a Lambda Function
type SpartaOptions struct {
	// User supplied function name to use for
	// http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-lambda-function.html#cfn-lambda-function-functionname
	// value. If this is not supplied, a reflection-based
	// name will be automatically used.
	Name string
}

// TemplateDecorator allows Lambda functions to annotate the CloudFormation
// template definition.  Both the resources and the outputs params
// are initialized to an empty ArbitraryJSONObject and should
// be populated with valid CloudFormation ArbitraryJSONObject values.  The
// CloudFormationResourceName() function can be used to generate
// logical CloudFormation-compatible resource names.
// See http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-template-resource-type-ref.html and
// http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/outputs-section-structure.html for
// more information.
type TemplateDecorator func(serviceName string,
	lambdaResourceName string,
	lambdaResource gocf.LambdaFunction,
	resourceMetadata map[string]interface{},
	S3Bucket string,
	S3Key string,
	buildID string,
	template *gocf.Template,
	context map[string]interface{},
	logger *logrus.Logger) error

// WorkflowHook defines a user function that should be called at a specific
// point in the larger Sparta workflow. The first argument is a map that
// is shared across all LifecycleHooks and which Sparta treats as an opaque
// value.
type WorkflowHook func(context map[string]interface{},
	serviceName string,
	S3Bucket string,
	buildID string,
	awsSession *session.Session,
	noop bool,
	logger *logrus.Logger) error

// ServiceDecoratorHook defines a user function that is called a single
// time in the marshall workflow.
type ServiceDecoratorHook func(context map[string]interface{},
	serviceName string,
	template *gocf.Template,
	S3Bucket string,
	buildID string,
	awsSession *session.Session,
	noop bool,
	logger *logrus.Logger) error

// ArchiveHook provides callers an opportunity to insert additional
// files into the ZIP archive deployed to S3
type ArchiveHook func(context map[string]interface{},
	serviceName string,
	zipWriter *zip.Writer,
	awsSession *session.Session,
	noop bool,
	logger *logrus.Logger) error

// RollbackHook provides callers an opportunity to handle failures
// associated with failing to perform the requested operation
type RollbackHook func(context map[string]interface{},
	serviceName string,
	awsSession *session.Session,
	noop bool,
	logger *logrus.Logger)

// WorkflowHooks is a structure that allows callers to customize the Sparta provisioning
// pipeline to add contents the Lambda archive or perform other workflow operations.
type WorkflowHooks struct {
	// Initial hook context. May be empty
	Context map[string]interface{}
	// PreBuild is called before the current Sparta-binary is compiled
	PreBuild WorkflowHook
	// PostBuild is called after the current Sparta-binary is compiled
	PostBuild WorkflowHook
	// ArchiveHook is called after Sparta has populated the ZIP archive containing the
	// AWS Lambda code package and before the ZIP writer is closed.  Define this hook
	// to add additional resource files to your Lambda package
	Archive ArchiveHook
	// PreMarshall is called before Sparta marshalls the application contents to a CloudFormation template
	PreMarshall WorkflowHook
	// ServiceDecorator is called before Sparta marshalls the CloudFormation template
	ServiceDecorator ServiceDecoratorHook
	// PostMarshall is called after Sparta marshalls the application contents to a CloudFormation template
	PostMarshall WorkflowHook
	// Rollback is called if there is an error performing the requested operation
	Rollback RollbackHook
}

////////////////////////////////////////////////////////////////////////////////
// START - IAMRolePrivilege
//

// IAMRolePrivilege struct stores data necessary to create an IAM Policy Document
// as part of the inline IAM::Role resource definition.  See
// http://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies.html
// for more information
type IAMRolePrivilege struct {
	// What actions you will allow.
	// Each AWS service has its own set of actions.
	// For example, you might allow a user to use the Amazon S3 ListBucket action,
	// which returns information about the items in a bucket.
	// Any actions that you don't explicitly allow are denied.
	Actions []string
	// Which resources you allow the action on. For example, what specific Amazon
	// S3 buckets will you allow the user to perform the ListBucket action on?
	// Users cannot access any resources that you have not explicitly granted
	// permissions to.
	Resource interface{}
}

func (rolePrivilege *IAMRolePrivilege) resourceExpr() *gocf.StringExpr {
	switch rolePrivilege.Resource.(type) {
	case string:
		return gocf.String(rolePrivilege.Resource.(string))
	default:
		return rolePrivilege.Resource.(*gocf.StringExpr)
	}
}

// IAMRoleDefinition stores a slice of IAMRolePrivilege values
// to "Allow" for the given IAM::Role.
// Note that the CommonIAMStatements will be automatically included and do
// not need to be multiply specified.
type IAMRoleDefinition struct {
	// Slice of IAMRolePrivilege entries
	Privileges []IAMRolePrivilege
	// Cached logical resource name
	cachedLogicalName string
}

func (roleDefinition *IAMRoleDefinition) toResource(eventSourceMappings []*EventSourceMapping,
	options *LambdaFunctionOptions,
	logger *logrus.Logger) gocf.IAMRole {

	statements := CommonIAMStatements.Core
	for _, eachPrivilege := range roleDefinition.Privileges {
		policyStatement := spartaIAM.PolicyStatement{
			Effect:   "Allow",
			Action:   eachPrivilege.Actions,
			Resource: eachPrivilege.resourceExpr(),
		}
		statements = append(statements, policyStatement)
	}

	// Add VPC permissions iff needed
	if options != nil && options.VpcConfig != nil {
		statements = append(statements, CommonIAMStatements.VPC...)
	}

	// http://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html
	for _, eachEventSourceMapping := range eventSourceMappings {
		arnParts := strings.Split(eachEventSourceMapping.EventSourceArn, ":")
		// 3rd slot is service scope
		if len(arnParts) >= 2 {
			awsService := arnParts[2]
			logger.Debug("Looking up common IAM privileges for EventSource: ", awsService)
			switch awsService {
			case "dynamodb":
				for _, statement := range CommonIAMStatements.DynamoDB {
					statement.Resource = gocf.String(eachEventSourceMapping.EventSourceArn)
					statements = append(statements, statement)
				}
			case "kinesis":
				for _, statement := range CommonIAMStatements.Kinesis {
					statement.Resource = gocf.String(eachEventSourceMapping.EventSourceArn)
					statements = append(statements, statement)
				}
			default:
				logger.Debug("No additional statements found")
			}
		}
	}

	iamPolicies := gocf.IAMRolePolicyList{}
	iamPolicies = append(iamPolicies, gocf.IAMRolePolicy{
		PolicyDocument: ArbitraryJSONObject{
			"Version":   "2012-10-17",
			"Statement": statements,
		},
		PolicyName: gocf.String("LambdaPolicy"),
	})
	return gocf.IAMRole{
		AssumeRolePolicyDocument: AssumePolicyDocument,
		Policies:                 &iamPolicies,
	}
}

// Returns the stable logical name for this IAMRoleDefinition, which depends on the serviceName
// and owning targetLambdaFnName.  This potentially creates semantically equivalent IAM::Role entries
// from the same struct pointer, so:
// TODO: Create a canonical IAMRoleDefinition serialization that can be used as the digest source
func (roleDefinition *IAMRoleDefinition) logicalName(serviceName string, targetLambdaFnName string) string {
	if "" == roleDefinition.cachedLogicalName {
		roleDefinition.cachedLogicalName = CloudFormationResourceName("IAMRole", serviceName, targetLambdaFnName)
	}
	return roleDefinition.cachedLogicalName
}

//
// END - IAMRolePrivilege
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// START - EventSourceMapping

// EventSourceMapping specifies data necessary for pull-based configuration. The fields
// directly correspond to the golang AWS SDK's CreateEventSourceMappingInput
// (http://docs.aws.amazon.com/sdk-for-go/api/service/lambda.html#type-CreateEventSourceMappingInput)
type EventSourceMapping struct {
	StartingPosition string
	EventSourceArn   string
	Disabled         bool
	BatchSize        int64
}

func (mapping *EventSourceMapping) export(serviceName string,
	targetLambda *gocf.StringExpr,
	S3Bucket string,
	S3Key string,
	template *gocf.Template,
	logger *logrus.Logger) error {

	eventSourceMappingResource := gocf.LambdaEventSourceMapping{
		EventSourceArn:   gocf.String(mapping.EventSourceArn),
		FunctionName:     targetLambda,
		StartingPosition: gocf.String(mapping.StartingPosition),
		BatchSize:        gocf.Integer(mapping.BatchSize),
		Enabled:          gocf.Bool(!mapping.Disabled),
	}

	hash := sha1.New()
	hash.Write([]byte(mapping.EventSourceArn))
	binary.Write(hash, binary.LittleEndian, mapping.BatchSize)
	hash.Write([]byte(mapping.StartingPosition))
	resourceName := fmt.Sprintf("LambdaES%s", hex.EncodeToString(hash.Sum(nil)))
	template.AddResource(resourceName, eventSourceMappingResource)
	return nil
}

//
// END - EventSourceMapping
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// START - customResourceInfo

// customResourceInfo wraps up information about any userDefined CloudFormation
// user-defined Resources
type customResourceInfo struct {
	roleDefinition   *IAMRoleDefinition
	roleName         string
	userFunction     CustomResourceFunction
	userFunctionName string
	options          *LambdaFunctionOptions
	properties       map[string]interface{}
}

// Returns a JavaScript compatible function name for the golang function name.  This
// value will be used as the URL path component for the HTTP proxying layer.
func (resourceInfo *customResourceInfo) scriptExportHandlerName() string {
	// The JS handler name must take into account the
	return sanitizedName(resourceInfo.userFunctionName)
}

// Returns the stable CloudFormation resource logical name for this resource.  For
// a CustomResource, this name corresponds to the AWS::CloudFormation::CustomResource
// invocation of the Lambda function, not the lambda function itself
func (resourceInfo *customResourceInfo) logicalName() string {
	hash := sha1.New()
	// The name has to be stable so that the ServiceToken value which is
	// part the CustomResource invocation doesn't change during stack updates. CF
	// will throw an error if the ServiceToken changes across updates.
	source := fmt.Sprintf("%#v", resourceInfo.userFunctionName)
	hash.Write([]byte(source))
	return CloudFormationResourceName(resourceInfo.userFunctionName, hex.EncodeToString(hash.Sum(nil)))
}

func (resourceInfo *customResourceInfo) export(serviceName string,
	targetLambda *gocf.StringExpr,
	runtime string,
	S3Bucket string,
	S3Key string,
	roleNameMap map[string]*gocf.StringExpr,
	template *gocf.Template,
	logger *logrus.Logger) error {

	// Figure out the role name
	iamRoleArnName := resourceInfo.roleName

	// If there is no user supplied role, that means that the associated
	// IAMRoleDefinition name has been created and this resource needs to
	// depend on that being created.
	if iamRoleArnName == "" && resourceInfo.roleDefinition != nil {
		iamRoleArnName = resourceInfo.roleDefinition.logicalName(serviceName, resourceInfo.userFunctionName)
	}
	lambdaDescription := resourceInfo.options.Description
	if "" == lambdaDescription {
		lambdaDescription = fmt.Sprintf("%s CustomResource: %s", serviceName, resourceInfo.userFunctionName)
	}

	// Create the Lambda Function
	lambdaResource := gocf.LambdaFunction{
		Code: &gocf.LambdaFunctionCode{
			S3Bucket: gocf.String(S3Bucket),
			S3Key:    gocf.String(S3Key),
		},
		Description: gocf.String(lambdaDescription),
		Handler:     gocf.String(fmt.Sprintf("index.%s", resourceInfo.scriptExportHandlerName())),
		MemorySize:  gocf.Integer(resourceInfo.options.MemorySize),
		Role:        roleNameMap[iamRoleArnName],
		Runtime:     gocf.String(runtime),
		Timeout:     gocf.Integer(resourceInfo.options.Timeout),
		VPCConfig:   resourceInfo.options.VpcConfig,
	}

	lambdaFunctionCFName := CloudFormationResourceName("CustomResourceLambda",
		resourceInfo.userFunctionName,
		resourceInfo.logicalName())

	cfResource := template.AddResource(lambdaFunctionCFName, lambdaResource)
	safeMetadataInsert(cfResource, "golangFunc", resourceInfo.userFunctionName)

	// And create the CustomResource that actually invokes it...
	newResource, newResourceError := newCloudFormationResource(cloudFormationLambda, logger)
	if nil != newResourceError {
		return newResourceError
	}
	customResource := newResource.(*cloudFormationLambdaCustomResource)
	customResource.ServiceToken = gocf.GetAtt(lambdaFunctionCFName, "Arn")
	customResource.UserProperties = resourceInfo.properties
	template.AddResource(resourceInfo.logicalName(), customResource)
	return nil
}

// END - customResourceInfo
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// START - LambdaAWSInfo

// LambdaAWSInfo stores all data necessary to provision a golang-based AWS Lambda function.
type LambdaAWSInfo struct {
	// pointer to lambda function
	lambdaFn LambdaFunction
	// The user supplied internal name
	userSuppliedFunctionName string
	// HTTP handler function
	httpHandler http.Handler
	// Role name (NOT ARN) to use during AWS Lambda Execution.  See
	// the FunctionConfiguration (http://docs.aws.amazon.com/lambda/latest/dg/API_FunctionConfiguration.html)
	// docs for more info.
	// Note that either `RoleName` or `RoleDefinition` must be supplied
	RoleName string
	// IAM Role Definition if the stack should implicitly create an IAM role for
	// lambda execution. Note that either `RoleName` or `RoleDefinition` must be supplied
	RoleDefinition *IAMRoleDefinition
	// Additional exeuction options
	Options *LambdaFunctionOptions
	// Permissions to enable push-based Lambda execution.  See the
	// Permission Model docs (http://docs.aws.amazon.com/lambda/latest/dg/intro-permission-model.html)
	// for more information.
	Permissions []LambdaPermissionExporter
	// EventSource mappings to enable for pull-based Lambda execution.  See the
	// Event Source docs (http://docs.aws.amazon.com/lambda/latest/dg/intro-core-components.html)
	// for more information
	EventSourceMappings []*EventSourceMapping
	// Template decorator. If defined, the decorator will be called to insert additional
	// resources on behalf of this lambda function
	Decorator TemplateDecorator
	// Optional array of infrastructure resource logical names, typically
	// defined by a TemplateDecorator, that this lambda depends on
	DependsOn []string
	// Slice of customResourceInfo pointers for any associated CloudFormation
	// CustomResources associated with this lambda
	customResources []*customResourceInfo
	// Cached lambda name s.t. we only compute it once
	cachedLambdaFunctionName string
}

// lambdaFunctionName returns the internal script-sanitized
// function name for lambda export binding
func (info *LambdaAWSInfo) lambdaFunctionName() string {
	if info.cachedLambdaFunctionName != "" {
		return info.cachedLambdaFunctionName
	}
	lambdaFuncName := info.userSuppliedFunctionName
	if nil != info.Options &&
		nil != info.Options.SpartaOptions &&
		"" != info.Options.SpartaOptions.Name {
		lambdaFuncName = info.Options.SpartaOptions.Name
	} else {
		// Using the default name, let's at least remove the
		// first prefix, since that's the SCM provider and
		// doesn't provide a lot of value...
		if info.lambdaFn != nil {
			lambdaPtr := runtime.FuncForPC(reflect.ValueOf(info.lambdaFn).Pointer())
			lambdaFuncName = lambdaPtr.Name()
		}

		// Split
		// cwd: /Users/mweagle/Documents/gopath/src/github.com/mweagle/SpartaHelloWorld
		// anonymous: github.com/mweagle/Sparta.(*StructHandler1).(github.com/mweagle/Sparta.handler)-fm
		//	RE==> var reSplit = regexp.MustCompile("[\\(\\)\\.\\*]+")
		// 	RESULT ==> Hello,[github com/mweagle/Sparta StructHandler1 github com/mweagle/Sparta handler -fm]
		// Same package: main.helloWorld
		// Other package, free function: github.com/mweagle/SpartaPython.HelloWorld

		// Grab the name of the function...
		structDefined := strings.Contains(lambdaFuncName, "(") || strings.Contains(lambdaFuncName, ")")
		otherPackage := strings.Contains(lambdaFuncName, "/")
		canonicalName := lambdaFuncName
		if structDefined {
			var reCapture = regexp.MustCompile(`\(([^\(\)]+)\)`)
			parts := reCapture.FindAllString(lambdaFuncName, -1)
			// (*StructHandler1),(github.com/mweagle/Sparta.handler)
			funcNameParts := strings.Split(parts[1], "/")
			intermediateName := fmt.Sprintf("%s-%s", parts[0], funcNameParts[len(funcNameParts)-1])
			reClean := regexp.MustCompile(`[\*\(\)]+`)
			canonicalName = reClean.ReplaceAllString(intermediateName, "")
		} else if otherPackage {
			parts := strings.Split(lambdaFuncName, "/")
			canonicalName = parts[len(parts)-1]
		}
		// Final sanitization
		// Issue: https://github.com/mweagle/Sparta/issues/63
		lambdaFuncName = sanitizedName(canonicalName)
	}
	// Cache it so we only do this once
	info.cachedLambdaFunctionName = lambdaFuncName
	return info.cachedLambdaFunctionName
}

// URLPath returns the URL path that can be used as an argument
// to NewLambdaRequest or NewAPIGatewayRequest
func (info *LambdaAWSInfo) URLPath() string {
	return info.lambdaFunctionName()
}

// RequireCustomResource adds a Lambda-backed CustomResource entry to the CloudFormation
// template. This function will be made a dependency of the owning Lambda function.
// The returned string is the custom resource's CloudFormation logical resource
// name that can be used for `Fn:GetAtt` calls for metadata lookups
func (info *LambdaAWSInfo) RequireCustomResource(roleNameOrIAMRoleDefinition interface{},
	userFunc CustomResourceFunction,
	lambdaOptions *LambdaFunctionOptions,
	resourceProps map[string]interface{}) (string, error) {
	if nil == userFunc {
		return "", fmt.Errorf("RequireCustomResource userFunc must not be nil")
	}
	if nil == lambdaOptions {
		lambdaOptions = defaultLambdaFunctionOptions()
	}
	funcPtr := runtime.FuncForPC(reflect.ValueOf(userFunc).Pointer())
	resourceInfo := &customResourceInfo{
		userFunction:     userFunc,
		userFunctionName: funcPtr.Name(),
		options:          lambdaOptions,
		properties:       resourceProps,
	}
	switch v := roleNameOrIAMRoleDefinition.(type) {
	case string:
		resourceInfo.roleName = roleNameOrIAMRoleDefinition.(string)
	case IAMRoleDefinition:
		definition := roleNameOrIAMRoleDefinition.(IAMRoleDefinition)
		resourceInfo.roleDefinition = &definition
	default:
		panic(fmt.Sprintf("Unsupported IAM Role type: %s", v))
	}
	info.customResources = append(info.customResources, resourceInfo)
	info.DependsOn = append(info.DependsOn, resourceInfo.logicalName())
	return resourceInfo.logicalName(), nil
}

// Returns a script compatible function name for the golang function name.  This
// value will be used as the URL path component for the HTTP proxying layer.
func (info *LambdaAWSInfo) scriptExportHandlerName() string {
	return sanitizedName(info.lambdaFunctionName())
}

// Returns the stable logical name for this LambdaAWSInfo value
func (info *LambdaAWSInfo) logicalName() string {
	// Per http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/resources-section-structure.html,
	// we can only use alphanumeric, so we'll take the sanitized name and
	// remove all underscores
	// Prefer the user-supplied stable name to the internal one.
	baseName := info.lambdaFunctionName()
	resourceName := strings.Replace(sanitizedName(baseName), "_", "", -1)
	prefix := fmt.Sprintf("%sLambda", resourceName)
	return CloudFormationResourceName(prefix, info.lambdaFunctionName())
}

// Marshal this object into 1 or more CloudFormation resource definitions that are accumulated
// in the resources map
func (info *LambdaAWSInfo) export(serviceName string,
	useCGO bool,
	lambdaRuntime string,
	S3Bucket string,
	S3Key string,
	S3Version string,
	buildID string,
	roleNameMap map[string]*gocf.StringExpr,
	template *gocf.Template,
	context map[string]interface{},
	logger *logrus.Logger) error {

	// If we have RoleName, then get the ARN, otherwise get the Ref
	var dependsOn []string
	if nil != info.DependsOn {
		dependsOn = append(dependsOn, info.DependsOn...)
	}

	iamRoleArnName := info.RoleName

	// If there is no user supplied role, that means that the associated
	// IAMRoleDefinition name has been created and this resource needs to
	// depend on that being created.
	if iamRoleArnName == "" && info.RoleDefinition != nil {
		iamRoleArnName = info.RoleDefinition.logicalName(serviceName, info.lambdaFunctionName())
		dependsOn = append(dependsOn, info.RoleDefinition.logicalName(serviceName, info.lambdaFunctionName()))
	}
	lambdaDescription := info.Options.Description
	if "" == lambdaDescription {
		lambdaDescription = fmt.Sprintf("%s: %s", serviceName, info.lambdaFunctionName())
	}

	// Create the primary resource
	lambdaResource := gocf.LambdaFunction{
		Code: &gocf.LambdaFunctionCode{
			S3Bucket: gocf.String(S3Bucket),
			S3Key:    gocf.String(S3Key),
		},
		Description: gocf.String(lambdaDescription),
		Handler:     gocf.String(fmt.Sprintf("index.%s", info.scriptExportHandlerName())),
		MemorySize:  gocf.Integer(info.Options.MemorySize),
		Role:        roleNameMap[iamRoleArnName],
		Runtime:     gocf.String(lambdaRuntime),
		Timeout:     gocf.Integer(info.Options.Timeout),
		VPCConfig:   info.Options.VpcConfig,
	}
	if "" != S3Version {
		lambdaResource.Code.S3ObjectVersion = gocf.String(S3Version)
	}
	if "" != info.Options.KmsKeyArn {
		lambdaResource.KmsKeyArn = gocf.String(info.Options.KmsKeyArn)
	}
	if nil != info.Options.Tags {
		tagList := gocf.TagList{}
		for eachKey, eachValue := range info.Options.Tags {
			tagList = append(tagList, gocf.Tag{
				Key:   gocf.String(eachKey),
				Value: gocf.String(eachValue),
			})
		}
		lambdaResource.Tags = &tagList
	}
	if nil != info.Options.TracingConfig {
		lambdaResource.TracingConfig = info.Options.TracingConfig
	}

	if nil != info.Options.Environment {
		lambdaResource.Environment = &gocf.LambdaFunctionEnvironment{
			Variables: info.Options.Environment,
		}
	}
	// Need to check if a functionName exists in the LambdaAwsInfo struct
	// If an empty string is passed, the template will error with invalid
	// function name.
	lambdaResource.FunctionName = gocf.Join("-",
		gocf.Ref("AWS::StackName"),
		gocf.String(info.lambdaFunctionName()))
	cfResource := template.AddResource(info.logicalName(), lambdaResource)
	cfResource.DependsOn = append(cfResource.DependsOn, dependsOn...)
	safeMetadataInsert(cfResource, "golangFunc", info.lambdaFunctionName())

	// Create the lambda Ref in case we need a permission or event mapping
	functionAttr := gocf.GetAtt(info.logicalName(), "Arn")

	// Permissions
	for _, eachPermission := range info.Permissions {
		_, err := eachPermission.export(serviceName,
			useCGO,
			info.lambdaFunctionName(),
			info.logicalName(),
			template,
			S3Bucket,
			S3Key,
			logger)
		if nil != err {
			return err
		}
	}

	// Event Source Mappings
	for _, eachEventSourceMapping := range info.EventSourceMappings {
		mappingErr := eachEventSourceMapping.export(serviceName,
			functionAttr,
			S3Bucket,
			S3Key,
			template,
			logger)
		if nil != mappingErr {
			return mappingErr
		}
	}

	// CustomResource
	for _, eachCustomResource := range info.customResources {
		resourceErr := eachCustomResource.export(serviceName,
			functionAttr,
			lambdaRuntime,
			S3Bucket,
			S3Key,
			roleNameMap,
			template,
			logger)
		if nil != resourceErr {
			return resourceErr
		}
	}

	// Decorator
	if nil != info.Decorator {
		logger.Debug("Decorator found for Lambda: ", info.lambdaFunctionName())
		// Create an empty template so that we can track whether things
		// are overwritten
		metadataMap := make(map[string]interface{})
		decoratorProxyTemplate := gocf.NewTemplate()
		err := info.Decorator(serviceName,
			info.logicalName(),
			lambdaResource,
			metadataMap,
			S3Bucket,
			S3Key,
			buildID,
			decoratorProxyTemplate,
			context,
			logger)
		if nil != err {
			return err
		}

		// This data is marshalled into a DiscoveryInfo struct s.t. it can be
		// unmarshalled via sparta.Discover.  We're going to just stuff it into
		// it's own same named property
		if len(metadataMap) != 0 {
			safeMetadataInsert(cfResource, info.logicalName(), metadataMap)
		}
		// Append the custom resources
		err = safeMergeTemplates(decoratorProxyTemplate, template, logger)
		if nil != err {
			return fmt.Errorf("Lambda (%s) decorator created conflicting resources", info.lambdaFunctionName())
		}
	}
	return nil
}

//
// END - LambdaAWSInfo
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
//
// BEGIN - Private
//

func validateSpartaPreconditions(lambdaAWSInfos []*LambdaAWSInfo, logger *logrus.Logger) error {

	var errorText []string
	collisionMemo := make(map[string]int)

	incrementCounter := func(keyName string) {
		_, exists := collisionMemo[keyName]
		if !exists {
			collisionMemo[keyName] = 1
		} else {
			collisionMemo[keyName] = collisionMemo[keyName] + 1
		}
	}

	// 1 - check for duplicate golang function references.
	for _, eachLambda := range lambdaAWSInfos {
		incrementCounter(eachLambda.lambdaFunctionName())
		for _, eachCustom := range eachLambda.customResources {
			incrementCounter(eachCustom.userFunctionName)
		}
	}
	// Duplicates?
	for eachLambdaName, eachCount := range collisionMemo {
		if eachCount > 1 {
			logger.WithFields(logrus.Fields{
				"CollisionCount": eachCount,
				"Name":           eachLambdaName,
			}).Error("HandleAWSLambda")
			errorText = append(errorText, fmt.Sprintf("Multiple definitions of lambda: %s", eachLambdaName))
		}
	}
	logger.WithFields(logrus.Fields{
		"CollisionMap": collisionMemo,
	}).Debug("Lambda collision map")

	if len(errorText) != 0 {
		return errors.New(strings.Join(errorText[:], "\n"))
	}
	return nil
}

// Sanitize the provided input by replacing illegal characters with underscores
func sanitizedName(input string) string {
	return reSanitize.ReplaceAllString(input, "_")
}

//
// END - Private
//
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// Public
////////////////////////////////////////////////////////////////////////////////

// CloudFormationResourceName returns a name suitable as a logical
// CloudFormation resource value.  See http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/resources-section-structure.html
// for more information.  The `prefix` value should provide a hint as to the
// resource type (eg, `SNSConfigurator`, `ImageTranscoder`).  Note that the returned
// name is not content-addressable.
func CloudFormationResourceName(prefix string, parts ...string) string {
	return spartaCF.CloudFormationResourceName(prefix, parts...)
}

// LambdaName returns the Go-reflection discovered name for a given
// function
func LambdaName(handlerFunc http.HandlerFunc) string {
	lambdaPtr := runtime.FuncForPC(reflect.ValueOf(handlerFunc).Pointer())
	return lambdaPtr.Name()
}

// NewLambda returns a LambdaAWSInfo value that can be provisioned via CloudFormation. The
// roleNameOrIAMRoleDefinition must either be a `string` or `IAMRoleDefinition`
// type
func NewLambda(roleNameOrIAMRoleDefinition interface{},
	fn LambdaFunction,
	lambdaOptions *LambdaFunctionOptions) *LambdaAWSInfo {

	if nil == lambdaOptions {
		lambdaOptions = defaultLambdaFunctionOptions()
	}
	lambda := &LambdaAWSInfo{
		lambdaFn:            fn,
		Options:             lambdaOptions,
		Permissions:         make([]LambdaPermissionExporter, 0),
		EventSourceMappings: make([]*EventSourceMapping, 0),
	}

	switch v := roleNameOrIAMRoleDefinition.(type) {
	case string:
		lambda.RoleName = roleNameOrIAMRoleDefinition.(string)
	case IAMRoleDefinition:
		definition := roleNameOrIAMRoleDefinition.(IAMRoleDefinition)
		lambda.RoleDefinition = &definition
	default:
		panic(fmt.Sprintf("Unsupported IAM Role type: %s", v))
	}
	// Defaults
	if lambda.Options.MemorySize <= 0 {
		lambda.Options.MemorySize = 128
	}
	if lambda.Options.Timeout <= 0 {
		lambda.Options.Timeout = 3
	}
	return lambda
}

// HandleAWSLambda registers lambdaHandler with the given functionName
// using the default lambdaFunctionOptions
func HandleAWSLambda(functionName string,
	lambdaHandler http.Handler,
	roleNameOrIAMRoleDefinition interface{}) *LambdaAWSInfo {

	lambda := &LambdaAWSInfo{
		userSuppliedFunctionName: functionName,
		httpHandler:              lambdaHandler,
		Options:                  defaultLambdaFunctionOptions(),
		Permissions:              make([]LambdaPermissionExporter, 0),
		EventSourceMappings:      make([]*EventSourceMapping, 0),
	}

	switch v := roleNameOrIAMRoleDefinition.(type) {
	case string:
		lambda.RoleName = roleNameOrIAMRoleDefinition.(string)
	case IAMRoleDefinition:
		definition := roleNameOrIAMRoleDefinition.(IAMRoleDefinition)
		lambda.RoleDefinition = &definition
	default:
		panic(fmt.Sprintf("Unsupported IAM Role type: %s", v))
	}
	return lambda
}

// NewLoggerWithFormatter returns a logger with the given formatter. If formatter
// is nil, a TTY-aware formatter is used
func NewLoggerWithFormatter(level string, formatter logrus.Formatter) (*logrus.Logger, error) {
	logger := logrus.New()
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return nil, err
	}
	logger.Level = logLevel
	if nil != formatter {
		logger.Formatter = formatter
	}
	logger.Out = os.Stdout
	return logger, nil
}

// NewLogger returns a new logrus.Logger instance. It is the caller's responsibility
// to set the formatter if needed.
func NewLogger(level string) (*logrus.Logger, error) {
	return NewLoggerWithFormatter(level, nil)
}
