package sparta

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mweagle/cloudformationresources"

	gocf "github.com/crewjam/go-cloudformation"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/service/s3"
)

/*
Notes to future self...

TODO - Simplify this as part of https://trello.com/c/aOULlJcz/14-port-nodejs-customresources-to-go

Adding a new permission type?
  1. Add the principal name value to sparta.go constants
  2. Define the new struct and satisfy LambdaPermissionExporter
  3. Update provision_utils.go's `PushSourceConfigurationActions` map with the new principal's permissions
  4. Update `PROXIED_MODULES` in resources/index.js to include the first principal component name( eg, 'events')
  5. Update `customResourceScripts` in provision.go to ensure the embedded JS file is included in the deployed archive.
  6. Implement the custom type defined in 2
  7. Implement the service configuration logic referred to in 4.
*/

////////////////////////////////////////////////////////////////////////////////
// Types to handle permissions & push source configuration
type descriptionNode struct {
	Name     string
	Relation string
	Color    string
}

// LambdaPermissionExporter defines an interface for polymorphic collection of
// Permission entries that support specialization for additional resource generation.
type LambdaPermissionExporter interface {
	// Export the permission object to a set of CloudFormation resources
	// in the provided resources param.  The targetLambdaFuncRef
	// interface represents the Fn::GetAtt "Arn" JSON value
	// of the parent Lambda target
	export(serviceName string,
		lambdaFunctionDisplayName string,
		lambdaLogicalCFResourceName string,
		template *gocf.Template,
		S3Bucket string,
		S3Key string,
		logger *logrus.Logger) (string, error)
	// Return a `describe` compatible output for the given permission.  Return
	// value is a list of tuples for node, edgeLabel
	descriptionInfo() ([]descriptionNode, error)
}

////////////////////////////////////////////////////////////////////////////////
// START - BasePermission
//

// BasePermission (http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-lambda-permission.html)
// type for common AWS Lambda permission data.
type BasePermission struct {
	// The AWS account ID (without hyphens) of the source owner
	SourceAccount string `json:"SourceAccount,omitempty"`
	// The ARN of a resource that is invoking your function.
	SourceArn interface{} `json:"SourceArn,omitempty"`
}

func (perm *BasePermission) sourceArnExpr(joinParts ...gocf.Stringable) *gocf.StringExpr {
	var parts []gocf.Stringable
	if nil != joinParts {
		parts = append(parts, joinParts...)
	}
	switch perm.SourceArn.(type) {
	case string:
		// Don't be smart if the Arn value is a user supplied literal
		parts = []gocf.Stringable{gocf.String(perm.SourceArn.(string))}
	case *gocf.StringExpr:
		parts = append(parts, perm.SourceArn.(*gocf.StringExpr))
	case gocf.RefFunc:
		parts = append(parts, perm.SourceArn.(gocf.RefFunc).String())
	default:
		panic(fmt.Sprintf("Unsupported SourceArn value type: %+v", perm.SourceArn))
	}
	return gocf.Join("", parts...)
}

func describeInfoArn(arnExpression interface{}) string {
	switch arnExpression.(type) {
	case string:
		return arnExpression.(string)
	case *gocf.StringExpr,
		gocf.RefFunc:
		data, _ := json.Marshal(arnExpression)
		return string(data)
	default:
		panic(fmt.Sprintf("Unsupported SourceArn value type: %+v", arnExpression))
	}
}

func (perm BasePermission) export(principal *gocf.StringExpr,
	arnPrefixParts []gocf.Stringable,
	lambdaFunctionDisplayName string,
	lambdaLogicalCFResourceName string,
	template *gocf.Template,
	S3Bucket string,
	S3Key string,
	logger *logrus.Logger) (string, error) {

	lambdaPermission := gocf.LambdaPermission{
		Action:       gocf.String("lambda:InvokeFunction"),
		FunctionName: gocf.GetAtt(lambdaLogicalCFResourceName, "Arn"),
		Principal:    principal,
	}
	// If the Arn isn't the wildcard value, then include it.
	if nil != perm.SourceArn {
		switch perm.SourceArn.(type) {
		case string:
			// Don't be smart if the Arn value is a user supplied literal
			if "*" != perm.SourceArn.(string) {
				lambdaPermission.SourceArn = gocf.String(perm.SourceArn.(string))
			}
		default:
			lambdaPermission.SourceArn = perm.sourceArnExpr(arnPrefixParts...)
		}
	}

	if "" != perm.SourceAccount {
		lambdaPermission.SourceAccount = gocf.String(perm.SourceAccount)
	}

	arnLiteral, arnLiteralErr := json.Marshal(lambdaPermission.SourceArn)
	if nil != arnLiteralErr {
		return "", arnLiteralErr
	}
	resourceName := CloudFormationResourceName("LambdaPerm%s",
		principal.Literal,
		string(arnLiteral),
		lambdaLogicalCFResourceName)
	template.AddResource(resourceName, lambdaPermission)
	return resourceName, nil
}

//
// END - BasePermission
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// START - LambdaPermission
//
var lambdaSourceArnParts = []gocf.Stringable{
	gocf.String("arn:aws:lambda:"),
	gocf.Ref("AWS::Region"),
	gocf.String(":function:"),
}

