package cloudformation

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/Sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	gocf "github.com/mweagle/go-cloudformation"

	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

var cloudFormationStackTemplateMap map[string]*gocf.Template

func init() {
	cloudFormationStackTemplateMap = make(map[string]*gocf.Template, 0)
	rand.Seed(time.Now().Unix())
}

// RE to ensure CloudFormation compatible resource names
// Issue: https://github.com/mweagle/Sparta/issues/8
// Ref: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/resources-section-structure.html
var reCloudFormationInvalidChars = regexp.MustCompile("[^A-Za-z0-9]+")

////////////////////////////////////////////////////////////////////////////////
// Private
////////////////////////////////////////////////////////////////////////////////

// BEGIN - templateConverter
// Struct to encapsulate transforming data into
type templateConverter struct {
	templateReader          io.Reader
	additionalTemplateProps map[string]interface{}
	// internals
	expandedTemplate string
	contents         []gocf.Stringable
	conversionError  error
}

func (converter *templateConverter) expandTemplate() *templateConverter {
	if nil != converter.conversionError {
		return converter
	}
	templateDataBytes, templateDataErr := ioutil.ReadAll(converter.templateReader)
	if nil != templateDataErr {
		converter.conversionError = templateDataErr
		return converter
	}
	templateData := string(templateDataBytes)

	parsedTemplate, templateErr := template.New("CloudFormation").Parse(templateData)
	if nil != templateErr {
		converter.conversionError = templateDataErr
		return converter
	}
	output := &bytes.Buffer{}
	executeErr := parsedTemplate.Execute(output, converter.additionalTemplateProps)
	if nil != executeErr {
		converter.conversionError = executeErr
		return converter
	}
	converter.expandedTemplate = output.String()
	return converter
}

func (converter *templateConverter) parseData() *templateConverter {
	if converter.conversionError != nil {
		return converter
	}
	reAWSProp := regexp.MustCompile("\\{\\s*\"\\s*(Ref|Fn::GetAtt|Fn::FindInMap)")
	splitData := strings.Split(converter.expandedTemplate, "\n")
	splitDataLineCount := len(splitData)

	for eachLineIndex, eachLine := range splitData {
		curContents := eachLine
		for len(curContents) != 0 {

			matchInfo := reAWSProp.FindStringSubmatchIndex(curContents)
			if nil != matchInfo {
				// If there's anything at the head, push it.
				if matchInfo[0] != 0 {
					head := curContents[0:matchInfo[0]]
					converter.contents = append(converter.contents, gocf.String(fmt.Sprintf("%s", head)))
					curContents = curContents[len(head):]
				}

				// There's at least one match...find the closing brace...
				var parsed map[string]interface{}
				for indexPos, eachChar := range curContents {
					if string(eachChar) == "}" {
						testBlock := curContents[0 : indexPos+1]
						err := json.Unmarshal([]byte(testBlock), &parsed)
						if err == nil {
							parsedContents, parsedContentsErr := parseFnJoinExpr(parsed)
							if nil != parsedContentsErr {
								converter.conversionError = parsedContentsErr
								return converter
							}
							converter.contents = append(converter.contents, parsedContents)
							curContents = curContents[indexPos+1:]
							if len(curContents) <= 0 && (eachLineIndex < (splitDataLineCount - 1)) {
								converter.contents = append(converter.contents, gocf.String("\n"))
							}
							break
						}
					}
				}
				if nil == parsed {
					// We never did find the end...
					converter.conversionError = fmt.Errorf("Invalid CloudFormation JSON expression on line: %s", eachLine)
					return converter
				}
			} else {
				// No match, just include it iff there is another line afterwards
				newlineValue := ""
				if eachLineIndex < (splitDataLineCount - 1) {
					newlineValue = "\n"
				}
				// Always include a newline at a minimum
				appendLine := fmt.Sprintf("%s%s", curContents, newlineValue)
				if len(appendLine) != 0 {
					converter.contents = append(converter.contents, gocf.String(appendLine))
				}
				break
			}
		}
	}
	return converter
}

func (converter *templateConverter) results() (*gocf.StringExpr, error) {
	if nil != converter.conversionError {
		return nil, converter.conversionError
	}
	return gocf.Join("", converter.contents...), nil
}

// END - templateConverter

