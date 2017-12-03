// +build !lambdabinary

package sparta

import (
	"fmt"

	"github.com/Sirupsen/logrus"
)

var helperScript = fmt.Sprintf(`
#!/bin/bash -x
# File: explore.sh
eventData=%scat eventData.json | base64%s
echo "Base64 eventData: $eventData"
requestData="{\"context\": {}, \"event\":\"$eventData\"}"
curl -vs -X POST -H "Content-Type: application/json" --data "$requestData" $1
`,
	"`",
	"`")
var samplePayload = `{
	"myKey" : "myData"
}`

// Explore supports interactive command line invocation of the previously
// provisioned Sparta service
func Explore(lambdaAWSInfos []*LambdaAWSInfo, port int, logger *logrus.Logger) error {
	validationErr := validateSpartaPreconditions(lambdaAWSInfos, logger)
	if validationErr != nil {
		return validationErr
	}

	if 0 == port {
		port = 9999
	}
	urlHost := fmt.Sprintf("http://localhost:%d", port)
	logger.Info("The following URLs are available for testing.")

	msgText := ""

	// Get unique paths
	lambdaPaths := make(map[string]*LambdaAWSInfo)
	for _, eachLambdaInfo := range lambdaAWSInfos {
		lambdaPaths[eachLambdaInfo.lambdaFunctionName()] = eachLambdaInfo
	}

	for _, eachLambdaInfo := range lambdaPaths {
		functionURL := fmt.Sprintf("%s/%s", urlHost, eachLambdaInfo.lambdaFunctionName())
		logger.WithFields(logrus.Fields{
			"URL": functionURL,
		}).Info(eachLambdaInfo.lambdaFunctionName())

		if msgText == "" {
			msgText = fmt.Sprintf("\n\t./explore.sh %s\n", functionURL)
		}
	}
	logger.Info("Functions can be invoked via application/json over POST using a helper script")
	logger.Info("Create the following **explore.sh** script: \n", helperScript)

	// Generate the BASH file that includes everything...
	logger.Info("Then create eventData.json with the payload to submit:\n", samplePayload)

	logger.Info("Finally, submit the event data to one of your functions as in:\n", msgText)
	logger.Info("")
	logger.Info("You can also write standard httptest functions. See TestExplore() in:")
	logger.Info("https://github.com/mweagle/Sparta/blob/master/explore_test.go")
	// Start up the localhost server and publish the info
	return Execute(lambdaAWSInfos, port, 0, logger)
}
