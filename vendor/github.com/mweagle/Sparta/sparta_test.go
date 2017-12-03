package sparta

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/Sirupsen/logrus"
)

type StructHandler1 struct {
}

func (handler *StructHandler1) handler(event *json.RawMessage, context *LambdaContext, w http.ResponseWriter, logger *logrus.Logger) {
	fmt.Fprintf(w, "StructHandler1 handler")
}

type StructHandler2 struct {
}

func (handler *StructHandler2) handler(event *json.RawMessage, context *LambdaContext, w http.ResponseWriter, logger *logrus.Logger) {
	fmt.Fprintf(w, "StructHandler1 handler")
}

func testLambdaStructData() []*LambdaAWSInfo {
	var lambdaFunctions []*LambdaAWSInfo

	handler1 := &StructHandler1{}
	lambdaFn1 := NewLambda(LambdaExecuteARN, handler1.handler, nil)
	lambdaFunctions = append(lambdaFunctions, lambdaFn1)

	handler2 := &StructHandler2{}
	lambdaFn2 := NewLambda(LambdaExecuteARN, handler2.handler, nil)
	lambdaFunctions = append(lambdaFunctions, lambdaFn2)

	return lambdaFunctions
}

func testLambdaDoubleStructPtrData() []*LambdaAWSInfo {
	var lambdaFunctions []*LambdaAWSInfo

	handler1 := &StructHandler1{}
	lambdaFn1 := NewLambda(LambdaExecuteARN, handler1.handler, nil)
	lambdaFunctions = append(lambdaFunctions, lambdaFn1)

	handler2 := &StructHandler1{}
	lambdaFn2 := NewLambda(LambdaExecuteARN, handler2.handler, nil)
	lambdaFunctions = append(lambdaFunctions, lambdaFn2)

	return lambdaFunctions
}

func userDefinedCustomResource1(requestType string,
	stackID string,
	properties map[string]interface{},
	logger *logrus.Logger) (map[string]interface{}, error) {
	return nil, nil
}

func userDefinedCustomResource2(requestType string,
	stackID string,
	properties map[string]interface{},
	logger *logrus.Logger) (map[string]interface{}, error) {
	return nil, nil
}

func TestStruct(t *testing.T) {
	logger, err := NewLogger("info")
	var templateWriter bytes.Buffer
	err = Provision(true,
		"SampleProvision",
		"",
		testLambdaStructData(),
		nil,
		nil,
		"S3Bucket",
		&templateWriter,
		logger)
	if nil != err {
		t.Fatal(err.Error())
	}
}

func TestDoubleRefStruct(t *testing.T) {
	logger, err := NewLogger("info")
	var templateWriter bytes.Buffer
	err = Provision(true,
		"SampleProvision",
		"",
		testLambdaDoubleStructPtrData(),
		nil,
		nil,
		"S3Bucket",
		&templateWriter,
		logger)

	if nil == err {
		t.Fatal("Failed to enforce lambda function uniqueness")
	}
}

func TestCustomResource(t *testing.T) {
	logger, err := NewLogger("info")
	lambdaFuncs := testLambdaStructData()
	lambdaFuncs[0].RequireCustomResource(IAMRoleDefinition{},
		userDefinedCustomResource1,
		nil,
		nil)

	lambdaFuncs[1].RequireCustomResource(IAMRoleDefinition{},
		userDefinedCustomResource2,
		nil,
		nil)

	var templateWriter bytes.Buffer
	err = Provision(true,
		"SampleProvision",
		"",
		lambdaFuncs,
		nil,
		nil,
		"S3Bucket",
		&templateWriter,
		logger)

	if nil != err {
		t.Fatal("Failed to accept unique user CustomResource functions")
	}
}

func TestDoubleRefCustomResource(t *testing.T) {
	logger, err := NewLogger("info")
	lambdaFuncs := testLambdaStructData()

	for _, eachLambda := range lambdaFuncs {
		eachLambda.RequireCustomResource(IAMRoleDefinition{},
			userDefinedCustomResource1,
			nil,
			nil)
	}
	var templateWriter bytes.Buffer
	err = Provision(true,
		"SampleProvision",
		"",
		lambdaFuncs,
		nil,
		nil,
		"S3Bucket",
		&templateWriter,
		logger)

	if nil == err {
		t.Fatal("Failed to reject duplicate user CustomResource functions")
	}
}
