package sparta

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigateway"
	gocf "github.com/crewjam/go-cloudformation"
)

/*
"context" : {
  "apiId" : "$util.escapeJavaScript($context.apiId)",
  "method" : "$util.escapeJavaScript($context.httpMethod)",
  "requestId" : "$util.escapeJavaScript($context.requestId)",
  "resourceId" : "$util.escapeJavaScript($context.resourceId)",
  "resourcePath" : "$util.escapeJavaScript($context.resourcePath)",
  "stage" : "$util.escapeJavaScript($context.stage)",
  "identity" : {
    "accountId" : "$util.escapeJavaScript($context.identity.accountId)",
    "apiKey" : "$util.escapeJavaScript($context.identity.apiKey)",
    "caller" : "$util.escapeJavaScript($context.identity.caller)",
    "cognitoAuthenticationProvider" : "$util.escapeJavaScript($context.identity.cognitoAuthenticationProvider)",
    "cognitoAuthenticationType" : "$util.escapeJavaScript($context.identity.cognitoAuthenticationType)",
    "cognitoIdentityId" : "$util.escapeJavaScript($context.identity.cognitoIdentityId)",
    "cognitoIdentityPoolId" : "$util.escapeJavaScript($context.identity.cognitoIdentityPoolId)",
    "sourceIp" : "$util.escapeJavaScript($context.identity.sourceIp)",
    "user" : "$util.escapeJavaScript($context.identity.user)",
    "userAgent" : "$util.escapeJavaScript($context.identity.userAgent)",
    "userArn" : "$util.escapeJavaScript($context.identity.userArn)"
  }
*/

const (
	// OutputAPIGatewayURL is the keyname used in the CloudFormation Output
	// that stores the APIGateway provisioned URL
	// @enum OutputKey
	OutputAPIGatewayURL = "APIGatewayURL"

	// boolTrue is the string representation that CF needs
	boolTrue = "true"

	// boolTrue is the string representation that CF needs
	boolFalse = "false"
)

// APIGatewayIdentity represents the user identity of a request
// made on behalf of the API Gateway
type APIGatewayIdentity struct {
	// Account ID
	AccountID string `json:"accountId"`
	// API Key
	APIKey string `json:"apiKey"`
	// Caller
	Caller string `json:"caller"`
	// Cognito Authentication Provider
	CognitoAuthenticationProvider string `json:"cognitoAuthenticationProvider"`
	// Cognito Authentication Type
	CognitoAuthenticationType string `json:"cognitoAuthenticationType"`
	// CognitoIdentityId
	CognitoIdentityID string `json:"cognitoIdentityId"`
	// CognitoIdentityPoolId
	CognitoIdentityPoolID string `json:"cognitoIdentityPoolId"`
	// Source IP
	SourceIP string `json:"sourceIp"`
	// User
	User string `json:"user"`
	// User Agent
	UserAgent string `json:"userAgent"`
	// User ARN
	UserARN string `json:"userArn"`
}

// APIGatewayContext represents the context available to an AWS Lambda
// function that is invoked by an API Gateway integration.
type APIGatewayContext struct {
	// API ID
	APIID string `json:"apiId"`
	// HTTPMethod
	Method string `json:"method"`
	// Request ID
	RequestID string `json:"requestId"`
	// Resource ID
	ResourceID string `json:"resourceId"`
	// Resource Path
	ResourcePath string `json:"resourcePath"`
	// Stage
	Stage string `json:"stage"`
	// User identity
	Identity APIGatewayIdentity `json:"identity"`
}

// APIGatewayLambdaJSONEvent provides a pass through mapping
// of all whitelisted Parameters.  The transformation is defined
// by the resources/gateway/inputmapping_json.vtl template.
type APIGatewayLambdaJSONEvent struct {
	// HTTPMethod
	Method string `json:"method"`
	// Body, if available
	Body json.RawMessage `json:"body"`
	// Whitelisted HTTP headers
	Headers map[string]string `json:"headers"`
	// Whitelisted HTTP query params
	QueryParams map[string]string `json:"queryParams"`
	// Whitelisted path parameters
	PathParams map[string]string `json:"pathParams"`
	// Context information - http://docs.aws.amazon.com/apigateway/latest/developerguide/api-gateway-mapping-template-reference.html#context-variable-reference
	Context APIGatewayContext `json:"context"`
}