func existingStackTemplate(serviceName string,
	session *session.Session,
	logger *logrus.Logger) (*gocf.Template, error) {
	template, templateExists := cloudFormationStackTemplateMap[serviceName]
	if !templateExists {
		templateParams := &cloudformation.GetTemplateInput{
			StackName: aws.String(serviceName),
		}
		logger.WithFields(logrus.Fields{
			"Service": serviceName,
		}).Info("Fetching existing CloudFormation template")

		cloudformationSvc := cloudformation.New(session)
		rawTemplate, rawTemplateErr := cloudformationSvc.GetTemplate(templateParams)
		if nil != rawTemplateErr {
			if strings.Contains(rawTemplateErr.Error(), "does not exist") {
				template = nil
			} else {
				return nil, rawTemplateErr
			}
		} else {
			t := gocf.Template{}
			jsonDecodeErr := json.NewDecoder(strings.NewReader(*rawTemplate.TemplateBody)).Decode(&t)
			if nil != jsonDecodeErr {
				return nil, jsonDecodeErr
			}
			template = &t
		}
		cloudFormationStackTemplateMap[serviceName] = template
	} else {
		logger.WithFields(logrus.Fields{
			"Service": serviceName,
		}).Debug("Using cached CloudFormation Template resources")
	}

	return template, nil
}

func updateStackViaChangeSet(serviceName string,
	cfTemplate *gocf.Template,
	cfTemplateURL string,
	awsTags []*cloudformation.Tag,
	awsCloudFormation *cloudformation.CloudFormation,
	logger *logrus.Logger) error {

	// Create a change set name...
	changeSetRequestName := CloudFormationResourceName(fmt.Sprintf("%sChangeSet", serviceName))
	_, changesErr := CreateStackChangeSet(changeSetRequestName,
		serviceName,
		cfTemplate,
		cfTemplateURL,
		awsTags,
		awsCloudFormation,
		logger)
	if nil != changesErr {
		return changesErr
	}

	//////////////////////////////////////////////////////////////////////////////
	// Apply the change
	executeChangeSetInput := cloudformation.ExecuteChangeSetInput{
		ChangeSetName: aws.String(changeSetRequestName),
		StackName:     aws.String(serviceName),
	}
	executeChangeSetOutput, executeChangeSetError := awsCloudFormation.ExecuteChangeSet(&executeChangeSetInput)

	logger.WithFields(logrus.Fields{
		"ExecuteChangeSetOutput": executeChangeSetOutput,
	}).Debug("ExecuteChangeSet result")

	if nil == executeChangeSetError {
		logger.WithFields(logrus.Fields{
			"StackName": serviceName,
		}).Info("Issued ExecuteChangeSet request")
	}
	return executeChangeSetError

}

func existingLambdaResourceVersions(serviceName string,
	lambdaResourceName string,
	session *session.Session,
	logger *logrus.Logger) (*lambda.ListVersionsByFunctionOutput, error) {

	errorIsNotExist := func(apiError error) bool {
		return apiError != nil && strings.Contains(apiError.Error(), "does not exist")
	}

	logger.WithFields(logrus.Fields{
		"ResourceName": lambdaResourceName,
	}).Info("Fetching existing function versions")

	cloudFormationSvc := cloudformation.New(session)
	describeParams := &cloudformation.DescribeStackResourceInput{
		StackName:         aws.String(serviceName),
		LogicalResourceId: aws.String(lambdaResourceName),
	}
	describeResponse, describeResponseErr := cloudFormationSvc.DescribeStackResource(describeParams)
	logger.WithFields(logrus.Fields{
		"Response":    describeResponse,
		"ResponseErr": describeResponseErr,
	}).Debug("Describe response")
	if errorIsNotExist(describeResponseErr) {
		return nil, nil
	} else if describeResponseErr != nil {
		return nil, describeResponseErr
	}

	listVersionsParams := &lambda.ListVersionsByFunctionInput{
		FunctionName: describeResponse.StackResourceDetail.PhysicalResourceId,
		MaxItems:     aws.Int64(128),
	}
	lambdaSvc := lambda.New(session)
	listVersionsResp, listVersionsRespErr := lambdaSvc.ListVersionsByFunction(listVersionsParams)
	if errorIsNotExist(listVersionsRespErr) {
		return nil, nil
	} else if listVersionsRespErr != nil {
		return nil, listVersionsRespErr
	}
	logger.WithFields(logrus.Fields{
		"Response":    listVersionsResp,
		"ResponseErr": listVersionsRespErr,
	}).Debug("ListVersionsByFunction")
	return listVersionsResp, nil
}