// LambdaPermission type that creates a Lambda::Permission entry
// in the generated template, but does NOT automatically register the lambda
// with the BasePermission.SourceArn.  Typically used to register lambdas with
// externally managed event producers
type LambdaPermission struct {
	BasePermission
	// The entity for which you are granting permission to invoke the Lambda function
	Principal string
}

func (perm LambdaPermission) export(serviceName string,
	lambdaFunctionDisplayName string,
	lambdaLogicalCFResourceName string,
	template *gocf.Template,
	S3Bucket string,
	S3Key string,
	logger *logrus.Logger) (string, error) {

	return perm.BasePermission.export(gocf.String(perm.Principal),
		lambdaSourceArnParts,
		lambdaFunctionDisplayName,
		lambdaLogicalCFResourceName,
		template,
		S3Bucket,
		S3Key,
		logger)
}

func (perm LambdaPermission) descriptionInfo() ([]descriptionNode, error) {
	nodes := []descriptionNode{
		{
			Name:     "Source",
			Relation: describeInfoArn(perm.SourceArn),
		},
	}
	return nodes, nil
}

//
// END - LambdaPermission
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// START - S3Permission
//
var s3SourceArnParts = []gocf.Stringable{
	gocf.String("arn:aws:s3:::"),
}

// S3Permission struct implies that the S3 BasePermission.SourceArn should be
// updated (via PutBucketNotificationConfiguration) to automatically push
// events to the owning Lambda.
// See http://docs.aws.amazon.com/lambda/latest/dg/intro-core-components.html#intro-core-components-event-sources
// for more information.
type S3Permission struct {
	BasePermission
	// S3 events to register for (eg: `[]string{s3:GetObjectObjectCreated:*", "s3:ObjectRemoved:*"}`).
	Events []string `json:"Events,omitempty"`
	// S3.NotificationConfigurationFilter
	// to scope event forwarding.  See
	// 		http://docs.aws.amazon.com/AmazonS3/latest/dev/NotificationHowTo.html
	// for more information.
	Filter s3.NotificationConfigurationFilter `json:"Filter,omitempty"`
}

func (perm S3Permission) export(serviceName string,
	lambdaFunctionDisplayName string,
	lambdaLogicalCFResourceName string,
	template *gocf.Template,
	S3Bucket string,
	S3Key string,
	logger *logrus.Logger) (string, error) {

	targetLambdaResourceName, err := perm.BasePermission.export(gocf.String("s3.amazonaws.com"),
		s3SourceArnParts,
		lambdaFunctionDisplayName,
		lambdaLogicalCFResourceName,
		template,
		S3Bucket,
		S3Key,
		logger)

	if nil != err {
		return "", err
	}

	// Make sure the custom lambda that manages s3 notifications is provisioned.
	sourceArnExpression := perm.BasePermission.sourceArnExpr(s3SourceArnParts...)
	configuratorResName, err := ensureCustomResourceHandler(serviceName,
		cloudformationresources.S3LambdaEventSource,
		sourceArnExpression,
		[]string{},
		template,
		S3Bucket,
		S3Key,
		logger)

	if nil != err {
		return "", err
	}

	// Add a custom resource invocation for this configuration
	//////////////////////////////////////////////////////////////////////////////
	newResource, newResourceError := newCloudFormationResource(cloudformationresources.S3LambdaEventSource, logger)
	if nil != newResourceError {
		return "", newResourceError
	}
	customResource := newResource.(*cloudformationresources.S3LambdaEventSourceResource)
	customResource.ServiceToken = gocf.GetAtt(configuratorResName, "Arn")
	customResource.BucketArn = sourceArnExpression
	customResource.LambdaTargetArn = gocf.GetAtt(lambdaLogicalCFResourceName, "Arn")
	customResource.Events = perm.Events
	if nil != perm.Filter.Key {
		customResource.Filter = &perm.Filter
	}

	// Name?
	resourceInvokerName := CloudFormationResourceName("ConfigS3",
		lambdaLogicalCFResourceName,
		perm.BasePermission.SourceAccount)

	// Add it
	cfResource := template.AddResource(resourceInvokerName, customResource)
	cfResource.DependsOn = append(cfResource.DependsOn,
		targetLambdaResourceName,
		configuratorResName)
	return "", nil
}

func (perm S3Permission) descriptionInfo() ([]descriptionNode, error) {
	s3Events := ""
	for _, eachEvent := range perm.Events {
		s3Events = fmt.Sprintf("%s\n%s", eachEvent, s3Events)
	}

	nodes := []descriptionNode{
		{
			Name:     describeInfoArn(perm.SourceArn),
			Relation: s3Events,
		},
	}
	return nodes, nil
}

// END - S3Permission
///////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// SNSPermission - START
var snsSourceArnParts = []gocf.Stringable{}

// SNSPermission struct implies that the BasePermisison.SourceArn should be
// configured for subscriptions as part of this stacks provisioning.
// See http://docs.aws.amazon.com/lambda/latest/dg/intro-core-components.html#intro-core-components-event-sources
// for more information.
type SNSPermission struct {
	BasePermission
}