// Model proxies the AWS SDK's Model data.  See
// http://docs.aws.amazon.com/sdk-for-go/api/service/apigateway.html#type-Model
//
// TODO: Support Dynamic Model creation
type Model struct {
	Description string `json:",omitempty"`
	Name        string `json:",omitempty"`
	Schema      string `json:",omitempty"`
}

// Response proxies the AWS SDK's PutMethodResponseInput data.  See
// http://docs.aws.amazon.com/sdk-for-go/api/service/apigateway.html#type-PutMethodResponseInput
type Response struct {
	Parameters map[string]bool   `json:",omitempty"`
	Models     map[string]*Model `json:",omitempty"`
}

// IntegrationResponse proxies the AWS SDK's IntegrationResponse data.  See
// http://docs.aws.amazon.com/sdk-for-go/api/service/apigateway.html#type-IntegrationResponse
type IntegrationResponse struct {
	Parameters       map[string]string `json:",omitempty"`
	SelectionPattern string            `json:",omitempty"`
	Templates        map[string]string `json:",omitempty"`
}

// Integration proxies the AWS SDK's Integration data.  See
// http://docs.aws.amazon.com/sdk-for-go/api/service/apigateway.html#type-Integration
type Integration struct {
	Parameters         map[string]string
	RequestTemplates   map[string]string
	CacheKeyParameters []string
	CacheNamespace     string
	Credentials        string

	Responses map[int]*IntegrationResponse

	// Typically "AWS", but for OPTIONS CORS support is set to "MOCK"
	integrationType string
}

// Method proxies the AWS SDK's Method data.  See
// http://docs.aws.amazon.com/sdk-for-go/api/service/apigateway.html#type-Method
type Method struct {
	authorizationType string
	httpMethod        string
	APIKeyRequired    bool

	// Request data
	Parameters map[string]bool
	Models     map[string]*Model

	// Response map
	Responses map[int]*Response

	// Integration response map
	Integration Integration
}

// Resource proxies the AWS SDK's Resource data.  See
// http://docs.aws.amazon.com/sdk-for-go/api/service/apigateway.html#type-Resource
type Resource struct {
	pathPart     string
	parentLambda *LambdaAWSInfo
	Methods      map[string]*Method
}

// Stage proxies the AWS SDK's Stage data.  See
// http://docs.aws.amazon.com/sdk-for-go/api/service/apigateway.html#type-Stage
type Stage struct {
	name                string
	CacheClusterEnabled bool
	CacheClusterSize    string
	Description         string
	Variables           map[string]string
}

// API represents the AWS API Gateway data associated with a given Sparta app.  Proxies
// the AWS SDK's CreateRestApiInput data.  See
// http://docs.aws.amazon.com/sdk-for-go/api/service/apigateway.html#type-CreateRestApiInput
type API struct {
	// The API name
	// TODO: bind this to the stack name to prevent provisioning collisions.
	name string
	// Optional stage. If defined, the API will be deployed
	stage *Stage
	// Existing API to CloneFrom
	CloneFrom string
	// API Description
	Description string
	// Non-empty map of urlPaths->Resource definitions
	resources map[string]*Resource
	// Should CORS be enabled for this API?
	CORSEnabled bool
}

func corsMethodResponseParams() map[string]string {
	responseParams := make(map[string]string, 0)
	responseParams["method.response.header.Access-Control-Allow-Headers"] = boolTrue
	responseParams["method.response.header.Access-Control-Allow-Methods"] = boolTrue
	responseParams["method.response.header.Access-Control-Allow-Origin"] = boolTrue
	return responseParams
}

// DefaultMethodResponses returns the default set of Method HTTPStatus->Response
// pass through responses.  The successfulHTTPStatusCode param is the single
// 2XX response code to use for the method.
func methodResponses(successfulHTTPStatusCode int,
	userResponses map[int]*Response,
	corsEnabled bool) *gocf.APIGatewayMethodMethodResponseList {

	var responses gocf.APIGatewayMethodMethodResponseList
	if len(userResponses) != 0 {
		for eachStatusCode := range userResponses {
			methodResponse := gocf.APIGatewayMethodMethodResponse{
				StatusCode: gocf.String(strconv.Itoa(eachStatusCode)),
			}
			if corsEnabled {
				methodResponse.ResponseParameters = corsMethodResponseParams()
			}
			responses = append(responses, methodResponse)
		}
	} else {
		// Add the single successful response
		methodResponse := gocf.APIGatewayMethodMethodResponse{
			StatusCode: gocf.String(strconv.Itoa(successfulHTTPStatusCode)),
		}
		if corsEnabled {
			methodResponse.ResponseParameters = corsMethodResponseParams()
		}
		responses = append(responses, methodResponse)

		for i := 300; i <= 599; i++ {
			statusText := http.StatusText(i)
			if "" != statusText {

				methodResponse := gocf.APIGatewayMethodMethodResponse{
					StatusCode: gocf.String(strconv.Itoa(i)),
				}
				// TODO - handle user defined params
				if corsEnabled {
					methodResponse.ResponseParameters = corsMethodResponseParams()
				}
				responses = append(responses, methodResponse)
			}
		}
	}
	return &responses
}

