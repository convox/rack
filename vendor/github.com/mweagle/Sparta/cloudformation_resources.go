package sparta

import (
	"errors"
	"fmt"
	"reflect"

	// Also included in lambda_permissions.go, but doubly included
	// here as the package's init() function handles registering
	// the resources we look up in this package.
	_ "github.com/mweagle/cloudformationresources"

	"github.com/Sirupsen/logrus"
	gocf "github.com/mweagle/go-cloudformation"
)

// See http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/pseudo-parameter-reference.html
const (
	// TagLogicalResourceID is the current logical resource name
	TagLogicalResourceID = "aws:cloudformation:logical-id"
	// TagResourceType is the type of the referred resource type
	TagResourceType = "sparta:cloudformation:restype"
	// TagStackRegion is the current stack's logical id
	TagStackRegion = "sparta:cloudformation:region"
	// TagStackID is the current stack's ID
	TagStackID = "aws:cloudformation:stack-id"
	// TagStackName is the current stack name
	TagStackName = "aws:cloudformation:stack-name"
)

var cloudformationTypeMapDiscoveryOutputs = map[string][]string{
	"AWS::DynamoDB::Table":    {"StreamArn"},
	"AWS::Kinesis::Stream":    {"Arn"},
	"AWS::Route53::RecordSet": {""},
	"AWS::S3::Bucket":         {"DomainName", "WebsiteURL"},
	"AWS::SNS::Topic":         {"TopicName"},
	"AWS::SQS::Queue":         {"Arn", "QueueName"},
}

func newCloudFormationResource(resourceType string, logger *logrus.Logger) (gocf.ResourceProperties, error) {
	resProps := gocf.NewResourceByType(resourceType)
	if nil == resProps {
		logger.WithFields(logrus.Fields{
			"Type": resourceType,
		}).Fatal("Failed to create CloudFormation CustomResource!")
		return nil, fmt.Errorf("Unsupported CustomResourceType: %s", resourceType)
	}
	return resProps, nil
}

func outputsForResource(template *gocf.Template,
	logicalResourceName string,
	logger *logrus.Logger) (map[string]interface{}, error) {

	item, ok := template.Resources[logicalResourceName]
	if !ok {
		return nil, nil
	}

	outputs := make(map[string]interface{})
	attrs, exists := cloudformationTypeMapDiscoveryOutputs[item.Properties.CfnResourceType()]
	if exists {
		outputs["Ref"] = gocf.Ref(logicalResourceName).String()
		outputs[TagResourceType] = item.Properties.CfnResourceType()

		for _, eachAttr := range attrs {
			outputs[eachAttr] = gocf.GetAtt(logicalResourceName, eachAttr)
		}

		// Any tags?
		r := reflect.ValueOf(item.Properties)
		tagsField := reflect.Indirect(r).FieldByName("Tags")
		if tagsField.IsValid() && !tagsField.IsNil() {
			outputs["Tags"] = tagsField.Interface()
		}
	}

	if len(outputs) != 0 {
		logger.WithFields(logrus.Fields{
			"ResourceName": logicalResourceName,
			"Outputs":      outputs,
		}).Debug("Resource Outputs")
	}

	return outputs, nil
}
func safeAppendDependency(resource *gocf.Resource, dependencyName string) {
	if nil == resource.DependsOn {
		resource.DependsOn = []string{}
	}
	resource.DependsOn = append(resource.DependsOn, dependencyName)
}
func safeMetadataInsert(resource *gocf.Resource, key string, value interface{}) {
	if nil == resource.Metadata {
		resource.Metadata = make(map[string]interface{})
	}
	resource.Metadata[key] = value
}

func safeMergeTemplates(sourceTemplate *gocf.Template, destTemplate *gocf.Template, logger *logrus.Logger) error {
	var mergeErrors []string

	// Append the custom resources
	for eachKey, eachLambdaResource := range sourceTemplate.Resources {
		_, exists := destTemplate.Resources[eachKey]
		if exists {
			errorMsg := fmt.Sprintf("Duplicate CloudFormation resource name: %s", eachKey)
			mergeErrors = append(mergeErrors, errorMsg)
		} else {
			destTemplate.Resources[eachKey] = eachLambdaResource
		}
	}

	// Append the custom Mappings
	for eachKey, eachMapping := range sourceTemplate.Mappings {
		_, exists := destTemplate.Mappings[eachKey]
		if exists {
			errorMsg := fmt.Sprintf("Duplicate CloudFormation Mapping name: %s", eachKey)
			mergeErrors = append(mergeErrors, errorMsg)
		} else {
			destTemplate.Mappings[eachKey] = eachMapping
		}
	}

	// Append the custom outputs
	for eachKey, eachLambdaOutput := range sourceTemplate.Outputs {
		_, exists := destTemplate.Outputs[eachKey]
		if exists {
			errorMsg := fmt.Sprintf("Duplicate CloudFormation output key name: %s", eachKey)
			mergeErrors = append(mergeErrors, errorMsg)
		} else {
			destTemplate.Outputs[eachKey] = eachLambdaOutput
		}
	}
	if len(mergeErrors) > 0 {
		logger.Error("Failed to update template. The following collisions were found:")
		for _, eachError := range mergeErrors {
			logger.Error("\t" + eachError)
		}
		return errors.New("Template merge failed")
	}
	return nil
}