func (perm SNSPermission) export(serviceName string,
	lambdaFunctionDisplayName string,
	lambdaLogicalCFResourceName string,
	template *gocf.Template,
	S3Bucket string,
	S3Key string,
	logger *logrus.Logger) (string, error) {
	sourceArnExpression := perm.BasePermission.sourceArnExpr(snsSourceArnParts...)

	targetLambdaResourceName, err := perm.BasePermission.export(gocf.String(SNSPrincipal),
		snsSourceArnParts,
		lambdaFunctionDisplayName,
		lambdaLogicalCFResourceName,
		template,
		S3Bucket,
		S3Key,
		logger)
	if nil != err {
		return "", err
	}

	// Make sure the custom lambda that manages s3 notifications is provisioned.
	configuratorResName, err := ensureCustomResourceHandler(serviceName,
		cloudformationresources.SNSLambdaEventSource,
		sourceArnExpression,
		[]string{},
		template,
		S3Bucket,
		S3Key,
		logger)

	if nil != err {
		return "", err
	}

	// Add a custom resource invocation for this configuration
	//////////////////////////////////////////////////////////////////////////////
	newResource, newResourceError := newCloudFormationResource(cloudformationresources.SNSLambdaEventSource, logger)
	if nil != newResourceError {
		return "", newResourceError
	}
	customResource := newResource.(*cloudformationresources.SNSLambdaEventSourceResource)
	customResource.ServiceToken = gocf.GetAtt(configuratorResName, "Arn")
	customResource.LambdaTargetArn = gocf.GetAtt(lambdaLogicalCFResourceName, "Arn")
	customResource.SNSTopicArn = sourceArnExpression

	// Name?
	resourceInvokerName := CloudFormationResourceName("ConfigSNS",
		lambdaLogicalCFResourceName,
		perm.BasePermission.SourceAccount)

	// Add it
	cfResource := template.AddResource(resourceInvokerName, customResource)
	cfResource.DependsOn = append(cfResource.DependsOn,
		targetLambdaResourceName,
		configuratorResName)
	return "", nil
}

func (perm SNSPermission) descriptionInfo() ([]descriptionNode, error) {
	nodes := []descriptionNode{
		{
			Name:     describeInfoArn(perm.SourceArn),
			Relation: "",
		},
	}
	return nodes, nil
}

//
// END - SNSPermission
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// MessageBodyStorageOptions - START

// MessageBodyStorageOptions define additional options for storing SES
// message body content.  By default, all rules associated with the owning
// SESPermission object will store message bodies if the MessageBodyStorage
// field is non-nil.  Message bodies are by default prefixed with
// `ServiceName/RuleName/`, which can be overridden by specifying a non-empty
// ObjectKeyPrefix value.  A rule can opt-out of message body storage
// with the DisableStorage field.  See
// http://docs.aws.amazon.com/ses/latest/DeveloperGuide/receiving-email-action-s3.html
// for additional field documentation.
// The message body is saved as MIME (https://tools.ietf.org/html/rfc2045)
type MessageBodyStorageOptions struct {
	ObjectKeyPrefix string
	KmsKeyArn       string
	TopicArn        string
	DisableStorage  bool
}

//
// END - MessageBodyStorageOptions
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// MessageBodyStorage - START

// MessageBodyStorage represents either a new S3 bucket or an existing S3 bucket
// to which SES message bodies should be stored.
// NOTE: New MessageBodyStorage create S3 buckets which will be orphaned after your
// service is deleted.
type MessageBodyStorage struct {
	logicalBucketName                  string
	bucketNameExpr                     *gocf.StringExpr
	cloudFormationS3BucketResourceName string
}

// BucketArn returns an Arn value that can be used as an
// lambdaFn.RoleDefinition.Privileges `Resource` value.
func (storage *MessageBodyStorage) BucketArn() *gocf.StringExpr {
	return gocf.Join("",
		gocf.String("arn:aws:s3:::"),
		storage.bucketNameExpr)
}

// BucketArnAllKeys returns an Arn value that can be used
// lambdaFn.RoleDefinition.Privileges `Resource` value.  It includes
// the trailing `/*` wildcard to support item acccess
func (storage *MessageBodyStorage) BucketArnAllKeys() *gocf.StringExpr {
	return gocf.Join("",
		gocf.String("arn:aws:s3:::"),
		storage.bucketNameExpr,
		gocf.String("/*"))
}