func toExpressionSlice(input interface{}) ([]string, error) {
	var expressions []string
	slice, sliceOK := input.([]interface{})
	if !sliceOK {
		return nil, fmt.Errorf("Failed to convert to slice")
	}
	for _, eachValue := range slice {
		switch str := eachValue.(type) {
		case string:
			expressions = append(expressions, str)
		}
	}
	return expressions, nil
}
func parseFnJoinExpr(data map[string]interface{}) (*gocf.StringExpr, error) {
	if len(data) <= 0 {
		return nil, fmt.Errorf("FnJoinExpr data is empty")
	}
	for eachKey, eachValue := range data {
		switch eachKey {
		case "Ref":
			return gocf.Ref(eachValue.(string)).String(), nil
		case "Fn::GetAtt":
			attrValues, attrValuesErr := toExpressionSlice(eachValue)
			if nil != attrValuesErr {
				return nil, attrValuesErr
			}
			if len(attrValues) != 2 {
				return nil, fmt.Errorf("Invalid params for Fn::GetAtt: %s", eachValue)
			}
			return gocf.GetAtt(attrValues[0], attrValues[1]).String(), nil
		case "Fn::FindInMap":
			attrValues, attrValuesErr := toExpressionSlice(eachValue)
			if nil != attrValuesErr {
				return nil, attrValuesErr
			}
			if len(attrValues) != 3 {
				return nil, fmt.Errorf("Invalid params for Fn::FindInMap: %s", eachValue)
			}
			return gocf.FindInMap(attrValues[0], gocf.String(attrValues[1]), gocf.String(attrValues[2])), nil
		}
	}
	return nil, fmt.Errorf("Unsupported AWS Function detected: %#v", data)
}

func stackCapabilities(template *gocf.Template) []*string {
	// Only require IAM capability if the definition requires it.
	capabilities := make([]*string, 0)
	for _, eachResource := range template.Resources {
		if eachResource.Properties.CfnResourceType() == "AWS::IAM::Role" {
			found := false
			for _, eachElement := range capabilities {
				found = (found || (*eachElement == "CAPABILITY_IAM"))
			}
			if !found {
				capabilities = append(capabilities, aws.String("CAPABILITY_IAM"))
			}
		}
	}
	return capabilities
}

////////////////////////////////////////////////////////////////////////////////
// Public
////////////////////////////////////////////////////////////////////////////////

// S3AllKeysArnForBucket returns a CloudFormation-compatible Arn expression
// (string or Ref) for all bucket keys (`/*`).  The bucket
// parameter may be either a string or an interface{} ("Ref: "myResource")
// value
func S3AllKeysArnForBucket(bucket interface{}) *gocf.StringExpr {
	arnParts := []gocf.Stringable{gocf.String("arn:aws:s3:::")}

	switch bucket.(type) {
	case string:
		// Don't be smart if the Arn value is a user supplied literal
		arnParts = append(arnParts, gocf.String(bucket.(string)))
	case *gocf.StringExpr:
		arnParts = append(arnParts, bucket.(*gocf.StringExpr))
	case gocf.RefFunc:
		arnParts = append(arnParts, bucket.(gocf.RefFunc).String())
	default:
		panic(fmt.Sprintf("Unsupported SourceArn value type: %+v", bucket))
	}
	arnParts = append(arnParts, gocf.String("/*"))
	return gocf.Join("", arnParts...).String()
}

// S3ArnForBucket returns a CloudFormation-compatible Arn expression
// (string or Ref) suitable for template reference.  The bucket
// parameter may be either a string or an interface{} ("Ref: "myResource")
// value
func S3ArnForBucket(bucket interface{}) *gocf.StringExpr {
	arnParts := []gocf.Stringable{gocf.String("arn:aws:s3:::")}

	switch bucket.(type) {
	case string:
		// Don't be smart if the Arn value is a user supplied literal
		arnParts = append(arnParts, gocf.String(bucket.(string)))
	case *gocf.StringExpr:
		arnParts = append(arnParts, bucket.(*gocf.StringExpr))
	case gocf.RefFunc:
		arnParts = append(arnParts, bucket.(gocf.RefFunc).String())
	default:
		panic(fmt.Sprintf("Unsupported SourceArn value type: %+v", bucket))
	}
	return gocf.Join("", arnParts...).String()
}

