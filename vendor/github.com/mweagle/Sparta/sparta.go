package sparta

import (
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	spartaIAM "github.com/mweagle/Sparta/aws/iam"
	"math/rand"
	"net/http"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	gocf "github.com/crewjam/go-cloudformation"

	"github.com/Sirupsen/logrus"
)

////////////////////////////////////////////////////////////////////////////////
// Constants
////////////////////////////////////////////////////////////////////////////////

const (
	// SpartaVersion defines the current Sparta release
	SpartaVersion = "0.7.1"
	// NodeJSVersion is the Node JS runtime used for the shim layer
	NodeJSVersion = "nodejs4.3"

	// Custom Resource typename used to create new cloudFormationUserDefinedFunctionCustomResource
	cloudFormationLambda = "Custom::SpartaLambdaCustomResource"
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
var reSanitize = regexp.MustCompile("[:\\.\\-\\s]+")

// RE to ensure CloudFormation compatible resource names
// Issue: https://github.com/mweagle/Sparta/issues/8
// Ref: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/resources-section-structure.html
var reCloudFormationInvalidChars = regexp.MustCompile("[^A-Za-z0-9]+")

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
	Event   json.RawMessage `json:"event"`
	Context LambdaContext   `json:"context"`
}

// LambdaContext defines the AWS Lambda Context object provided by the AWS Lambda runtime.
// See http://docs.aws.amazon.com/lambda/latest/dg/nodejs-prog-model-context.html
// for more information on field values.  Note that the golang version doesn't functions
// defined on the Context object.
type LambdaContext struct {
	AWSRequestID       string `json:"awsRequestId"`
	InvokeID           string `json:"invokeid"`
	LogGroupName       string `json:"logGroupName"`
	LogStreamName      string `json:"logStreamName"`
	FunctionName       string `json:"functionName"`
	MemoryLimitInMB    string `json:"memoryLimitInMB"`
	FunctionVersion    string `json:"functionVersion"`
	InvokedFunctionARN string `json:"invokedFunctionArn"`
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
}

