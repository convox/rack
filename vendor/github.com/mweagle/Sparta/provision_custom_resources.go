package sparta

import "fmt"

func scriptExportNameForCustomResourceType(customResourceTypeName string) string {
	return sanitizedName(customResourceTypeName)
}
func lambdaExportNameForCustomResourceType(customResourceTypeName string) string {
	return fmt.Sprintf("index.%s", scriptExportNameForCustomResourceType(customResourceTypeName))
}

func customResourceDescription(serviceName string, targetType string) string {
	return fmt.Sprintf("%s: CloudFormation CustomResource to configure %s",
		serviceName,
		targetType)
}
