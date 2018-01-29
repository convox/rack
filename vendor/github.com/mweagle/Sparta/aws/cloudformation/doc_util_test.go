package cloudformation

import (
	"strings"
)

var sampleTemplate = `
BASIC_AUTH_USERNAME={{ .Username }}
CONCOURSE_BASIC_AUTH_PASSWORD={{ .Password }}
SPARTA_CICD_BINARY_PATH=/home/ubuntu/{{ .ServiceName }}.lambda.amd64
POSTGRES_ADDRESS={ "Fn::GetAtt" : [ "{{ .PostgreSQLCloudFormationResource }}" , "Endpoint.Address" ] }
`
var sampleTemplateProps = map[string]interface{}{
	"Username":                         "MyPassword",
	"Password":                         "MyPassword",
	"PostgreSQLCloudFormationResource": "DBInstance0bef52bca519f672fddf3a6e0cbf1325e0a3263c",
}

func ExampleConvertToTemplateExpression() {
	templateReader := strings.NewReader(sampleTemplate)
	ConvertToTemplateExpression(templateReader, sampleTemplateProps)
}