func corsIntegrationResponseParams() map[string]string {
	responseParams := make(map[string]string, 0)
	responseParams["method.response.header.Access-Control-Allow-Headers"] = "'Content-Type,X-Amz-Date,Authorization,X-Api-Key'"
	responseParams["method.response.header.Access-Control-Allow-Methods"] = "'*'"
	responseParams["method.response.header.Access-Control-Allow-Origin"] = "'*'"
	return responseParams
}

func integrationResponses(successfulHTTPStatusCode int,
	userResponses map[int]*IntegrationResponse,
	corsEnabled bool) *gocf.APIGatewayMethodIntegrationIntegrationResponseList {

	// TODO - userResponses
	var integrationResponses gocf.APIGatewayMethodIntegrationIntegrationResponseList
	for i := 200; i <= 599; i++ {
		statusText := http.StatusText(i)
		if "" != statusText {
			regExp := fmt.Sprintf(".*%s.*", statusText)
			if i == successfulHTTPStatusCode {
				regExp = ""
			}
			integrationResponse := gocf.APIGatewayMethodIntegrationIntegrationResponse{
				ResponseTemplates: map[string]string{
					"application/json": "",
					"text/*":           "",
				},
				SelectionPattern: gocf.String(regExp),
				StatusCode:       gocf.String(strconv.Itoa(i)),
			}
			// TODO - handle user defined params
			if corsEnabled {
				integrationResponse.ResponseParameters = corsIntegrationResponseParams()
			}
			integrationResponses = append(integrationResponses, integrationResponse)
		}
	}
	return &integrationResponses
}

func defaultRequestTemplates() map[string]string {
	return map[string]string{
		"application/json":                  _escFSMustString(false, "/resources/provision/apigateway/inputmapping_json.vtl"),
		"text/plain":                        _escFSMustString(false, "/resources/provision/apigateway/inputmapping_default.vtl"),
		"application/x-www-form-urlencoded": _escFSMustString(false, "/resources/provision/apigateway/inputmapping_default.vtl"),
		"multipart/form-data":               _escFSMustString(false, "/resources/provision/apigateway/inputmapping_default.vtl"),
	}
}

func corsOptionsGatewayMethod(restAPIID gocf.Stringable, resourceID gocf.Stringable) *gocf.ApiGatewayMethod {
	methodResponse := gocf.APIGatewayMethodMethodResponse{
		StatusCode:         gocf.String("200"),
		ResponseParameters: corsMethodResponseParams(),
	}

	integrationResponse := gocf.APIGatewayMethodIntegrationIntegrationResponse{
		ResponseTemplates: map[string]string{
			"application/*": "",
			"text/*":        "",
		},
		StatusCode:         gocf.String("200"),
		ResponseParameters: corsIntegrationResponseParams(),
	}

	methodIntegrationIntegrationResponseList := gocf.APIGatewayMethodIntegrationIntegrationResponseList{}
	methodIntegrationIntegrationResponseList = append(methodIntegrationIntegrationResponseList,
		integrationResponse)
	methodResponseList := gocf.APIGatewayMethodMethodResponseList{}
	methodResponseList = append(methodResponseList, methodResponse)

	corsMethod := &gocf.ApiGatewayMethod{
		HttpMethod:        gocf.String("OPTIONS"),
		AuthorizationType: gocf.String("NONE"),
		RestApiId:         restAPIID.String(),
		ResourceId:        resourceID.String(),
		Integration: &gocf.APIGatewayMethodIntegration{
			Type: gocf.String("MOCK"),
			RequestTemplates: map[string]string{
				"application/json": "{\"statusCode\": 200}",
				"text/plain":       "statusCode: 200",
			},
			IntegrationResponses: &methodIntegrationIntegrationResponseList,
		},
		MethodResponses: &methodResponseList,
	}
	return corsMethod
}

