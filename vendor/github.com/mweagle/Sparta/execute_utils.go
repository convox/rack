package sparta

import (
	"bytes"
	"context"
	"encoding/json"
	"expvar"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/pprof"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/mweagle/Sparta/proxy"
	"github.com/mweagle/cloudformationresources"
)

// Dispatch map for user defined CloudFormation CustomResources to
// lambda functions
type dispatchMap map[string]*LambdaAWSInfo

// Dispatch map for normal AWS Lambda to user defined Sparta lambda functions
type customResourceDispatchMap map[string]*customResourceInfo

var pprofDispatchMap = map[string]http.HandlerFunc{}

func init() {
	pprofDispatchMap["/debug/pprof"] = http.HandlerFunc(pprof.Index)
	pprofDispatchMap["/debug/pprof/cmdline"] = http.HandlerFunc(pprof.Cmdline)
	pprofDispatchMap["/debug/pprof/profile"] = http.HandlerFunc(pprof.Profile)
	pprofDispatchMap["/debug/pprof/symbol"] = http.HandlerFunc(pprof.Symbol)
	pprofDispatchMap["/debug/pprof/trace"] = http.HandlerFunc(pprof.Trace)
}

// This is a copy of the expvarHandler implementation from
// https://golang.org/src/expvar/expvar.go
func expvarHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, "{\n")
	first := true
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(w, "\n}\n")
}

func userDefinedCustomResourceForwarder(customResource *customResourceInfo,
	event *json.RawMessage,
	context *LambdaContext,
	w http.ResponseWriter,
	logger *logrus.Logger) {

	var rawProps map[string]interface{}
	json.Unmarshal([]byte(*event), &rawProps)

	var lambdaEvent cloudformationresources.CloudFormationLambdaEvent
	jsonErr := json.Unmarshal([]byte(*event), &lambdaEvent)
	if jsonErr != nil {
		logger.WithFields(logrus.Fields{
			"RawEvent":       rawProps,
			"UnmarshalError": jsonErr,
		}).Warn("Raw event data")
		http.Error(w, jsonErr.Error(), http.StatusInternalServerError)
	}

	logger.WithFields(logrus.Fields{
		"LambdaEvent": lambdaEvent,
	}).Debug("CloudFormation user resource lambda event")

	// Create the new request and send it off
	customResourceRequest := &cloudformationresources.UserFuncResourceRequest{}
	customResourceRequest.LambdaHandler = func(requestType string,
		stackID string,
		properties map[string]interface{},
		logger *logrus.Logger) (map[string]interface{}, error) {

		//  Descend to get the "UserProperties" field iff defined by the customResource
		var userProperties map[string]interface{}
		if _, exists := lambdaEvent.ResourceProperties["UserProperties"]; exists {
			childProps, ok := lambdaEvent.ResourceProperties["UserProperties"].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("Failed to extract UserProperties from payload")
			}
			userProperties = childProps
		}
		return customResource.userFunction(requestType, stackID, userProperties, logger)
	}
	customResourceRequest.RequestType = lambdaEvent.RequestType
	customResourceRequest.ResponseURL = lambdaEvent.ResponseURL
	customResourceRequest.StackID = lambdaEvent.StackID
	customResourceRequest.RequestID = lambdaEvent.RequestID
	customResourceRequest.LogicalResourceID = lambdaEvent.LogicalResourceID
	customResourceRequest.PhysicalResourceID = lambdaEvent.PhysicalResourceID
	customResourceRequest.LogGroupName = context.LogGroupName
	customResourceRequest.LogStreamName = context.LogStreamName
	customResourceRequest.ResourceProperties = lambdaEvent.ResourceProperties
	if "" == customResourceRequest.PhysicalResourceID {
		customResourceRequest.PhysicalResourceID = fmt.Sprintf("LogStreamName: %s", context.LogStreamName)
	}
	requestErr := cloudformationresources.Run(customResourceRequest, logger)
	if requestErr != nil {
		http.Error(w, requestErr.Error(), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, "CustomResource handled: "+lambdaEvent.LogicalResourceID)
	}
}

