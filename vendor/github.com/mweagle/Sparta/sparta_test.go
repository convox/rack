package sparta

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/Sirupsen/logrus"
)

type StructHandler1 struct {
}

func (handler *StructHandler1) handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "StructHandler1 handler")
}

type StructHandler2 struct {
}

func (handler *StructHandler2) handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "StructHandler1 handler")
}

func testLambdaStructData() []*LambdaAWSInfo {
	var lambdaFunctions []*LambdaAWSInfo

	handler1 := &StructHandler1{}
	lambdaFn1 := HandleAWSLambda(LambdaName(handler1.handler),
		http.HandlerFunc(handler1.handler),
		LambdaExecuteARN)
	lambdaFunctions = append(lambdaFunctions, lambdaFn1)

	handler2 := &StructHandler2{}
	lambdaFn2 := HandleAWSLambda(LambdaName(handler2.handler),
		http.HandlerFunc(handler2.handler),
		LambdaExecuteARN)
	lambdaFunctions = append(lambdaFunctions, lambdaFn2)

	return lambdaFunctions
}

func testLambdaDoubleStructPtrData() []*LambdaAWSInfo {
	var lambdaFunctions []*LambdaAWSInfo

	handler1 := &StructHandler1{}
	lambdaFn1 := HandleAWSLambda(LambdaName(handler1.handler),
		http.HandlerFunc(handler1.handler),
		LambdaExecuteARN)
	lambdaFunctions = append(lambdaFunctions, lambdaFn1)

	handler2 := &StructHandler1{}
	lambdaFn2 := HandleAWSLambda(LambdaName(handler2.handler),
		http.HandlerFunc(handler2.handler),
		LambdaExecuteARN)
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
	logger, _ := NewLogger("info")
	var templateWriter bytes.Buffer
	err := Provision(true,
		"SampleProvision",
		"",
		testLambdaStructData(),
		nil,
		nil,
		os.Getenv("S3_BUCKET"),
		false,
		false,
		"testBuildID",
		"",
		"",
		"",
		&templateWriter,
		nil,
		logger)
	if nil != err {
		t.Fatal(err.Error())
	}
}

func TestDoubleRefStruct(t *testing.T) {
	logger, _ := NewLogger("info")
	var templateWriter bytes.Buffer
	err := Provision(true,
		"SampleProvision",
		"",
		testLambdaDoubleStructPtrData(),
		nil,
		nil,
		os.Getenv("S3_BUCKET"),
		false,
		false,
		"testBuildID",
		"",
		"",
		"",
		&templateWriter,
		nil,
		logger)

	if nil == err {
		t.Fatal("Failed to enforce lambda function uniqueness")
	}
}

func TestCustomResource(t *testing.T) {
	logger, _ := NewLogger("info")
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
	err := Provision(true,
		"SampleProvision",
		"",
		lambdaFuncs,
		nil,
		nil,
		os.Getenv("S3_BUCKET"),
		false,
		false,
		"testBuildID",
		"",
		"",
		"",
		&templateWriter,
		nil,
		logger)

	if nil != err {
		t.Fatal("Failed to accept unique user CustomResource functions")
	}
}

func TestDoubleRefCustomResource(t *testing.T) {
	logger, _ := NewLogger("info")
	lambdaFuncs := testLambdaStructData()

	for _, eachLambda := range lambdaFuncs {
		eachLambda.RequireCustomResource(IAMRoleDefinition{},
			userDefinedCustomResource1,
			nil,
			nil)
	}
	var templateWriter bytes.Buffer
	err := Provision(true,
		"SampleProvision",
		"",
		lambdaFuncs,
		nil,
		nil,
		os.Getenv("S3_BUCKET"),
		false,
		false,
		"testBuildID",
		"",
		"",
		"",
		&templateWriter,
		nil,
		logger)

	if nil == err {
		t.Fatal("Failed to reject duplicate user CustomResource functions")
	}
}

func TestSignatureVersion(t *testing.T) {
	logger, _ := NewLogger("info")

	lambdaFunctions := testLambdaDoubleStructPtrData()
	lambdaFunctions[0].Options = &LambdaFunctionOptions{
		SpartaOptions: &SpartaOptions{
			Name: fmt.Sprintf("Handler0"),
		},
	}
	lambdaFunctions[1].Options = &LambdaFunctionOptions{
		SpartaOptions: &SpartaOptions{
			Name: fmt.Sprintf("Handler1"),
		},
	}
	var templateWriter bytes.Buffer
	err := Provision(true,
		"TestOverlappingLambdas",
		"",
		lambdaFunctions,
		nil,
		nil,
		os.Getenv("S3_BUCKET"),
		false,
		false,
		"testBuildID",
		"",
		"",
		"",
		&templateWriter,
		nil,
		logger)

	if nil != err {
		t.Fatal("Failed to respect duplicate lambdas with user supplied names")
	} else {
		t.Logf("Rejected duplicate lambdas")
	}
}

func TestUserDefinedOverlappingLambdaNames(t *testing.T) {
	logger, _ := NewLogger("info")

	lambdaFunctions := testLambdaDoubleStructPtrData()
	for _, eachLambda := range lambdaFunctions {
		eachLambda.Options = &LambdaFunctionOptions{
			SpartaOptions: &SpartaOptions{
				Name: fmt.Sprintf("HandlerX"),
			},
		}
	}

	var templateWriter bytes.Buffer
	err := Provision(true,
		"TestOverlappingLambdas",
		"",
		lambdaFunctions,
		nil,
		nil,
		os.Getenv("S3_BUCKET"),
		false,
		false,
		"testBuildID",
		"",
		"",
		"",
		&templateWriter,
		nil,
		logger)

	if nil == err {
		t.Fatal("Failed to reject duplicate lambdas with overlapping user supplied names")
	} else {
		t.Logf("Rejected overlapping user supplied names")
	}
}

////////////////////////////////////////////////////////////////////////////////
// LEGACY
////////////////////////////////////////////////////////////////////////////////
func legacyLambdaSignature(event *json.RawMessage,
	context *LambdaContext,
	w http.ResponseWriter,
	logger *logrus.Logger) {
	logger.Info("Hello World: ", string(*event))
	fmt.Fprint(w, string(*event))
}

func TestLegacyLambdaSignature(t *testing.T) {
	logger, _ := NewLogger("info")
	lambdaFn := NewLambda(IAMRoleDefinition{}, legacyLambdaSignature, nil)

	lambdaFunctions := []*LambdaAWSInfo{
		lambdaFn,
	}

	var templateWriter bytes.Buffer
	err := Provision(true,
		"TestLegacyLambdaSignature",
		"",
		lambdaFunctions,
		nil,
		nil,
		os.Getenv("S3_BUCKET"),
		false,
		false,
		"testBuildID",
		"",
		"",
		"",
		&templateWriter,
		nil,
		logger)

	if err != nil {
		t.Fatal("Failed to build legacy Sparta NewLambda signature ")
	} else {
		t.Logf("Correctly supported NewLambda signature")
	}
}