func apiStageInfo(apiName string, stageName string, session *session.Session, noop bool, logger *logrus.Logger) (*apigateway.Stage, error) {
	logger.WithFields(logrus.Fields{
		"APIName":   apiName,
		"StageName": stageName,
	}).Info("Checking current APIGateway stage status")

	if noop {
		logger.Info("Bypassing APIGateway check to -n/-noop command line argument")
		return nil, nil
	}

	svc := apigateway.New(session)
	restApisInput := &apigateway.GetRestApisInput{
		Limit: aws.Int64(500),
	}

	restApisOutput, restApisOutputErr := svc.GetRestApis(restApisInput)
	if nil != restApisOutputErr {
		return nil, restApisOutputErr
	}
	// Find the entry that has this name
	restAPIID := ""
	for _, eachRestAPI := range restApisOutput.Items {
		if *eachRestAPI.Name == apiName {
			if restAPIID != "" {
				return nil, fmt.Errorf("Multiple RestAPI matches for API Name: %s", apiName)
			}
			restAPIID = *eachRestAPI.Id
		}
	}
	if "" == restAPIID {
		return nil, nil
	}
	// API exists...does the stage name exist?
	stagesInput := &apigateway.GetStagesInput{
		RestApiId: aws.String(restAPIID),
	}
	stagesOutput, stagesOutputErr := svc.GetStages(stagesInput)
	if nil != stagesOutputErr {
		return nil, stagesOutputErr
	}

	// Find this stage name...
	var matchingStageOutput *apigateway.Stage
	for _, eachStage := range stagesOutput.Item {
		if *eachStage.StageName == stageName {
			if nil != matchingStageOutput {
				return nil, fmt.Errorf("Multiple stage matches for name: %s", stageName)
			}
			matchingStageOutput = eachStage
		}
	}
	if nil != matchingStageOutput {
		logger.WithFields(logrus.Fields{
			"DeploymentId": *matchingStageOutput.DeploymentId,
			"LastUpdated":  matchingStageOutput.LastUpdatedDate,
			"CreatedDate":  matchingStageOutput.CreatedDate,
		}).Info("Checking current APIGateway stage status")
	} else {
		logger.Info("APIGateway stage has not been deployed")
	}

	return matchingStageOutput, nil
}