// Extract the fields and forward the event to the resource
func spartaCustomResourceForwarder(creds credentials.Value,
	event *json.RawMessage,
	context *LambdaContext,
	w http.ResponseWriter,
	logger *logrus.Logger) {

	var rawProps map[string]interface{}
	json.Unmarshal([]byte(*event), &rawProps)

	var lambdaEvent cloudformationresources.CloudFormationLambdaEvent
	jsonErr := json.Unmarshal([]byte(*event), &lambdaEvent)
	if jsonErr != nil {
		logger.WithFields(logrus.Fields{
			"RawEvent":       rawProps,
			"UnmarshalError": jsonErr,
		}).Warn("Raw event data")
		http.Error(w, jsonErr.Error(), http.StatusInternalServerError)
	}

	logger.WithFields(logrus.Fields{
		"LambdaEvent": lambdaEvent,
	}).Debug("CloudFormation Lambda event")

	// Setup the request and send it off
	customResourceRequest := &cloudformationresources.CustomResourceRequest{}
	customResourceRequest.RequestType = lambdaEvent.RequestType
	customResourceRequest.ResponseURL = lambdaEvent.ResponseURL
	customResourceRequest.StackID = lambdaEvent.StackID
	customResourceRequest.RequestID = lambdaEvent.RequestID
	customResourceRequest.LogicalResourceID = lambdaEvent.LogicalResourceID
	customResourceRequest.PhysicalResourceID = lambdaEvent.PhysicalResourceID
	customResourceRequest.LogGroupName = context.LogGroupName
	customResourceRequest.LogStreamName = context.LogStreamName
	customResourceRequest.ResourceProperties = lambdaEvent.ResourceProperties
	if "" == customResourceRequest.PhysicalResourceID {
		customResourceRequest.PhysicalResourceID = fmt.Sprintf("LogStreamName: %s", context.LogStreamName)
	}

	requestErr := cloudformationresources.Handle(customResourceRequest, creds, logger)
	if requestErr != nil {
		http.Error(w, requestErr.Error(), http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, "CustomResource handled: "+lambdaEvent.LogicalResourceID)
	}
}

// ServeMuxLambda is an HTTP compliant handler that implements
// ServeHTTP
type ServeMuxLambda struct {
	LambdaDispatchMap         dispatchMap
	muValue                   sync.Mutex
	customCreds               credentials.Value
	customResourceDispatchMap customResourceDispatchMap
	logger                    *logrus.Logger
}

// Credentials allows the user to supply a custom Credentials value
// object for any internal calls
func (handler *ServeMuxLambda) Credentials(creds credentials.Value) {
	handler.muValue.Lock()
	defer handler.muValue.Unlock()
	handler.customCreds = creds
}