// MapToResourceTags transforms a go map[string]string to a CloudFormation-compliant
// Tags representation.  See http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-resource-tags.html
func MapToResourceTags(tagMap map[string]string) []interface{} {
	var tags []interface{}
	for eachKey, eachValue := range tagMap {
		tags = append(tags, map[string]interface{}{
			"Key":   eachKey,
			"Value": eachValue,
		})
	}
	return tags
}

// ConvertToTemplateExpression transforms the templateData contents into
// an Fn::Join- compatible representation for template serialization.
// The templateData contents may include both golang text/template properties
// and single-line JSON Fn::Join supported serializations.
func ConvertToTemplateExpression(templateData io.Reader, additionalUserTemplateProperties map[string]interface{}) (*gocf.StringExpr, error) {
	converter := &templateConverter{
		templateReader:          templateData,
		additionalTemplateProps: additionalUserTemplateProperties,
	}
	return converter.expandTemplate().parseData().results()
}

// AutoIncrementingLambdaVersionInfo is dynamically populated during
// a call AddAutoIncrementingLambdaVersionResource. The VersionHistory
// is a map of published versions to their CloudFormation resource names
type AutoIncrementingLambdaVersionInfo struct {
	// The version that will be published as part of this operation
	CurrentVersion int
	// The CloudFormation resource name that defines the
	// AWS::Lambda::Version resource to be included with this operation
	CurrentVersionResourceName string
	// The version history that maps a published version value
	// to its CloudFormation resource name. Used for defining lagging
	// indicator Alias values
	VersionHistory map[int]string
}

// AddAutoIncrementingLambdaVersionResource inserts a new
// AWS::Lambda::Version resource into the template. It uses
// the existing CloudFormation template representation
// to determine the version index to append. The returned
// map is from `versionIndex`->`CloudFormationResourceName`
// to support second-order AWS::Lambda::Alias records on a
// per-version level
func AddAutoIncrementingLambdaVersionResource(serviceName string,
	lambdaResourceName string,
	cfTemplate *gocf.Template,
	logger *logrus.Logger) (*AutoIncrementingLambdaVersionInfo, error) {

	// Get the template
	session, sessionErr := session.NewSession()
	if sessionErr != nil {
		return nil, sessionErr
	}

	// Get the current template - for each version we find in the version listing
	// we look up the actual CF resource and copy it into this template
	existingStackDefinition, existingStackDefinitionErr := existingStackTemplate(serviceName,
		session,
		logger)
	if nil != existingStackDefinitionErr {
		return nil, existingStackDefinitionErr
	}

	existingVersions, existingVersionsErr := existingLambdaResourceVersions(serviceName,
		lambdaResourceName,
		session,
		logger)
	if nil != existingVersionsErr {
		return nil, existingVersionsErr
	}

	// Initialize the auto incrementing version struct
	autoIncrementingLambdaVersionInfo := AutoIncrementingLambdaVersionInfo{
		CurrentVersion:             0,
		CurrentVersionResourceName: "",
		VersionHistory:             make(map[int]string, 0),
	}

	lambdaVersionResourceName := func(versionIndex int) string {
		return CloudFormationResourceName(lambdaResourceName,
			"version",
			strconv.Itoa(versionIndex))
	}

	if nil != existingVersions {
		// Add the CloudFormation resource
		logger.WithFields(logrus.Fields{
			"VersionCount": len(existingVersions.Versions) - 1, // Ignore $LATEST
			"ResourceName": lambdaResourceName,
		}).Info("Total number of published versions")

		for _, eachEntry := range existingVersions.Versions {
			versionIndex, versionIndexErr := strconv.Atoi(*eachEntry.Version)
			if nil == versionIndexErr {
				// Find the existing resource...
				versionResourceName := lambdaVersionResourceName(versionIndex)
				if nil == existingStackDefinition {
					return nil, fmt.Errorf("Unable to find existing Version resource in nil Template")
				}
				cfResourceDefinition, cfResourceDefinitionExists := existingStackDefinition.Resources[versionResourceName]
				if !cfResourceDefinitionExists {
					return nil, fmt.Errorf("Unable to find existing Version resource (Resource: %s, Version: %d) in template",
						versionResourceName,
						versionIndex)
				}
				cfTemplate.Resources[versionResourceName] = cfResourceDefinition
				// Add the CloudFormation resource
				logger.WithFields(logrus.Fields{
					"Version":      versionIndex,
					"ResourceName": versionResourceName,
				}).Debug("Preserving Lambda version")

				// Store the state, tracking the latest version
				autoIncrementingLambdaVersionInfo.VersionHistory[versionIndex] = versionResourceName
				if versionIndex > autoIncrementingLambdaVersionInfo.CurrentVersion {
					autoIncrementingLambdaVersionInfo.CurrentVersion = versionIndex
				}
			}
		}
	}

	// Bump the version and add a new entry...
	autoIncrementingLambdaVersionInfo.CurrentVersion++
	versionResource := &gocf.LambdaVersion{
		FunctionName: gocf.GetAtt(lambdaResourceName, "Arn").String(),
	}
	autoIncrementingLambdaVersionInfo.CurrentVersionResourceName = lambdaVersionResourceName(autoIncrementingLambdaVersionInfo.CurrentVersion)
	cfTemplate.AddResource(autoIncrementingLambdaVersionInfo.CurrentVersionResourceName, versionResource)

	// Log the version we're about to publish...
	logger.WithFields(logrus.Fields{
		"ResourceName": lambdaResourceName,
		"StackVersion": autoIncrementingLambdaVersionInfo.CurrentVersion,
	}).Info("Inserting new version resource")

	return &autoIncrementingLambdaVersionInfo, nil
}

