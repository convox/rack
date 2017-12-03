package sparta

import "fmt"

func javascriptExportNameForCustomResourceType(customResourceTypeName string) string {
	return sanitizedName(customResourceTypeName)
}
func lambdaExportNameForCustomResourceType(customResourceTypeName string) string {
	return fmt.Sprintf("index.%s", javascriptExportNameForCustomResourceType(customResourceTypeName))
}

func customResourceDescription(serviceName string, targetType string) string {
	return fmt.Sprintf("%s: CloudFormation CustomResource to configure %s",
		serviceName,
		targetType)
}
