// +build !lambdabinary

package sparta

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"text/template"

	"github.com/Sirupsen/logrus"
)

// RE for sanitizing golang/JS layer
var reSanitizeMermaidNodeName = regexp.MustCompile("[\\W\\s]+")
var reSanitizeMermaidLabelValue = regexp.MustCompile("[\\{\\}\"\\[\\]']+")

func mermaidNodeName(sourceName string) string {
	return reSanitizeMermaidNodeName.ReplaceAllString(sourceName, "x")
}

func mermaidLabelValue(labelText string) string {
	return reSanitizeMermaidLabelValue.ReplaceAllString(labelText, "")
}

func writeNode(writer io.Writer, nodeName string, nodeColor string) {
	sanitizedName := mermaidNodeName(nodeName)
	fmt.Fprintf(writer, "style %s fill:#%s,stroke:#000,stroke-width:1px;\n", sanitizedName, nodeColor)
	fmt.Fprintf(writer, "%s[%s]\n", sanitizedName, mermaidLabelValue(nodeName))
}

func writeLink(writer io.Writer, fromNode string, toNode string, label string) {
	sanitizedFrom := mermaidNodeName(fromNode)
	sanitizedTo := mermaidNodeName(toNode)

	if "" != label {
		fmt.Fprintf(writer, "%s-- \"%s\" -->%s\n", sanitizedFrom, mermaidLabelValue(label), sanitizedTo)
	} else {
		fmt.Fprintf(writer, "%s-->%s\n", sanitizedFrom, sanitizedTo)
	}
}

// Describe produces a graphical representation of a service's Lambda and data sources.  Typically
// automatically called as part of a compiled golang binary via the `describe` command
// line option.
func Describe(serviceName string,
	serviceDescription string,
	lambdaAWSInfos []*LambdaAWSInfo,
	api *API,
	s3Site *S3Site,
	outputWriter io.Writer,
	logger *logrus.Logger) error {

	validationErr := validateSpartaPreconditions(lambdaAWSInfos, logger)
	if validationErr != nil {
		return validationErr
	}

	var cloudFormationTemplate bytes.Buffer
	err := Provision(true,
		serviceName,
		serviceDescription,
		lambdaAWSInfos,
		api,
		s3Site,
		"S3Bucket",
		&cloudFormationTemplate,
		logger)
	if nil != err {
		return err
	}

	tmpl, err := template.New("description").Parse(_escFSMustString(false, "/resources/describe/template.html"))
	if err != nil {
		return errors.New(err.Error())
	}

	var b bytes.Buffer

	// Setup the root object
	writeNode(&b, serviceName, "2AF1EA")

	for _, eachLambda := range lambdaAWSInfos {
		// Create the node...
		writeNode(&b, eachLambda.lambdaFnName, "00A49F")
		writeLink(&b, eachLambda.lambdaFnName, serviceName, "")

		// Create permission & event mappings
		// functions declared in this
		for _, eachPermission := range eachLambda.Permissions {
			nodes, err := eachPermission.descriptionInfo()
			if nil != err {
				return err
			}

			for _, eachNode := range nodes {
				name := strings.TrimSpace(eachNode.Name)
				link := strings.TrimSpace(eachNode.Relation)
				// Style it to have the Amazon color
				nodeColor := eachNode.Color
				if "" == nodeColor {
					nodeColor = "F1702A"
				}
				writeNode(&b, name, nodeColor)
				writeLink(&b, name, eachLambda.lambdaFnName, strings.Replace(link, "\n", "<br><br>", -1))
			}
		}

		for _, eachEventSourceMapping := range eachLambda.EventSourceMappings {
			writeNode(&b, eachEventSourceMapping.EventSourceArn, "F1702A")
			writeLink(&b, eachEventSourceMapping.EventSourceArn, eachLambda.lambdaFnName, "")
		}
	}

	params := struct {
		SpartaVersion          string
		ServiceName            string
		ServiceDescription     string
		CloudFormationTemplate string
		BootstrapCSS           string
		MermaidCSS             string
		HighlightsCSS          string
		JQueryJS               string
		BootstrapJS            string
		MermaidJS              string
		HighlightsJS           string
		MermaidData            string
	}{
		SpartaVersion,
		serviceName,
		serviceDescription,
		cloudFormationTemplate.String(),
		_escFSMustString(false, "/resources/bootstrap/css/bootstrap.min.css"),
		_escFSMustString(false, "/resources/mermaid/mermaid.css"),
		_escFSMustString(false, "/resources/highlights/styles/vs.css"),
		_escFSMustString(false, "/resources/jquery/jquery-2.1.4.min.js"),
		_escFSMustString(false, "/resources/bootstrap/js/bootstrap.min.js"),
		_escFSMustString(false, "/resources/mermaid/mermaid.min.js"),
		_escFSMustString(false, "/resources/highlights/highlight.pack.js"),
		b.String(),
	}

	return tmpl.Execute(outputWriter, params)
}