// export marshals the API data to a CloudFormation compatible representation
func (api *API) export(serviceName string,
	session *session.Session,
	S3Bucket string,
	S3Key string,
	roleNameMap map[string]*gocf.StringExpr,
	template *gocf.Template,
	noop bool,
	logger *logrus.Logger) error {

	apiGatewayResourceNameForPath := func(fullPath string) string {
		pathParts := strings.Split(fullPath, "/")
		return CloudFormationResourceName("%sResource", pathParts[0], fullPath)
	}
	apiGatewayResName := CloudFormationResourceName("APIGateway", api.name)

	// Create an API gateway entry
	apiGatewayRes := &gocf.ApiGatewayRestApi{
		Description:    gocf.String(api.Description),
		FailOnWarnings: gocf.Bool(false),
		Name:           gocf.String(api.name),
	}
	if "" != api.CloneFrom {
		apiGatewayRes.CloneFrom = gocf.String(api.CloneFrom)
	}
	if "" == api.Description {
		apiGatewayRes.Description = gocf.String(fmt.Sprintf("%s RestApi", serviceName))
	} else {
		apiGatewayRes.Description = gocf.String(api.Description)
	}

	template.AddResource(apiGatewayResName, apiGatewayRes)
	apiGatewayRestAPIID := gocf.Ref(apiGatewayResName)

	// List of all the method resources we're creating s.t. the
	// deployment can DependOn them
	var apiMethodCloudFormationResources []string
	for eachResourcePath, eachResourceDef := range api.resources {
		// First walk all the user resources and create intermediate paths
		// to repreesent all the resources
		var parentResource *gocf.StringExpr
		pathParts := strings.Split(strings.TrimLeft(eachResourceDef.pathPart, "/"), "/")
		pathAccumulator := []string{"/"}
		for index, eachPathPart := range pathParts {
			pathAccumulator = append(pathAccumulator, eachPathPart)
			resourcePathName := apiGatewayResourceNameForPath(strings.Join(pathAccumulator, "/"))
			if _, exists := template.Resources[resourcePathName]; !exists {
				cfResource := &gocf.ApiGatewayResource{
					RestApiId: apiGatewayRestAPIID.String(),
					PathPart:  gocf.String(eachPathPart),
				}
				if index <= 0 {
					cfResource.ParentId = gocf.GetAtt(apiGatewayResName, "RootResourceId")
				} else {
					cfResource.ParentId = parentResource
				}
				template.AddResource(resourcePathName, cfResource)
			}
			parentResource = gocf.Ref(resourcePathName).String()
		}

		// Add the lambda permission
		apiGatewayPermissionResourceName := CloudFormationResourceName("APIGatewayLambdaPerm", eachResourcePath)
		lambdaInvokePermission := &gocf.LambdaPermission{
			Action:       gocf.String("lambda:InvokeFunction"),
			FunctionName: gocf.GetAtt(eachResourceDef.parentLambda.logicalName(), "Arn"),
			Principal:    gocf.String(APIGatewayPrincipal),
		}
		template.AddResource(apiGatewayPermissionResourceName, lambdaInvokePermission)

		// BEGIN CORS - OPTIONS verb
		// Then if the api is CORS enabled, setup the options method
		if api.CORSEnabled {
			methodResourceName := CloudFormationResourceName(fmt.Sprintf("%s-OPTIONS", eachResourcePath), eachResourcePath)
			template.AddResource(methodResourceName,
				corsOptionsGatewayMethod(apiGatewayRestAPIID, parentResource))

			apiMethodCloudFormationResources = append(apiMethodCloudFormationResources, methodResourceName)
		}
		// END CORS - OPTIONS verb

		// BEGIN - user defined verbs
		for eachMethodName, eachMethodDef := range eachResourceDef.Methods {
			statusSuccessfulCode := http.StatusOK
			if eachMethodDef.httpMethod == "POST" {
				statusSuccessfulCode = http.StatusCreated
			}

			apiGatewayMethod := &gocf.ApiGatewayMethod{
				HttpMethod:        gocf.String(eachMethodName),
				AuthorizationType: gocf.String("NONE"),
				ResourceId:        parentResource.String(),
				RestApiId:         apiGatewayRestAPIID.String(),
				Integration: &gocf.APIGatewayMethodIntegration{
					IntegrationHttpMethod: gocf.String("POST"),
					Type:             gocf.String("AWS"),
					RequestTemplates: defaultRequestTemplates(),
					Uri: gocf.Join("",
						gocf.String("arn:aws:apigateway:"),
						gocf.Ref("AWS::Region"),
						gocf.String(":lambda:path/2015-03-31/functions/"),
						gocf.GetAtt(eachResourceDef.parentLambda.logicalName(), "Arn"),
						gocf.String("/invocations")),
				},
			}

			// Add the integration response RegExps
			apiGatewayMethod.Integration.IntegrationResponses = integrationResponses(statusSuccessfulCode, eachMethodDef.Integration.Responses, api.CORSEnabled)

			// Add outbound method responses
			apiGatewayMethod.MethodResponses = methodResponses(statusSuccessfulCode, eachMethodDef.Responses, api.CORSEnabled)

			prefix := fmt.Sprintf("%s%s", eachMethodDef.httpMethod, eachResourcePath)
			methodResourceName := CloudFormationResourceName(prefix, eachResourcePath, serviceName)
			res := template.AddResource(methodResourceName, apiGatewayMethod)
			res.DependsOn = append(res.DependsOn, apiGatewayPermissionResourceName)
			apiMethodCloudFormationResources = append(apiMethodCloudFormationResources, methodResourceName)
		}
	}
	// END

	if nil != api.stage {
		// Is the stack already deployed?
		stageName := api.stage.name
		stageInfo, stageInfoErr := apiStageInfo(api.name, stageName, session, noop, logger)
		if nil != stageInfoErr {
			return stageInfoErr
		}
		if nil == stageInfo {
			// Use a stable identifier so that we can update the existing deployment
			apiDeploymentResName := CloudFormationResourceName("APIGatewayDeployment", serviceName)
			apiDeployment := &gocf.ApiGatewayDeployment{
				Description: gocf.String(api.stage.Description),
				RestApiId:   apiGatewayRestAPIID.String(),
				StageName:   gocf.String(stageName),
				StageDescription: &gocf.APIGatewayDeploymentStageDescription{
					StageName:   gocf.String(api.stage.name),
					Description: gocf.String(api.stage.Description),
					Variables:   api.stage.Variables,
				},
			}
			if api.stage.CacheClusterEnabled {
				apiDeployment.StageDescription.CacheClusterEnabled = gocf.Bool(api.stage.CacheClusterEnabled)
			}
			if api.stage.CacheClusterSize != "" {
				apiDeployment.StageDescription.CacheClusterSize = gocf.String(api.stage.CacheClusterSize)
			}
			deployment := template.AddResource(apiDeploymentResName, apiDeployment)
			deployment.DependsOn = append(deployment.DependsOn, apiMethodCloudFormationResources...)
			deployment.DependsOn = append(deployment.DependsOn, apiGatewayResName)
		} else {
			newDeployment := &gocf.ApiGatewayDeployment{
				Description: gocf.String("Sparta deploy"),
				RestApiId:   apiGatewayRestAPIID.String(),
				StageName:   gocf.String(stageName),
			}
			// Use an unstable ID s.t. we can actually create a new deployment event.  Not sure how this
			// is going to work with deletes...
			deploymentResName := CloudFormationResourceName("APIGatewayDeployment")
			deployment := template.AddResource(deploymentResName, newDeployment)
			deployment.DependsOn = append(deployment.DependsOn, apiMethodCloudFormationResources...)
			deployment.DependsOn = append(deployment.DependsOn, apiGatewayResName)
		}

		template.Outputs[OutputAPIGatewayURL] = &gocf.Output{
			Description: "API Gateway URL",
			Value: gocf.Join("",
				gocf.String("https://"),
				apiGatewayRestAPIID,
				gocf.String(".execute-api."),
				gocf.Ref("AWS::Region"),
				gocf.String(".amazonaws.com/"),
				gocf.String(stageName)),
		}
	}
	return nil
}