func (storage *MessageBodyStorage) export(serviceName string,
	lambdaFunctionDisplayName string,
	lambdaLogicalCFResourceName string,
	template *gocf.Template,
	S3Bucket string,
	S3Key string,
	logger *logrus.Logger) (string, error) {

	if "" != storage.cloudFormationS3BucketResourceName {
		s3Bucket := &gocf.S3Bucket{
			Tags: []gocf.ResourceTag{
				{
					Key:   gocf.String("sparta:logicalBucketName"),
					Value: gocf.String(storage.logicalBucketName),
				},
			},
		}
		cfResource := template.AddResource(storage.cloudFormationS3BucketResourceName, s3Bucket)
		cfResource.DeletionPolicy = "Retain"

		lambdaResource, _ := template.Resources[lambdaLogicalCFResourceName]
		if nil != lambdaResource {
			safeAppendDependency(lambdaResource, storage.cloudFormationS3BucketResourceName)
		}

		logger.WithFields(logrus.Fields{
			"LogicalResourceName": storage.cloudFormationS3BucketResourceName,
		}).Info("Service will orphan S3 Bucket on deletion")

		// Save the output
		template.Outputs[storage.cloudFormationS3BucketResourceName] = &gocf.Output{
			Description: "SES Message Body Bucket",
			Value:       gocf.Ref(storage.cloudFormationS3BucketResourceName),
		}
	}
	// Add the S3 Access policy
	s3BodyStoragePolicy := &gocf.S3BucketPolicy{
		Bucket: storage.bucketNameExpr,
		PolicyDocument: ArbitraryJSONObject{
			"Version": "2012-10-17",
			"Statement": []ArbitraryJSONObject{
				{
					"Sid":    "PermitSESServiceToSaveEmailBody",
					"Effect": "Allow",
					"Principal": ArbitraryJSONObject{
						"Service": "ses.amazonaws.com",
					},
					"Action": []string{"s3:PutObjectAcl", "s3:PutObject"},
					"Resource": gocf.Join("",
						gocf.String("arn:aws:s3:::"),
						storage.bucketNameExpr,
						gocf.String("/*")),
					"Condition": ArbitraryJSONObject{
						"StringEquals": ArbitraryJSONObject{
							"aws:Referer": gocf.Ref("AWS::AccountId"),
						},
					},
				},
			},
		},
	}

	s3BucketPolicyResourceName := CloudFormationResourceName("SESMessageBodyBucketPolicy",
		fmt.Sprintf("%#v", storage.bucketNameExpr))
	template.AddResource(s3BucketPolicyResourceName, s3BodyStoragePolicy)

	// Return the name of the bucket policy s.t. the configurator resource
	// is properly sequenced.  The configurator will fail iff the Bucket Policies aren't
	// applied b/c the SES Rule Actions check PutObject access to S3 buckets
	return s3BucketPolicyResourceName, nil
}

// Return a function that

//
// END - MessageBodyStorage
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// ReceiptRule - START

// ReceiptRule represents an SES ReceiptRule
// (http://docs.aws.amazon.com/ses/latest/DeveloperGuide/receiving-email-receipt-rules.html)
// value.  To store message bodies, provide a non-nil MessageBodyStorage value
// to the owning SESPermission object
type ReceiptRule struct {
	Name               string
	Disabled           bool
	Recipients         []string
	ScanDisabled       bool
	TLSPolicy          string
	TopicArn           string
	InvocationType     string
	BodyStorageOptions MessageBodyStorageOptions
}

func (rule *ReceiptRule) toResourceRule(serviceName string,
	functionArnRef interface{},
	messageBodyStorage *MessageBodyStorage) *cloudformationresources.SESLambdaEventSourceResourceRule {

	resourceRule := &cloudformationresources.SESLambdaEventSourceResourceRule{
		Name:        gocf.String(rule.Name),
		ScanEnabled: gocf.Bool(!rule.ScanDisabled),
		Enabled:     gocf.Bool(!rule.Disabled),
		Actions:     make([]*cloudformationresources.SESLambdaEventSourceResourceAction, 0),
		Recipients:  make([]*gocf.StringExpr, 0),
	}
	for _, eachRecipient := range rule.Recipients {
		resourceRule.Recipients = append(resourceRule.Recipients, gocf.String(eachRecipient))
	}
	if "" != rule.TLSPolicy {
		resourceRule.TLSPolicy = gocf.String(rule.TLSPolicy)
	}

	// If there is a MessageBodyStorage reference, push that S3Action
	// to the head of the Actions list
	if nil != messageBodyStorage && !rule.BodyStorageOptions.DisableStorage {
		s3Action := &cloudformationresources.SESLambdaEventSourceResourceAction{
			ActionType: gocf.String("S3Action"),
			ActionProperties: map[string]interface{}{
				"BucketName": messageBodyStorage.bucketNameExpr,
			},
		}
		if "" != rule.BodyStorageOptions.ObjectKeyPrefix {
			s3Action.ActionProperties["ObjectKeyPrefix"] = rule.BodyStorageOptions.ObjectKeyPrefix
		}
		if "" != rule.BodyStorageOptions.KmsKeyArn {
			s3Action.ActionProperties["KmsKeyArn"] = rule.BodyStorageOptions.KmsKeyArn
		}
		if "" != rule.BodyStorageOptions.TopicArn {
			s3Action.ActionProperties["TopicArn"] = rule.BodyStorageOptions.TopicArn
		}
		resourceRule.Actions = append(resourceRule.Actions, s3Action)
	}
	// There's always a lambda action
	lambdaAction := &cloudformationresources.SESLambdaEventSourceResourceAction{
		ActionType: gocf.String("LambdaAction"),
		ActionProperties: map[string]interface{}{
			"FunctionArn": functionArnRef,
		},
	}
	lambdaAction.ActionProperties["InvocationType"] = rule.InvocationType
	if "" == rule.InvocationType {
		lambdaAction.ActionProperties["InvocationType"] = "Event"
	}
	if "" != rule.TopicArn {
		lambdaAction.ActionProperties["TopicArn"] = rule.TopicArn
	}
	resourceRule.Actions = append(resourceRule.Actions, lambdaAction)
	return resourceRule
}

