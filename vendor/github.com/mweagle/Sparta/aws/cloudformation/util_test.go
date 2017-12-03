package cloudformation

import (
	"encoding/json"
	"strings"
	"testing"
)

var conversionParams = map[string]interface{}{
	"Key1": "Value1",
	"Key2": "Value2",
}

var userdataPassingTests = []struct {
	input  string
	output []interface{}
}{
	{
		"HelloWorld",
		[]interface{}{
			`HelloWorld`,
		},
	},
	{
		"Hello {{ .Key1 }}",
		[]interface{}{
			`Hello Value1`,
		},
	},
	{
		`{{ .Key1 }}=={{ .Key2 }}`,
		[]interface{}{
			`Value1==Value2`,
		},
	},
	{
		`A { "Fn::GetAtt" : [ "ResName" , "AttrName" ] }`,
		[]interface{}{
			`A `,
			map[string]interface{}{
				"Fn::GetAtt": []string{"ResName", "AttrName"},
			},
		},
	},
	{
		`A { "Fn::GetAtt" : [ "ResName" , "AttrName" ] } B`,
		[]interface{}{
			`A `,
			map[string]interface{}{
				"Fn::GetAtt": []string{"ResName", "AttrName"},
			},
			` B`,
		},
	},
	{
		`{ "Fn::GetAtt" : [ "ResName" , "AttrName" ] }
A`,
		[]interface{}{
			map[string]interface{}{
				"Fn::GetAtt": []string{"ResName", "AttrName"},
			},
			"\n",
			"A",
		},
	},
	{
		`{"Ref": "AWS::Region"}`,
		[]interface{}{
			map[string]string{
				"Ref": "AWS::Region",
			},
		},
	},
	{
		`A {"Ref" : "AWS::Region"} B`,
		[]interface{}{
			"A ",
			map[string]string{
				"Ref": "AWS::Region",
			},
			" B",
		},
	},
	{
		`A
{"Ref" : "AWS::Region"}
B`,
		[]interface{}{
			"A\n",
			map[string]string{
				"Ref": "AWS::Region",
			},
			"\n",
			"B",
		},
	},
	{
		"{\"Ref\" : \"AWS::Region\"} = {\"Ref\" : \"AWS::AccountId\"}",
		[]interface{}{
			map[string]string{
				"Ref": "AWS::Region",
			},
			" = ",
			map[string]string{
				"Ref": "AWS::AccountId",
			},
		},
	},
}

/*
   "Fn::GetAtt" : []string{"ResName","AttrName"},
*/
func TestExpand(t *testing.T) {

	for _, eachTest := range userdataPassingTests {
		testReader := strings.NewReader(eachTest.input)
		expandResult, expandResultErr := ConvertToTemplateExpression(testReader, conversionParams)
		if nil != expandResultErr {
			t.Errorf("%s (Input: %s)", expandResultErr, eachTest.input)
		} else {
			testOutput := map[string]interface{}{
				"Fn::Join": []interface{}{
					"",
					eachTest.output,
				},
			}
			expectedResult, expectedResultErr := json.Marshal(testOutput)
			if nil != expectedResultErr {
				t.Error(expectedResultErr)
			} else {
				actualMarshal, actualMarshalErr := json.Marshal(expandResult)
				if nil != actualMarshalErr {
					t.Errorf("%s (Input: %s)", actualMarshalErr, eachTest.input)
				} else if string(expectedResult) != string(actualMarshal) {
					t.Errorf("Failed to validate\n")
					t.Errorf("\tEXPECTED: %s\n", string(expectedResult))
					t.Errorf("\tACTUAL: %s\n", string(actualMarshal))
				} else {
					t.Logf("Validated: %s == %s", string(expectedResult), string(actualMarshal))
				}
			}
		}
	}
}