func (handler *ServeMuxLambda) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Handle panics
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("%v", r)
			}
			stackTrace := debug.Stack()
			handler.logger.WithFields(logrus.Fields{
				"Stack": string(stackTrace),
			}).Error("PANIC")
			errorString := fmt.Sprintf("Lambda handler panic: %#v", err)
			http.Error(w, errorString, http.StatusBadRequest)
		}
	}()

	// If this is the expvar handler then skip it
	if "/golang/expvar" == req.URL.Path {
		expvarHandler(w, req)
		return
	}
	pprofHandler := pprofDispatchMap[req.URL.Path]
	if pprofHandler != nil {
		pprofHandler(w, req)
		return
	}

	// Read the request and unmarshal the right version
	defer req.Body.Close()
	bodyData, bodyDataErr := ioutil.ReadAll(req.Body)
	if bodyDataErr != nil {
		errorString := fmt.Sprintf("Failed to read proxy request: %s",
			bodyDataErr.Error())
		http.Error(w, errorString, http.StatusBadRequest)
		return
	}

	var proxyRequest proxy.AWSProxyRequest
	var unmarshalErr error
	switch req.Header.Get("Content-Type") {
	case "application/x-protobuf":
		unmarshalErr = proto.Unmarshal(bodyData, &proxyRequest)
	case "application/json":
		unmarshalErr = jsonpb.Unmarshal(bytes.NewReader(bodyData), &proxyRequest)
	default:
		unmarshalErr = fmt.Errorf("Unsupported proxy request Content-Type: %s",
			req.Header.Get("Content-Type"))
	}
	if unmarshalErr != nil {
		http.Error(w, unmarshalErr.Error(), http.StatusBadRequest)
		return
	}
	// Remove the leading slash and dispatch it to the golang handler
	lambdaFunc := strings.TrimLeft(req.URL.Path, "/")
	handler.logger.WithFields(logrus.Fields{
		"LookupName":   lambdaFunc,
		"ProtoMessage": true,
	}).Debug("Dispatching")

	proxyContext := proxyRequest.GetContext()
	request := lambdaRequest{
		Context: LambdaContext{
			AWSRequestID:       proxyContext.GetAwsRequestId(),
			LogGroupName:       proxyContext.GetLogGroupName(),
			LogStreamName:      proxyContext.GetLogStreamName(),
			FunctionName:       proxyContext.GetFunctionName(),
			MemoryLimitInMB:    proxyContext.GetMemoryLimitInMb(),
			FunctionVersion:    proxyContext.GetFunctionVersion(),
			InvokedFunctionARN: proxyContext.GetInvokedFunctionArn(),
		},
		Event: proxyRequest.GetEvent(),
	}
	lambdaAWSInfo := handler.LambdaDispatchMap[lambdaFunc]
	if nil != lambdaAWSInfo {
		if lambdaAWSInfo.httpHandler != nil {

			// Setup the request
			reqBody := bytes.NewReader(proxyRequest.GetEvent())
			spartaReq := httptest.NewRequest(req.Method,
				req.URL.Path,
				reqBody)
			spartaReqContext := context.WithValue(req.Context(),
				ContextKeyLogger,
				handler.logger)
			spartaReqContext = context.WithValue(spartaReqContext,
				ContextKeyLambdaContext,
				&request.Context)

			// Call the normal HTTP handler
			lambdaAWSInfo.httpHandler.ServeHTTP(w, spartaReq.WithContext(spartaReqContext))
		} else {
			lambdaAWSInfo.lambdaFn(&request.Event, &request.Context, w, handler.logger)
		}
	} else if strings.Contains(lambdaFunc, "::") {
		// Not the most exhaustive guard, but the CloudFormation custom resources
		// all have "::" delimiters in their type field.  Even if there is a false
		// positive, the spartaCustomResourceForwarder will simply error out.
		spartaCustomResourceForwarder(handler.customCreds,
			&request.Event,
			&request.Context,
			w,
			handler.logger)
	} else {
		// Final check for user-defined resource
		customResource, exists := handler.customResourceDispatchMap[lambdaFunc]
		handler.logger.WithFields(logrus.Fields{
			"Request":    request,
			"LookupName": lambdaFunc,
			"Exists":     exists,
		}).Debug("Custom Resource request")
		if exists {
			userDefinedCustomResourceForwarder(customResource,
				&request.Event,
				&request.Context,
				w,
				handler.logger)
		} else {
			http.Error(w, "Unsupported path: "+lambdaFunc, http.StatusBadRequest)
		}
	}
}

// NewServeMuxLambda returns an initialized ServeMuxLambda instance.  The returned value
// can be provided to https://golang.org/pkg/net/http/httptest/#NewServer to perform
// localhost testing.
func NewServeMuxLambda(lambdaAWSInfos []*LambdaAWSInfo,
	logger *logrus.Logger) *ServeMuxLambda {
	lookupMap := make(dispatchMap)
	customResourceMap := make(customResourceDispatchMap)
	for _, eachLambdaInfo := range lambdaAWSInfos {
		logger.WithFields(logrus.Fields{
			"Path": eachLambdaInfo.lambdaFunctionName(),
		}).Debug("Registering lambda URL")

		lookupMap[eachLambdaInfo.lambdaFunctionName()] = eachLambdaInfo
		// Build up the customResourceDispatchMap
		for _, eachCustomResource := range eachLambdaInfo.customResources {
			logger.WithFields(logrus.Fields{
				"Path": eachCustomResource.userFunctionName,
			}).Debug("Registering customResource URL")
			customResourceMap[eachCustomResource.userFunctionName] = eachCustomResource
		}
	}

	return &ServeMuxLambda{
		LambdaDispatchMap:         lookupMap,
		customResourceDispatchMap: customResourceMap,
		logger: logger,
	}
}