// StackEvents returns the slice of cloudformation.StackEvents for the given stackID or stackName
func StackEvents(stackID string,
	eventFilterLowerBound time.Time,
	awsSession *session.Session) ([]*cloudformation.StackEvent, error) {
	cfService := cloudformation.New(awsSession)
	var events []*cloudformation.StackEvent

	nextToken := ""
	for {
		params := &cloudformation.DescribeStackEventsInput{
			StackName: aws.String(stackID),
		}
		if len(nextToken) > 0 {
			params.NextToken = aws.String(nextToken)
		}

		resp, err := cfService.DescribeStackEvents(params)
		if nil != err {
			return nil, err
		}
		for _, eachEvent := range resp.StackEvents {
			if eachEvent.Timestamp.After(eventFilterLowerBound) {
				events = append(events, eachEvent)
			}
		}
		if nil == resp.NextToken {
			break
		} else {
			nextToken = *resp.NextToken
		}
	}
	return events, nil
}

// WaitForStackOperationCompleteResult encapsulates the stackInfo
// following a WaitForStackOperationComplete call
type WaitForStackOperationCompleteResult struct {
	operationSuccessful bool
	stackInfo           *cloudformation.Stack
}

// WaitForStackOperationComplete is a blocking, polling based call that
// periodically fetches the stackID set of events and uses the state value
// to determine if an operation is complete
func WaitForStackOperationComplete(stackID string,
	pollingMessage string,
	awsCloudFormation *cloudformation.CloudFormation,
	logger *logrus.Logger) (*WaitForStackOperationCompleteResult, error) {

	result := &WaitForStackOperationCompleteResult{}

	// Poll for the current stackID state, and
	describeStacksInput := &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackID),
	}
	for waitComplete := false; !waitComplete; {
		sleepDuration := time.Duration(11+rand.Int31n(13)) * time.Second
		time.Sleep(sleepDuration)

		describeStacksOutput, err := awsCloudFormation.DescribeStacks(describeStacksInput)
		if nil != err {
			// TODO - add retry iff we're RateExceeded due to collective access
			return nil, err
		}
		if len(describeStacksOutput.Stacks) <= 0 {
			return nil, fmt.Errorf("Failed to enumerate stack info: %v", *describeStacksInput.StackName)
		}
		result.stackInfo = describeStacksOutput.Stacks[0]
		switch *(result.stackInfo).StackStatus {
		case cloudformation.StackStatusCreateComplete,
			cloudformation.StackStatusUpdateComplete:
			result.operationSuccessful = true
			waitComplete = true
		case
			// Include DeleteComplete as new provisions will automatically rollback
			cloudformation.StackStatusDeleteComplete,
			cloudformation.StackStatusCreateFailed,
			cloudformation.StackStatusDeleteFailed,
			cloudformation.StackStatusRollbackFailed,
			cloudformation.StackStatusRollbackComplete,
			cloudformation.StackStatusUpdateRollbackComplete:
			result.operationSuccessful = false
			waitComplete = true
		default:
			logger.Info(pollingMessage)
		}
	}
	return result, nil
}

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

