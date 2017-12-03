package explore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/mweagle/Sparta/proxy"
)

type mockAPIGatewayIdentity struct {
	AccountID                     string `json:"accountId"`
	APIKey                        string `json:"apiKey"`
	Caller                        string `json:"caller"`
	CognitoAuthenticationProvider string `json:"cognitoAuthenticationProvider"`
	CognitoAuthenticationType     string `json:"cognitoAuthenticationType"`
	CognitoIdentityID             string `json:"cognitoIdentityId"`
	CognitoIdentityPoolID         string `json:"cognitoIdentityPoolId"`
	SourceIP                      string `json:"sourceIp"`
	User                          string `json:"user"`
	UserAgent                     string `json:"userAgent"`
	UserArn                       string `json:"userArn"`
}

type mockAPIGatewayContext struct {
	AppID        string                 `json:"appId"`
	Method       string                 `json:"method"`
	RequestID    string                 `json:"requestId"`
	ResourceID   string                 `json:"resourceId"`
	ResourcePath string                 `json:"resourcePath"`
	Stage        string                 `json:"stage"`
	Identity     mockAPIGatewayIdentity `json:"identity"`
}

type mockAPIGatewayRequest struct {
	Method      string                `json:"method"`
	Body        interface{}           `json:"body"`
	Headers     map[string]string     `json:"headers"`
	QueryParams map[string]string     `json:"queryParams"`
	PathParams  map[string]string     `json:"pathParams"`
	Context     mockAPIGatewayContext `json:"context"`
}

// NewRawRequest mocks the NodeJS proxying tier by creating a JSON request that is POST'd to
// the golang lambda handler running on localhost.  Most clients should use NewLambdaRequest or
// NewAPIGatewayRequest to create mock data. This function is available for
// advanced test cases who need more control over the mock request.
func NewRawRequest(lambdaName string, context *proxy.AWSLambdaContext, eventData interface{}, testingURL string) (*http.Response, error) {
	// Marshal the event data to a byte stream
	marshaledData, marshaledDataErr := json.Marshal(eventData)
	if marshaledDataErr != nil {
		return nil, marshaledDataErr
	}
	proxyRequest := proxy.AWSProxyRequest{
		Context: context,
		Event:   marshaledData,
	}
	proxyBytes, proxyBytesErr := proto.Marshal(&proxyRequest)
	if proxyBytesErr != nil {
		return nil, proxyBytesErr
	}

	// POST IT...
	var host = fmt.Sprintf("%s/%s", testingURL, lambdaName)
	req, err := http.NewRequest("POST", host, bytes.NewReader(proxyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-protobuf")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// NewLambdaRequest sends a mock request to a localhost server that
// was created by httptest.NewServer(NewLambdaHTTPHandler(lambdaFunctions, logger)).
// lambdaName is the lambdaFnName to be called, eventData is optional event-specific
// data, and the testingURL is the URL returned by httptest.NewServer().
func NewLambdaRequest(lambdaName string, eventData interface{}, testingURL string) (*http.Response, error) {
	nowTime := time.Now()

	context := &proxy.AWSLambdaContext{
		AwsRequestId:       "12341234-1234-1234-1234-123412341234",
		LogGroupName:       "/aws/lambda/SpartaApplicationMockLogGroup-9ZX7FITHEAG8",
		LogStreamName:      fmt.Sprintf("%d/%d/%d/[$LATEST]%d", nowTime.Year(), nowTime.Month(), nowTime.Day(), nowTime.Unix()),
		FunctionName:       "SpartaFunction",
		MemoryLimitInMb:    "128",
		FunctionVersion:    "[LATEST]",
		InvokedFunctionArn: fmt.Sprintf("arn:aws:lambda:us-west-2:123412341234:function:SpartaMockFunction-%d", nowTime.Unix()),
	}

	return NewRawRequest(lambdaName, context, eventData, testingURL)
}

// NewAPIGatewayRequest sends a mock request to a localhost server that
// was created by httptest.NewServer(NewLambdaHTTPHandler(lambdaFunctions, logger)).
// lambdaName is the lambdaFnName to be called, eventData is optional event-specific
// data, and the testingURL is the URL returned by httptest.NewServer().  The optional event data is
// embedded in the Sparta input mapping templates.
func NewAPIGatewayRequest(lambdaName string, httpMethod string, whitelistParamValues map[string]string, eventData interface{}, testingURL string) (*http.Response, error) {

	mockAPIGatewayRequest := mockAPIGatewayRequest{
		Method:      httpMethod,
		Body:        eventData,
		Headers:     make(map[string]string, 0),
		QueryParams: make(map[string]string, 0),
		PathParams:  make(map[string]string, 0),
	}
	for eachWhitelistKey, eachWhitelistValue := range whitelistParamValues {
		// Whitelisted params include their
		// namespace as part of the whitelist expression:
		// method.request.querystring.keyName
		parts := strings.Split(eachWhitelistKey, ".")

		// The string should have 4 parts...
		if len(parts) != 4 {
			return nil, fmt.Errorf("Invalid whitelist param name: %s (MUST be: method.request.KEY_TYPE.KEY_NAME, ex: method.request.querystring.myQueryParam", eachWhitelistKey)
		}
		keyType := parts[2]
		keyName := parts[3]
		switch keyType {
		case "header":
			mockAPIGatewayRequest.Headers[keyName] = eachWhitelistValue
		case "querystring":
			mockAPIGatewayRequest.QueryParams[keyName] = eachWhitelistValue
		case "path":
			mockAPIGatewayRequest.PathParams[keyName] = eachWhitelistValue
		default:
			return nil, fmt.Errorf("Unsupported whitelist param type: %s", keyType)
		}
	}

	mockAPIGatewayRequest.Context.AppID = fmt.Sprintf("spartaApp%d", os.Getpid())
	mockAPIGatewayRequest.Context.Method = httpMethod
	mockAPIGatewayRequest.Context.RequestID = "12341234-1234-1234-1234-123412341234"
	mockAPIGatewayRequest.Context.ResourceID = "anon42"
	mockAPIGatewayRequest.Context.ResourcePath = "/mock"
	mockAPIGatewayRequest.Context.Stage = "mock"
	mockAPIGatewayRequest.Context.Identity = mockAPIGatewayIdentity{
		AccountID: "123412341234",
		APIKey:    "",
		Caller:    "",
		CognitoAuthenticationProvider: "",
		CognitoAuthenticationType:     "",
		CognitoIdentityID:             "",
		CognitoIdentityPoolID:         "",
		SourceIP:                      "127.0.0.1",
		User:                          "Unknown",
		UserAgent:                     "Mozilla/Gecko",
		UserArn:                       "",
	}
	return NewLambdaRequest(lambdaName, mockAPIGatewayRequest, testingURL)
}