// NewAPIGateway returns a new API Gateway structure.  If stage is defined, the API Gateway
// will also be deployed as part of stack creation.
func NewAPIGateway(name string, stage *Stage) *API {
	return &API{
		name:      name,
		stage:     stage,
		resources: make(map[string]*Resource, 0),
	}
}

// NewStage returns a Stage object with the given name.  Providing a Stage value
// to NewAPIGateway implies that the API Gateway resources should be deployed
// (eg: made publicly accessible).  See
// http://docs.aws.amazon.com/apigateway/latest/developerguide/how-to-deploy-api.html
func NewStage(name string) *Stage {
	return &Stage{
		name:      name,
		Variables: make(map[string]string, 0),
	}
}

// NewResource associates a URL path value with the LambdaAWSInfo golang lambda.  To make
// the Resource available, associate one or more Methods via NewMethod().
func (api *API) NewResource(pathPart string, parentLambda *LambdaAWSInfo) (*Resource, error) {
	_, exists := api.resources[pathPart]
	if exists {
		return nil, fmt.Errorf("Path %s already defined for lambda function: %s", pathPart, parentLambda.lambdaFnName)
	}
	resource := &Resource{
		pathPart:     pathPart,
		parentLambda: parentLambda,
		Methods:      make(map[string]*Method, 0),
	}
	api.resources[pathPart] = resource
	return resource, nil
}

// NewMethod associates the httpMethod name with the given Resource.  The returned Method
// has no authorization requirements.
func (resource *Resource) NewMethod(httpMethod string) (*Method, error) {
	authorizationType := "NONE"

	// http://docs.aws.amazon.com/apigateway/latest/developerguide/how-to-method-settings.html#how-to-method-settings-console
	keyname := httpMethod
	_, exists := resource.Methods[keyname]
	if exists {
		errMsg := fmt.Sprintf("Method %s (Auth: %s) already defined for resource", httpMethod, authorizationType)
		return nil, errors.New(errMsg)
	}
	integration := Integration{
		Parameters:       make(map[string]string, 0),
		RequestTemplates: make(map[string]string, 0),
		Responses:        make(map[int]*IntegrationResponse, 0),
		integrationType:  "AWS", // Type used for Lambda integration
	}

	method := &Method{
		authorizationType: authorizationType,
		httpMethod:        httpMethod,
		Parameters:        make(map[string]bool, 0),
		Models:            make(map[string]*Model, 0),
		Responses:         make(map[int]*Response, 0),
		Integration:       integration,
	}
	resource.Methods[keyname] = method
	return method, nil
}

// NewAuthorizedMethod associates the httpMethod name and authorizationType with the given Resource.
func (resource *Resource) NewAuthorizedMethod(httpMethod string, authorizationType string) (*Method, error) {
	method, err := resource.NewMethod(httpMethod)
	if nil != err {
		method.authorizationType = authorizationType
	}
	return method, err
}