// UploadTemplate marshals the given cfTemplate and uploads it to the
// supplied bucket using the given KeyName
func UploadTemplate(serviceName string,
	cfTemplate *gocf.Template,
	s3Bucket string,
	s3KeyName string,
	awsSession *session.Session,
	logger *logrus.Logger) (string, error) {

	logger.WithFields(logrus.Fields{
		"Key":    s3KeyName,
		"Bucket": s3Bucket,
	}).Info("Uploading CloudFormation template")

	s3Uploader := s3manager.NewUploader(awsSession)

	// Serialize the template and upload it
	cfTemplateJSON, err := json.Marshal(cfTemplate)
	if err != nil {
		logger.Error("Failed to Marshal CloudFormation template: ", err.Error())
		return "", err
	}

	// Upload the actual CloudFormation template to S3 to maximize the template
	// size limit
	// Ref: http://docs.aws.amazon.com/AWSCloudFormation/latest/APIReference/API_CreateStack.html
	contentBody := string(cfTemplateJSON)
	uploadInput := &s3manager.UploadInput{
		Bucket:      &s3Bucket,
		Key:         &s3KeyName,
		ContentType: aws.String("application/json"),
		Body:        strings.NewReader(contentBody),
	}
	templateUploadResult, templateUploadResultErr := s3Uploader.Upload(uploadInput)
	if nil != templateUploadResultErr {
		return "", templateUploadResultErr
	}

	// Be transparent
	logger.WithFields(logrus.Fields{
		"URL": templateUploadResult.Location,
	}).Info("Template uploaded")
	return templateUploadResult.Location, nil
}

// StackExists returns whether the given stackName or stackID currently exists
func StackExists(stackNameOrID string, awsSession *session.Session, logger *logrus.Logger) (bool, error) {
	cf := cloudformation.New(awsSession)

	describeStacksInput := &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackNameOrID),
	}
	describeStacksOutput, err := cf.DescribeStacks(describeStacksInput)
	logger.WithFields(logrus.Fields{
		"DescribeStackOutput": describeStacksOutput,
	}).Debug("DescribeStackOutput results")

	exists := false
	if err != nil {
		logger.WithFields(logrus.Fields{
			"DescribeStackOutputError": err,
		}).Debug("DescribeStackOutput")

		// If the stack doesn't exist, then no worries
		if strings.Contains(err.Error(), "does not exist") {
			exists = false
		} else {
			return false, err
		}
	} else {
		exists = true
	}
	return exists, nil
}