//
// END - ReceiptRule
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// SESPermission - START

// SES doesn't use ARNs to scope access
var sesSourcePartArn = []gocf.Stringable{wildcardArn}

// SESPermission struct implies that the SES verified domain should be
// updated (via createReceiptRule) to automatically request or push events
// to the parent lambda
// See http://docs.aws.amazon.com/lambda/latest/dg/intro-core-components.html#intro-core-components-event-sources
// for more information.  See http://docs.aws.amazon.com/ses/latest/DeveloperGuide/receiving-email-concepts.html
// for setting up email receiving.
type SESPermission struct {
	BasePermission
	InvocationType     string /* RequestResponse, Event */
	ReceiptRules       []ReceiptRule
	MessageBodyStorage *MessageBodyStorage
}

// NewMessageBodyStorageResource provisions a new S3 bucket to store message body
// content.
func (perm *SESPermission) NewMessageBodyStorageResource(bucketLogicalName string) (*MessageBodyStorage, error) {
	if len(bucketLogicalName) <= 0 {
		return nil, errors.New("NewMessageBodyStorageResource requires a unique, non-empty `bucketLogicalName` parameter ")
	}
	store := &MessageBodyStorage{
		logicalBucketName: bucketLogicalName,
	}
	store.cloudFormationS3BucketResourceName = CloudFormationResourceName("SESMessageStoreBucket", bucketLogicalName)
	store.bucketNameExpr = gocf.Ref(store.cloudFormationS3BucketResourceName).String()
	return store, nil
}

// NewMessageBodyStorageReference uses a pre-existing S3 bucket for MessageBody storage.
// Sparta assumes that prexistingBucketName exists and will add an S3::BucketPolicy
// to enable SES PutObject access.
func (perm *SESPermission) NewMessageBodyStorageReference(prexistingBucketName string) (*MessageBodyStorage, error) {
	store := &MessageBodyStorage{}
	store.bucketNameExpr = gocf.String(prexistingBucketName)
	return store, nil
}

func (perm SESPermission) export(serviceName string,
	lambdaFunctionDisplayName string,
	lambdaLogicalCFResourceName string,
	template *gocf.Template,
	S3Bucket string,
	S3Key string,
	logger *logrus.Logger) (string, error) {

	sourceArnExpression := perm.BasePermission.sourceArnExpr(snsSourceArnParts...)

	targetLambdaResourceName, err := perm.BasePermission.export(gocf.String(SESPrincipal),
		sesSourcePartArn,
		lambdaFunctionDisplayName,
		lambdaLogicalCFResourceName,
		template,
		S3Bucket,
		S3Key,
		logger)
	if nil != err {
		return "", err
	}

	// MessageBody storage?
	var dependsOn []string
	if nil != perm.MessageBodyStorage {
		s3Policy, s3PolicyErr := perm.MessageBodyStorage.export(serviceName,
			lambdaFunctionDisplayName,
			lambdaLogicalCFResourceName,
			template,
			S3Bucket,
			S3Key,
			logger)
		if nil != s3PolicyErr {
			return "", s3PolicyErr
		}
		if "" != s3Policy {
			dependsOn = append(dependsOn, s3Policy)
		}
	}

	// Make sure the custom lambda that manages SNS notifications is provisioned.
	configuratorResName, err := ensureCustomResourceHandler(serviceName,
		cloudformationresources.SESLambdaEventSource,
		sourceArnExpression,
		dependsOn,
		template,
		S3Bucket,
		S3Key,
		logger)

	if nil != err {
		return "", err
	}

	// Add a custom resource invocation for this configuration
	//////////////////////////////////////////////////////////////////////////////
	newResource, newResourceError := newCloudFormationResource(cloudformationresources.SESLambdaEventSource, logger)
	if nil != newResourceError {
		return "", newResourceError
	}
	customResource := newResource.(*cloudformationresources.SESLambdaEventSourceResource)
	customResource.ServiceToken = gocf.GetAtt(configuratorResName, "Arn")
	// The shared ruleset name used by all Sparta applications
	customResource.RuleSetName = gocf.String("SpartaRuleSet")

	///////////////////
	// Build up the Rules
	// If there aren't any rules, make one that forwards everything...
	var sesRules []*cloudformationresources.SESLambdaEventSourceResourceRule
	if nil == perm.ReceiptRules {
		sesRules = append(sesRules,
			&cloudformationresources.SESLambdaEventSourceResourceRule{
				Name:        gocf.String("Default"),
				Actions:     make([]*cloudformationresources.SESLambdaEventSourceResourceAction, 0),
				ScanEnabled: gocf.Bool(false),
				Enabled:     gocf.Bool(true),
				Recipients:  []*gocf.StringExpr{},
				TLSPolicy:   gocf.String("Optional"),
			})
	}
	// Append all the user defined ones
	for _, eachReceiptRule := range perm.ReceiptRules {
		sesRules = append(sesRules, eachReceiptRule.toResourceRule(
			serviceName,
			gocf.GetAtt(lambdaLogicalCFResourceName, "Arn"),
			perm.MessageBodyStorage))
	}
	customResource.Rules = sesRules
	// Name?
	resourceInvokerName := CloudFormationResourceName("ConfigSNS",
		lambdaLogicalCFResourceName,
		perm.BasePermission.SourceAccount)

	// Add it
	cfResource := template.AddResource(resourceInvokerName, customResource)
	cfResource.DependsOn = append(cfResource.DependsOn,
		targetLambdaResourceName,
		configuratorResName)
	return "", nil
}