func defaultLambdaFunctionOptions() *LambdaFunctionOptions {
	return &LambdaFunctionOptions{Description: "",
		MemorySize: 128,
		Timeout:    3,
		VpcConfig:  nil,
	}
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
	template *gocf.Template,
	logger *logrus.Logger) error

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
		statements = append(statements, spartaIAM.PolicyStatement{
			Effect:   "Allow",
			Action:   eachPrivilege.Actions,
			Resource: eachPrivilege.resourceExpr(),
		})
	}

	// Add VPC permissions iff needed
	if options != nil && options.VpcConfig != nil {
		for _, eachStatement := range CommonIAMStatements.VPC {
			statements = append(statements, eachStatement)
		}
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
				statements = append(statements, CommonIAMStatements.DynamoDB...)
			case "kinesis":
				statements = append(statements, CommonIAMStatements.Kinesis...)
			default:
				logger.Debug("No additional statements found")
			}
		}
	}

	iamPolicies := gocf.IAMPoliciesList{}
	iamPolicies = append(iamPolicies, gocf.IAMPolicies{
		PolicyDocument: ArbitraryJSONObject{
			"Version":   "2012-10-17",
			"Statement": statements,
		},
		PolicyName: gocf.String(CloudFormationResourceName("LambdaPolicy")),
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
func (resourceInfo *customResourceInfo) jsHandlerName() string {
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
		Handler:     gocf.String(fmt.Sprintf("index.%s", resourceInfo.jsHandlerName())),
		MemorySize:  gocf.Integer(resourceInfo.options.MemorySize),
		Role:        roleNameMap[iamRoleArnName],
		Runtime:     gocf.String(NodeJSVersion),
		Timeout:     gocf.Integer(resourceInfo.options.Timeout),
		VpcConfig:   resourceInfo.options.VpcConfig,
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
	// internal function name, determined by reflection
	lambdaFnName string
	// pointer to lambda function
	lambdaFn LambdaFunction
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
}

// URLPath returns the URL path that can be used as an argument
// to NewLambdaRequest or NewAPIGatewayRequest
func (info *LambdaAWSInfo) URLPath() string {
	return info.lambdaFnName
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

// Returns a JavaScript compatible function name for the golang function name.  This
// value will be used as the URL path component for the HTTP proxying layer.
func (info *LambdaAWSInfo) jsHandlerName() string {
	return sanitizedName(info.lambdaFnName)
}

// Returns the stable logical name for this LambdaAWSInfo value
func (info *LambdaAWSInfo) logicalName() string {
	// Per http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/resources-section-structure.html,
	// we can only use alphanumeric, so we'll take the sanitized name and
	// remove all underscores
	resourceName := strings.Replace(sanitizedName(info.lambdaFnName), "_", "", -1)
	prefix := fmt.Sprintf("%sLambda", resourceName)
	return CloudFormationResourceName(prefix, info.lambdaFnName)
}

// Marshal this object into 1 or more CloudFormation resource definitions that are accumulated
// in the resources map
func (info *LambdaAWSInfo) export(serviceName string,
	S3Bucket string,
	S3Key string,
	roleNameMap map[string]*gocf.StringExpr,
	template *gocf.Template,
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
		iamRoleArnName = info.RoleDefinition.logicalName(serviceName, info.lambdaFnName)
		dependsOn = append(dependsOn, info.RoleDefinition.logicalName(serviceName, info.lambdaFnName))
	}
	lambdaDescription := info.Options.Description
	if "" == lambdaDescription {
		lambdaDescription = fmt.Sprintf("%s: %s", serviceName, info.lambdaFnName)
	}

	// Create the primary resource
	lambdaResource := gocf.LambdaFunction{
		Code: &gocf.LambdaFunctionCode{
			S3Bucket: gocf.String(S3Bucket),
			S3Key:    gocf.String(S3Key),
		},
		Description: gocf.String(lambdaDescription),
		Handler:     gocf.String(fmt.Sprintf("index.%s", info.jsHandlerName())),
		MemorySize:  gocf.Integer(info.Options.MemorySize),
		Role:        roleNameMap[iamRoleArnName],
		Runtime:     gocf.String(NodeJSVersion),
		Timeout:     gocf.Integer(info.Options.Timeout),
		VpcConfig:   info.Options.VpcConfig,
	}

	cfResource := template.AddResource(info.logicalName(), lambdaResource)
	cfResource.DependsOn = append(cfResource.DependsOn, dependsOn...)
	safeMetadataInsert(cfResource, "golangFunc", info.lambdaFnName)

	// Create the lambda Ref in case we need a permission or event mapping
	functionAttr := gocf.GetAtt(info.logicalName(), "Arn")

	// Permissions
	for _, eachPermission := range info.Permissions {
		_, err := eachPermission.export(serviceName,
			info.lambdaFnName,
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
		logger.Debug("Decorator found for Lambda: ", info.lambdaFnName)
		// Create an empty template so that we can track whether things
		// are overwritten
		metadataMap := make(map[string]interface{}, 0)
		decoratorProxyTemplate := gocf.NewTemplate()
		err := info.Decorator(serviceName,
			info.logicalName(),
			lambdaResource,
			metadataMap,
			S3Bucket,
			S3Key,
			decoratorProxyTemplate,
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
			return fmt.Errorf("Lambda (%s) decorator created conflicting resources", info.lambdaFnName)
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
	collisionMemo := make(map[string]int, 0)

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
		incrementCounter(eachLambda.lambdaFnName)
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
			}).Error("Detected logically equivalent function associated with multiple structs")
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
	hash := sha1.New()
	hash.Write([]byte(prefix))
	if len(parts) <= 0 {
		randValue := rand.Int63()
		hash.Write([]byte(strconv.FormatInt(randValue, 10)))
	} else {
		for _, eachPart := range parts {
			hash.Write([]byte(eachPart))
		}
	}
	resourceName := fmt.Sprintf("%s%s", prefix, hex.EncodeToString(hash.Sum(nil)))

	// Ensure that any non alphanumeric characters are replaced with ""
	return reCloudFormationInvalidChars.ReplaceAllString(resourceName, "x")
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
	lambdaPtr := runtime.FuncForPC(reflect.ValueOf(fn).Pointer())
	lambda := &LambdaAWSInfo{
		lambdaFnName:        lambdaPtr.Name(),
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

// NewLogger returns a new logrus.Logger instance. It is the caller's responsibility
// to set the formatter if needed.
func NewLogger(level string) (*logrus.Logger, error) {
	logger := logrus.New()
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return nil, err
	}
	logger.Level = logLevel
	return logger, nil
}