// CreateStackChangeSet returns the DescribeChangeSetOutput
// for a given stack transformation
func CreateStackChangeSet(changeSetRequestName string,
	serviceName string,
	cfTemplate *gocf.Template,
	templateURL string,
	awsTags []*cloudformation.Tag,
	awsCloudFormation *cloudformation.CloudFormation,
	logger *logrus.Logger) (*cloudformation.DescribeChangeSetOutput, error) {

	capabilities := stackCapabilities(cfTemplate)
	changeSetInput := &cloudformation.CreateChangeSetInput{
		Capabilities:  capabilities,
		ChangeSetName: aws.String(changeSetRequestName),
		ClientToken:   aws.String(changeSetRequestName),
		Description:   aws.String(fmt.Sprintf("Change set for service: %s", serviceName)),
		StackName:     aws.String(serviceName),
		TemplateURL:   aws.String(templateURL),
	}
	if len(awsTags) != 0 {
		changeSetInput.Tags = awsTags
	}
	_, changeSetError := awsCloudFormation.CreateChangeSet(changeSetInput)
	if nil != changeSetError {
		return nil, changeSetError
	}

	logger.WithFields(logrus.Fields{
		"StackName": serviceName,
	}).Info("Issued CreateChangeSet request")

	describeChangeSetInput := cloudformation.DescribeChangeSetInput{
		ChangeSetName: aws.String(changeSetRequestName),
		StackName:     aws.String(serviceName),
	}

	var describeChangeSetOutput *cloudformation.DescribeChangeSetOutput
	for i := 0; i != 5; i++ {
		sleepDuration := time.Duration(3+rand.Int31n(5)) * time.Second
		time.Sleep(sleepDuration)

		changeSetOutput, describeChangeSetError := awsCloudFormation.DescribeChangeSet(&describeChangeSetInput)

		if nil != describeChangeSetError {
			return nil, describeChangeSetError
		}
		describeChangeSetOutput = changeSetOutput
		if nil != describeChangeSetOutput &&
			*describeChangeSetOutput.Status == "CREATE_COMPLETE" {
			break
		}
	}
	if nil == describeChangeSetOutput {
		return nil, fmt.Errorf("ChangeSet failed to stabilize: %s", changeSetRequestName)
	}
	logger.WithFields(logrus.Fields{
		"DescribeChangeSetOutput": describeChangeSetOutput,
	}).Debug("DescribeChangeSet result")

	//////////////////////////////////////////////////////////////////////////////
	// If there aren't any changes, then skip it...
	if len(describeChangeSetOutput.Changes) <= 0 {
		logger.WithFields(logrus.Fields{
			"StackName": serviceName,
		}).Info("No changes detected for service")

		// Delete it...
		_, deleteChangeSetResultErr := DeleteChangeSet(serviceName,
			changeSetRequestName,
			awsCloudFormation)
		return nil, deleteChangeSetResultErr
	}
	return describeChangeSetOutput, nil
}

// DeleteChangeSet is a utility function that attempts to delete
// an existing CloudFormation change set, with a bit of retry
// logic in case of EC
func DeleteChangeSet(stackName string,
	changeSetRequestName string,
	awsCloudFormation *cloudformation.CloudFormation) (*cloudformation.DeleteChangeSetOutput, error) {

	// Delete request...
	deleteChangeSetInput := cloudformation.DeleteChangeSetInput{
		ChangeSetName: aws.String(changeSetRequestName),
		StackName:     aws.String(stackName),
	}

	var delChangeSetResultErr error
	for i := 0; i != 5; i++ {
		deleteChangeSetResults, deleteChangeSetResultErr :=
			awsCloudFormation.DeleteChangeSet(&deleteChangeSetInput)
		if nil == deleteChangeSetResultErr {
			return deleteChangeSetResults, nil
		} else if strings.Contains(deleteChangeSetResultErr.Error(), "CREATE_IN_PROGRESS") {
			delChangeSetResultErr = deleteChangeSetResultErr
			sleepDuration := time.Duration(1+rand.Int31n(5)) * time.Second
			time.Sleep(sleepDuration)
		} else {
			return nil, deleteChangeSetResultErr
		}
	}
	return nil, delChangeSetResultErr
}