func (perm SESPermission) descriptionInfo() ([]descriptionNode, error) {
	nodes := []descriptionNode{
		{
			Name:     "SimpleEmailService",
			Relation: "All verified domain(s) email",
		},
	}
	return nodes, nil
}

//
// END - SESPermission
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// START - CloudWatchEventsRuleTarget
//

// CloudWatchEventsRuleTarget specifies additional input and JSON selection
// paths to apply prior to forwarding the event to a lambda function
type CloudWatchEventsRuleTarget struct {
	Input     string
	InputPath string
}

//
// END - CloudWatchEventsRuleTarget
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// START - CloudWatchEventsRule
//

// CloudWatchEventsRule defines parameters for invoking a lambda function
// in response to specific CloudWatchEvents or cron triggers
type CloudWatchEventsRule struct {
	Description string
	// ArbitraryJSONObject filter for events as documented at
	// http://docs.aws.amazon.com/AmazonCloudWatch/latest/DeveloperGuide/CloudWatchEventsandEventPatterns.html
	// Rules matches should use the JSON representation (NOT the string form).  Sparta will serialize
	// the map[string]interface{} to a string form during CloudFormation Template
	// marshalling.
	EventPattern map[string]interface{} `json:"EventPattern,omitempty"`
	// Schedule pattern per http://docs.aws.amazon.com/AmazonCloudWatch/latest/DeveloperGuide/ScheduledEvents.html
	ScheduleExpression string
	RuleTarget         *CloudWatchEventsRuleTarget `json:"RuleTarget,omitempty"`
}

// MarshalJSON customizes the JSON representation used when serializing to the
// CloudFormation template representation.
func (rule CloudWatchEventsRule) MarshalJSON() ([]byte, error) {
	ruleJSON := map[string]interface{}{}

	if "" != rule.Description {
		ruleJSON["Description"] = rule.Description
	}
	if nil != rule.EventPattern {
		eventPatternString, err := json.Marshal(rule.EventPattern)
		if nil != err {
			return nil, err
		}
		ruleJSON["EventPattern"] = string(eventPatternString)
	}
	if "" != rule.ScheduleExpression {
		ruleJSON["ScheduleExpression"] = rule.ScheduleExpression
	}
	if nil != rule.RuleTarget {
		ruleJSON["RuleTarget"] = rule.RuleTarget
	}
	return json.Marshal(ruleJSON)
}

//
// END - CloudWatchEventsRule
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// START - CloudWatchEventsPermission
//
var cloudformationEventsSourceArnParts = []gocf.Stringable{}

// CloudWatchEventsPermission struct implies that the CloudWatchEvent sources
// should be configured as part of provisioning.  The BasePermission.SourceArn
// isn't considered for this configuration. Each CloudWatchEventsRule struct
// in the Rules map is used to register for push based event notifications via
// `putRule` and `deleteRule`.
// See http://docs.aws.amazon.com/lambda/latest/dg/intro-core-components.html#intro-core-components-event-sources
// for more information.
type CloudWatchEventsPermission struct {
	BasePermission
	// Map of rule names to events that trigger the lambda function
	Rules map[string]CloudWatchEventsRule
}

