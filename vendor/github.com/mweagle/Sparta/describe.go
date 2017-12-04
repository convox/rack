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

const (
	nodeColorService     = "#EFEFEF"
	nodeColorEventSource = "#FBBB06"
	nodeColorLambda      = "#F58206"
	nodeColorAPIGateway  = "#06B5F5"
	nodeNameAPIGateway   = "API Gateway"
)

// RE for sanitizing golang/JS layer
var reSanitizeMermaidNodeName = regexp.MustCompile(`[\W\s]+`)
var reSanitizeMermaidLabelValue = regexp.MustCompile(`[\{\}\"\[\]']+`)

func mermaidNodeName(sourceName string) string {
	return reSanitizeMermaidNodeName.ReplaceAllString(sourceName, "x")
}

func mermaidLabelValue(labelText string) string {
	return reSanitizeMermaidLabelValue.ReplaceAllString(labelText, "")
}

func writeNode(writer io.Writer, nodeName string, nodeColor string, extraStyles string) {
	if "" != extraStyles {
		extraStyles = fmt.Sprintf(",%s", extraStyles)
	}
	sanitizedName := mermaidNodeName(nodeName)
	fmt.Fprintf(writer, "style %s fill:%s,stroke:#000,stroke-width:1px%s;\n", sanitizedName, nodeColor, extraStyles)
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
	s3BucketName string,
	buildTags string,
	linkFlags string,
	outputWriter io.Writer,
	workflowHooks *WorkflowHooks,
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
		s3BucketName,
		false,
		false,
		"N/A",
		"",
		buildTags,
		linkFlags,
		&cloudFormationTemplate,
		workflowHooks,
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
	writeNode(&b, serviceName, nodeColorService, "color:white,font-weight:bold,stroke-width:4px")

	for _, eachLambda := range lambdaAWSInfos {
		// Create the node...
		writeNode(&b, eachLambda.lambdaFunctionName(), nodeColorLambda, "")
		writeLink(&b, eachLambda.lambdaFunctionName(), serviceName, "")

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
					nodeColor = nodeColorEventSource
				}
				writeNode(&b, name, nodeColor, "border-style:dotted")
				writeLink(&b, name, eachLambda.lambdaFunctionName(), strings.Replace(link, "\n", "<br><br>", -1))
			}
		}

		for _, eachEventSourceMapping := range eachLambda.EventSourceMappings {
			writeNode(&b, eachEventSourceMapping.EventSourceArn, nodeColorEventSource, "border-style:dotted")
			writeLink(&b, eachEventSourceMapping.EventSourceArn, eachLambda.lambdaFunctionName(), "")
		}
	}

	// API?
	if nil != api {
		// Create the APIGateway virtual node && connect it to the application
		writeNode(&b, nodeNameAPIGateway, nodeColorAPIGateway, "")

		for _, eachResource := range api.resources {
			for eachMethod := range eachResource.Methods {
				// Create the PATH node
				var nodeName = fmt.Sprintf("%s - %s", eachMethod, eachResource.pathPart)
				writeNode(&b, nodeName, nodeColorAPIGateway, "")
				writeLink(&b, nodeNameAPIGateway, nodeName, "")
				writeLink(&b, nodeName, eachResource.parentLambda.lambdaFunctionName(), "")
			}
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
		_escFSMustString(false, "/resources/bootstrap/lumen/bootstrap.min.css"),
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