// ConvergeStackState ensures that the serviceName converges to the template
// state defined by cfTemplate. This function establishes a polling loop to determine
// when the stack operation has completed.
func ConvergeStackState(serviceName string,
	cfTemplate *gocf.Template,
	templateURL string,
	tags map[string]string,
	startTime time.Time,
	awsSession *session.Session,
	logger *logrus.Logger) (*cloudformation.Stack, error) {

	awsCloudFormation := cloudformation.New(awsSession)
	// Update the tags
	awsTags := make([]*cloudformation.Tag, 0)
	if nil != tags {
		for eachKey, eachValue := range tags {
			awsTags = append(awsTags,
				&cloudformation.Tag{
					Key:   aws.String(eachKey),
					Value: aws.String(eachValue),
				})
		}
	}
	exists, existsErr := StackExists(serviceName, awsSession, logger)
	if nil != existsErr {
		return nil, existsErr
	}
	stackID := ""
	if exists {
		updateErr := updateStackViaChangeSet(serviceName,
			cfTemplate,
			templateURL,
			awsTags,
			awsCloudFormation,
			logger)

		if nil != updateErr {
			return nil, updateErr
		}
		stackID = serviceName
	} else {
		// Create stack
		createStackInput := &cloudformation.CreateStackInput{
			StackName:        aws.String(serviceName),
			TemplateURL:      aws.String(templateURL),
			TimeoutInMinutes: aws.Int64(20),
			OnFailure:        aws.String(cloudformation.OnFailureDelete),
			Capabilities:     stackCapabilities(cfTemplate),
		}
		if len(awsTags) != 0 {
			createStackInput.Tags = awsTags
		}
		createStackResponse, createStackResponseErr := awsCloudFormation.CreateStack(createStackInput)
		if nil != createStackResponseErr {
			return nil, createStackResponseErr
		}
		logger.WithFields(logrus.Fields{
			"StackID": *createStackResponse.StackId,
		}).Info("Creating stack")

		stackID = *createStackResponse.StackId
	}
	// Wait for the operation to succeed
	pollingMessage := "Waiting for CloudFormation operation to complete"
	convergeResult, convergeErr := WaitForStackOperationComplete(stackID,
		pollingMessage,
		awsCloudFormation,
		logger)
	if nil != convergeErr {
		return nil, convergeErr
	}

	// If it didn't work, then output some failure information
	if !convergeResult.operationSuccessful {
		// Get the stack events and find the ones that failed.
		events, err := StackEvents(stackID, startTime, awsSession)
		if nil != err {
			return nil, err
		}

		logger.Error("Stack provisioning error")
		for _, eachEvent := range events {
			switch *eachEvent.ResourceStatus {
			case cloudformation.ResourceStatusCreateFailed,
				cloudformation.ResourceStatusDeleteFailed,
				cloudformation.ResourceStatusUpdateFailed:
				errMsg := fmt.Sprintf("\tError ensuring %s (%s): %s",
					aws.StringValue(eachEvent.ResourceType),
					aws.StringValue(eachEvent.LogicalResourceId),
					aws.StringValue(eachEvent.ResourceStatusReason))
				logger.Error(errMsg)
			default:
				// NOP
			}
		}
		return nil, fmt.Errorf("Failed to provision: %s", serviceName)
	} else if nil != convergeResult.stackInfo.Outputs {
		for _, eachOutput := range convergeResult.stackInfo.Outputs {
			logger.WithFields(logrus.Fields{
				"Key":         aws.StringValue(eachOutput.OutputKey),
				"Value":       aws.StringValue(eachOutput.OutputValue),
				"Description": aws.StringValue(eachOutput.Description),
			}).Info("Stack output")
		}
	}
	return convergeResult.stackInfo, nil
}

// If the platform specific implementation of user.Current()
// isn't available, go get something that's a "stable" user
// name
func defaultUserName() string {
	userName := os.Getenv("USER")
	if "" == userName {
		userName = os.Getenv("USERNAME")
	}
	if "" == userName {
		userName = fmt.Sprintf("user%d", os.Getuid())
	}
	return userName
}

// UserScopedStackName returns a CloudFormation stack
// name that takes into account the current username
/*
A stack name can contain only alphanumeric characters
(case sensitive) and hyphens. It must start with an alphabetic
\character and cannot be longer than 128 characters.
*/
func UserScopedStackName(basename string) string {
	platformUserName := platformUserName()
	if platformUserName == "" {
		return basename
	}
	userName := strings.Replace(platformUserName, " ", "-", -1)
	return fmt.Sprintf("%s-%s", basename, userName)
}

// ListStacks returns a slice of stacks that meet the given filter.
func ListStacks(session *session.Session,
	maxReturned int,
	stackFilters ...string) ([]*cloudformation.StackSummary, error) {

	listStackInput := &cloudformation.ListStacksInput{
		StackStatusFilter: []*string{},
	}
	for _, eachFilter := range stackFilters {
		listStackInput.StackStatusFilter = append(listStackInput.StackStatusFilter, aws.String(eachFilter))
	}
	cloudformationSvc := cloudformation.New(session)
	accumulator := []*cloudformation.StackSummary{}
	for {
		listResult, listResultErr := cloudformationSvc.ListStacks(listStackInput)
		if listResultErr != nil {
			return nil, listResultErr
		}
		accumulator = append(accumulator, listResult.StackSummaries...)
		if len(accumulator) >= maxReturned || listResult.NextToken == nil {
			return accumulator, nil
		}
		listStackInput.NextToken = listResult.NextToken
	}
}