func (perm CloudWatchEventsPermission) export(serviceName string,
	lambdaFunctionDisplayName string,
	lambdaLogicalCFResourceName string,
	template *gocf.Template,
	S3Bucket string,
	S3Key string,
	logger *logrus.Logger) (string, error) {

	// There needs to be at least one rule to apply
	if len(perm.Rules) <= 0 {
		return "", fmt.Errorf("CloudWatchEventsPermission for function %s does not specify any expressions", lambdaFunctionDisplayName)
	}

	// Tell the user we're ignoring any Arns provided, since it doesn't make sense for this.
	if nil != perm.BasePermission.SourceArn &&
		perm.BasePermission.sourceArnExpr(cloudformationEventsSourceArnParts...).String() != wildcardArn.String() {
		logger.WithFields(logrus.Fields{
			"Arn": perm.BasePermission.sourceArnExpr(cloudformationEventsSourceArnParts...),
		}).Warn("CloudWatchEvents do not support literal ARN values")
	}

	arnPermissionForRuleName := func(ruleName string) *gocf.StringExpr {
		return gocf.Join("",
			gocf.String("arn:aws:events:"),
			gocf.Ref("AWS::Region"),
			gocf.String(":"),
			gocf.Ref("AWS::AccountId"),
			gocf.String(":rule/"),
			gocf.String(ruleName))
	}

	// Add the permission to invoke the lambda function
	uniqueRuleNameMap := make(map[string]int, 0)
	for eachRuleName, eachRuleDefinition := range perm.Rules {

		// We need a stable unique name s.t. the permission is properly configured...
		uniqueRuleName := fmt.Sprintf("%s-%s-%s", serviceName, lambdaFunctionDisplayName, eachRuleName)
		uniqueRuleNameMap[uniqueRuleName]++

		// Add the permission
		basePerm := BasePermission{
			SourceArn: arnPermissionForRuleName(uniqueRuleName),
		}
		_, exportErr := basePerm.export(gocf.String(CloudWatchEventsPrincipal),
			cloudformationEventsSourceArnParts,
			lambdaFunctionDisplayName,
			lambdaLogicalCFResourceName,
			template,
			S3Bucket,
			S3Key,
			logger)

		if nil != exportErr {
			return "", exportErr
		}

		cwEventsRuleTargetList := gocf.CloudWatchEventsRuleTargetList{}
		cwEventsRuleTargetList = append(cwEventsRuleTargetList,
			gocf.CloudWatchEventsRuleTarget{
				Arn: gocf.GetAtt(lambdaLogicalCFResourceName, "Arn"),
				Id:  gocf.String(uniqueRuleName),
			},
		)

		// Add the rule
		eventsRule := &gocf.EventsRule{
			Name:        gocf.String(uniqueRuleName),
			Description: gocf.String(eachRuleDefinition.Description),
			Targets:     &cwEventsRuleTargetList,
		}
		if nil != eachRuleDefinition.EventPattern && "" != eachRuleDefinition.ScheduleExpression {
			return "", fmt.Errorf("CloudWatchEvents rule %s specifies both EventPattern and ScheduleExpression", eachRuleName)
		}
		if nil != eachRuleDefinition.EventPattern {
			eventsRule.EventPattern = eachRuleDefinition.EventPattern
		} else if "" != eachRuleDefinition.ScheduleExpression {
			eventsRule.ScheduleExpression = eachRuleDefinition.ScheduleExpression
		}
		cloudWatchLogsEventResName := CloudFormationResourceName(fmt.Sprintf("%s-CloudWatchEventsRule", eachRuleName),
			lambdaLogicalCFResourceName,
			lambdaFunctionDisplayName)
		template.AddResource(cloudWatchLogsEventResName, eventsRule)
	}
	// Validate it
	for _, eachCount := range uniqueRuleNameMap {
		if eachCount != 1 {
			return "", fmt.Errorf("Integrity violation for CloudWatchEvent Rulenames: %#v", uniqueRuleNameMap)
		}
	}
	return "", nil
}

func (perm CloudWatchEventsPermission) descriptionInfo() ([]descriptionNode, error) {
	var ruleTriggers = " "
	for eachName, eachRule := range perm.Rules {
		filter := eachRule.ScheduleExpression
		if "" == filter && nil != eachRule.EventPattern {
			filter = fmt.Sprintf("%v", eachRule.EventPattern["source"])
		}
		ruleTriggers = fmt.Sprintf("%s-(%s)\n%s", eachName, filter, ruleTriggers)
	}
	nodes := []descriptionNode{
		{
			Name:     "CloudWatch Events",
			Relation: ruleTriggers,
		},
	}
	return nodes, nil
}

//
// END - CloudWatchEventsPermission
///////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// START - CloudWatchLogsPermission
//

// CloudWatchLogsSubscriptionFilter represents the CloudWatch Log filter
// information
type CloudWatchLogsSubscriptionFilter struct {
	FilterPattern string
	LogGroupName  string
}

var cloudformationLogsSourceArnParts = []gocf.Stringable{
	gocf.String("arn:aws:logs:"),
}

// CloudWatchLogsPermission struct implies that the corresponding
// CloudWatchLogsSubscriptionFilter definitions should be configured during
// stack provisioning.  The BasePermission.SourceArn isn't considered for
// this configuration operation.  Configuration of the remote push source
// is done via `putSubscriptionFilter` and `deleteSubscriptionFilter`.
// See http://docs.aws.amazon.com/lambda/latest/dg/intro-core-components.html#intro-core-components-event-sources
// for more information.
type CloudWatchLogsPermission struct {
	BasePermission
	// Map of filter names to the CloudWatchLogsSubscriptionFilter settings
	Filters map[string]CloudWatchLogsSubscriptionFilter
}

