package sparta

import (
	"bytes"
	"testing"

	gocf "github.com/crewjam/go-cloudformation"

	"github.com/Sirupsen/logrus"
)

type cloudFormationProvisionTestResource struct {
	gocf.CloudFormationCustomResource
	ServiceToken string
	TestKey      interface{}
}

func customResourceTestProvider(resourceType string) gocf.ResourceProperties {
	switch resourceType {
	case "Custom::ProvisionTestEmpty":
		{
			return &cloudFormationProvisionTestResource{}
		}
	default:
		return nil
	}
}

func init() {
	gocf.RegisterCustomResourceProvider(customResourceTestProvider)
}

func TestProvision(t *testing.T) {

	logger, err := NewLogger("info")
	var templateWriter bytes.Buffer
	err = Provision(true, "SampleProvision", "", testLambdaData(), nil, nil, "S3Bucket", &templateWriter, logger)
	if nil != err {
		t.Fatal(err.Error())
	}
}

func templateDecorator(serviceName string,
	lambdaResourceName string,
	lambdaResource gocf.LambdaFunction,
	resourceMetadata map[string]interface{},
	S3Bucket string,
	S3Key string,
	template *gocf.Template,
	logger *logrus.Logger) error {

	// Add an empty resource
	newResource, err := newCloudFormationResource("Custom::ProvisionTestEmpty", logger)
	if nil != err {
		return err
	}
	customResource := newResource.(*cloudFormationProvisionTestResource)
	customResource.ServiceToken = "arn:aws:sns:us-east-1:84969EXAMPLE:CRTest"
	customResource.TestKey = "Hello World"
	template.AddResource("ProvisionTestResource", customResource)

	// Add an output
	template.Outputs["OutputDecorationTest"] = &gocf.Output{
		Description: "Information about the value",
		Value:       gocf.String("My key"),
	}
	return nil
}

func TestDecorateProvision(t *testing.T) {

	lambdas := testLambdaData()
	lambdas[0].Decorator = templateDecorator

	logger, err := NewLogger("info")
	var templateWriter bytes.Buffer
	err = Provision(true, "SampleProvision", "", lambdas, nil, nil, "S3Bucket", &templateWriter, logger)
	if nil != err {
		t.Fatal(err.Error())
	}
}