func (perm CloudWatchLogsPermission) export(serviceName string,
	lambdaFunctionDisplayName string,
	lambdaLogicalCFResourceName string,
	template *gocf.Template,
	S3Bucket string,
	S3Key string,
	logger *logrus.Logger) (string, error) {

	// If there aren't any expressions to register with?
	if len(perm.Filters) <= 0 {
		return "", fmt.Errorf("CloudWatchLogsPermission for function %s does not specify any filters", lambdaFunctionDisplayName)
	}

	// The principal is region specific, so build that up...
	regionalPrincipal := gocf.Join(".",
		gocf.String("logs"),
		gocf.Ref("AWS::Region"),
		gocf.String("amazonaws.com"))

	// Tell the user we're ignoring any Arns provided, since it doesn't make sense for
	// this.
	if nil != perm.BasePermission.SourceArn &&
		perm.BasePermission.sourceArnExpr(cloudformationLogsSourceArnParts...).String() != wildcardArn.String() {
		logger.WithFields(logrus.Fields{
			"Arn": perm.BasePermission.sourceArnExpr(cloudformationEventsSourceArnParts...),
		}).Warn("CloudWatchLogs do not support literal ARN values")
	}

	// Make sure we grant InvokeFunction privileges to CloudWatchLogs
	lambdaInvokePermission, err := perm.BasePermission.export(regionalPrincipal,
		cloudformationLogsSourceArnParts,
		lambdaFunctionDisplayName,
		lambdaLogicalCFResourceName,
		template,
		S3Bucket,
		S3Key,
		logger)
	if nil != err {
		return "", err
	}

	// Then we need to uniqueify the rule names s.t. we prevent
	// collisions with other stacks.
	configurationResourceNames := make(map[string]int, 0)
	// Store the last name.  We'll do a uniqueness check when exiting the loop,
	// and if that passes, the last name will also be the unique one.
	var configurationResourceName string
	// Create the CustomResource entries
	globallyUniqueFilters := make(map[string]CloudWatchLogsSubscriptionFilter, len(perm.Filters))
	for eachFilterName, eachFilter := range perm.Filters {
		filterPrefix := fmt.Sprintf("%s_%s", serviceName, eachFilterName)
		uniqueFilterName := CloudFormationResourceName(filterPrefix, lambdaLogicalCFResourceName)
		globallyUniqueFilters[uniqueFilterName] = eachFilter

		// The ARN we supply to IAM is built up using the user supplied groupname
		cloudWatchLogsArn := gocf.Join("",
			gocf.String("arn:aws:logs:"),
			gocf.Ref("AWS::Region"),
			gocf.String(":"),
			gocf.Ref("AWS::AccountId"),
			gocf.String(":log-group:"),
			gocf.String(eachFilter.LogGroupName),
			gocf.String(":log-stream:*"))

		lastConfigurationResourceName, ensureCustomHandlerError := ensureCustomResourceHandler(serviceName,
			cloudformationresources.CloudWatchLogsLambdaEventSource,
			cloudWatchLogsArn,
			[]string{},
			template,
			S3Bucket,
			S3Key,
			logger)
		if nil != ensureCustomHandlerError {
			return "", err
		}
		configurationResourceNames[configurationResourceName] = 1
		configurationResourceName = lastConfigurationResourceName
	}
	if len(configurationResourceNames) > 1 {
		return "", fmt.Errorf("Internal integrity check failed. Multiple configurators (%d) provisioned for CloudWatchLogs",
			len(configurationResourceNames))
	}

	// Get the single configurator name from the

	// Add the custom resource that uses this...
	//////////////////////////////////////////////////////////////////////////////

	newResource, newResourceError := newCloudFormationResource(cloudformationresources.CloudWatchLogsLambdaEventSource, logger)
	if nil != newResourceError {
		return "", newResourceError
	}
	customResource := newResource.(*cloudformationresources.CloudWatchLogsLambdaEventSourceResource)
	customResource.ServiceToken = gocf.GetAtt(configurationResourceName, "Arn")
	customResource.LambdaTargetArn = gocf.GetAtt(lambdaLogicalCFResourceName, "Arn")
	// Build up the filters...
	customResource.Filters = make([]*cloudformationresources.CloudWatchLogsLambdaEventSourceFilter, 0)
	for eachName, eachFilter := range globallyUniqueFilters {
		customResource.Filters = append(customResource.Filters,
			&cloudformationresources.CloudWatchLogsLambdaEventSourceFilter{
				Name:         gocf.String(eachName),
				Pattern:      gocf.String(eachFilter.FilterPattern),
				LogGroupName: gocf.String(eachFilter.LogGroupName),
			})

	}

	resourceInvokerName := CloudFormationResourceName("ConfigCloudWatchLogs",
		lambdaLogicalCFResourceName,
		perm.BasePermission.SourceAccount)
	// Add it
	cfResource := template.AddResource(resourceInvokerName, customResource)

	cfResource.DependsOn = append(cfResource.DependsOn,
		lambdaInvokePermission,
		lambdaLogicalCFResourceName,
		configurationResourceName)
	return "", nil
}

func (perm CloudWatchLogsPermission) descriptionInfo() ([]descriptionNode, error) {
	var nodes []descriptionNode
	for eachFilterName, eachFilterDef := range perm.Filters {
		nodes = append(nodes, descriptionNode{
			Name:     describeInfoArn(eachFilterDef.LogGroupName),
			Relation: fmt.Sprintf("%s (%s)", eachFilterName, eachFilterDef.FilterPattern),
		})
	}
	return nodes, nil
}

//
// END - CloudWatchLogsPermission
///////////////////////////////////////////////////////////////////////////////////
