// +build lambdabinary,!noop

package cgo

// #include <stdlib.h>
// #include <string.h>
import "C"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime/debug"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/mweagle/Sparta"
	spartaAWS "github.com/mweagle/Sparta/aws"
	"github.com/zcalusic/sysinfo"
)

// Lock to update CGO related config
var muCredentials sync.Mutex
var pythonCredentialsValue credentials.Value

////////////////////////////////////////////////////////////////////////////////
// lambdaFunctionErrResponse is the struct used to return a CGO error response
type lambdaFunctionErrResponse struct {
	Code    int         `json:"code"`
	Status  string      `json:"status"`
	Headers http.Header `json:"headers"`
	Error   string      `json:"error"`
}

////////////////////////////////////////////////////////////////////////////////
// cgoLambdaHTTPAdapterStruct is the binding between the various params
// supplied to the LambdaHandler
type cgoLambdaHTTPAdapterStruct struct {
	serviceName               string
	lambdaHTTPHandlerInstance *sparta.ServeMuxLambda
	logger                    *logrus.Logger
}

var cgoLambdaHTTPAdapter cgoLambdaHTTPAdapterStruct

////////////////////////////////////////////////////////////////////////////////
// cgoMain is the primary entrypoint for the library version
func cgoMain(callerFile string,
	serviceName string,
	serviceDescription string,
	lambdaAWSInfos []*sparta.LambdaAWSInfo,
	api *sparta.API,
	site *sparta.S3Site,
	workflowHooks *sparta.WorkflowHooks) error {

	// Add a panic handler
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Failed to initialize `cgo` library: %+v", r)
		}
	}()
	logger, loggerErr := sparta.NewLoggerWithFormatter("info", &logrus.JSONFormatter{})
	if nil != loggerErr {
		panic("Failed to initialize logger")
	}
	// Log the latest - duplicate of sparta.platformLogSysInfo
	var si sysinfo.SysInfo
	si.GetSysInfo()
	logger.WithFields(logrus.Fields{
		"systemInfo": si,
	}).Info("SystemInfo")

	// Startup the server
	cgoLambdaHTTPAdapter = cgoLambdaHTTPAdapterStruct{
		serviceName:               serviceName,
		lambdaHTTPHandlerInstance: sparta.NewServeMuxLambda(lambdaAWSInfos, logger),
		logger: logger,
	}
	return nil
}

func makeRequest(functionName string,
	eventBody io.ReadCloser,
	eventBodySize int64) ([]byte, http.Header, error) {

	// Add a panic handler
	defer func() {
		if r := recover(); r != nil {
			// Log the stack track
			stackTrace := debug.Stack()
			cgoLambdaHTTPAdapter.logger.WithFields(logrus.Fields{
				"Stack": string(stackTrace),
			}).Error("PANIC: Failed to handle request in `cgo` library: ", r)
		}
	}()

	// Create an http.Request object with this data...
	spartaResp := httptest.NewRecorder()
	spartaReq := &http.Request{
		Method: "POST",
		Header: map[string][]string{
			"Content-Type": []string{"application/json"},
		},
		URL: &url.URL{
			Scheme: "http",
			Path:   fmt.Sprintf("/%s", functionName),
		},
		Proto:            "HTTP/1.1",
		ProtoMajor:       1,
		ProtoMinor:       1,
		Body:             eventBody,
		ContentLength:    eventBodySize,
		TransferEncoding: make([]string, 0),
		Host:             "localhost",
	}

	cgoLambdaHTTPAdapter.lambdaHTTPHandlerInstance.ServeHTTP(spartaResp, spartaReq)

	// If there was an HTTP error, transform that into a stable
	// error payload and continue. This is the same format that's
	// used by the NodeJS proxying tier at /resources/index.js
	if spartaResp.Code >= 400 {
		errResponseBody := lambdaFunctionErrResponse{
			Code:    spartaResp.Code,
			Status:  http.StatusText(spartaResp.Code),
			Headers: spartaResp.Header(),
			Error:   spartaResp.Body.String(),
		}

		// Replace the response with a new one
		jsonBytes, jsonBytesErr := json.Marshal(errResponseBody)
		if nil != jsonBytesErr {
			return nil, nil, jsonBytesErr
		}
		errResponse := httptest.NewRecorder()
		errResponse.Write(jsonBytes)
		errResponse.Header().Set("content-length", strconv.Itoa(len(jsonBytes)))
		errResponse.Header().Set("content-type", "application/json")
		spartaResp = errResponse
	}
	return spartaResp.Body.Bytes(), spartaResp.HeaderMap, nil
}

func postMetrics(awsCredentials *credentials.Credentials,
	path string,
	responseBodyLength int,
	duration time.Duration) {

	awsCloudWatchService := cloudwatch.New(NewSession())
	metricNamespace := fmt.Sprintf("Sparta/%s", cgoLambdaHTTPAdapter.serviceName)
	lambdaFunctionName := os.Getenv("AWS_LAMBDA_FUNCTION_NAME")

	metricData := make([]*cloudwatch.MetricDatum, 0)
	sharedDimensions := make([]*cloudwatch.Dimension, 0)
	sharedDimensions = append(sharedDimensions,
		&cloudwatch.Dimension{
			Name:  aws.String("Path"),
			Value: aws.String(path),
		},
		&cloudwatch.Dimension{
			Name:  aws.String("Name"),
			Value: aws.String(lambdaFunctionName),
		})

	var sysinfo syscall.Sysinfo_t
	sysinfoErr := syscall.Sysinfo(&sysinfo)
	if nil == sysinfoErr {
		metricData = append(metricData, &cloudwatch.MetricDatum{
			MetricName: aws.String("Uptime"),
			Dimensions: sharedDimensions,
			Unit:       aws.String("Seconds"),
			Value:      aws.Float64(float64(sysinfo.Uptime)),
		})
	}
	metricData = append(metricData, &cloudwatch.MetricDatum{
		MetricName: aws.String("LambdaResponseLength"),
		Dimensions: sharedDimensions,
		Unit:       aws.String("Bytes"),
		Value:      aws.Float64(float64(responseBodyLength)),
	})
	params := &cloudwatch.PutMetricDataInput{
		MetricData: metricData,
		Namespace:  aws.String(metricNamespace),
	}
	awsCloudWatchService.PutMetricData(params)
}

// LambdaHandler is the public handler that's called by the transformed
// CGO compliant userinput. Users should not need to call this function
// directly
func LambdaHandler(functionName string,
	logLevel string,
	eventJSON string,
	awsCredentials *credentials.Credentials) ([]byte, http.Header, error) {
	startTime := time.Now()

	readableBody := bytes.NewReader([]byte(eventJSON))
	readbleBodyCloser := ioutil.NopCloser(readableBody)
	// Update the credentials
	muCredentials.Lock()
	value, valueErr := awsCredentials.Get()
	if nil != valueErr {
		muCredentials.Unlock()
		return nil, nil, valueErr
	}
	pythonCredentialsValue.AccessKeyID = value.AccessKeyID
	pythonCredentialsValue.SecretAccessKey = value.SecretAccessKey
	pythonCredentialsValue.SessionToken = value.SessionToken
	pythonCredentialsValue.ProviderName = "PythonCGO"
	muCredentials.Unlock()

	// Unpack the JSON request, turn it into a proto here and pass it
	// into the handler...

	// Update the credentials in the HTTP handler
	// in case we're ultimately forwarding to a custom
	// resource provider
	cgoLambdaHTTPAdapter.lambdaHTTPHandlerInstance.Credentials(pythonCredentialsValue)
	logrusLevel, logrusLevelErr := logrus.ParseLevel(logLevel)
	if logrusLevelErr == nil {
		cgoLambdaHTTPAdapter.logger.SetLevel(logrusLevel)
	}
	cgoLambdaHTTPAdapter.logger.WithFields(logrus.Fields{
		"Resource": functionName,
		"Request":  eventJSON,
	}).Debug("Making request")

	// Make the request...
	response, header, err := makeRequest(functionName, readbleBodyCloser, int64(len(eventJSON)))

	// TODO: Consider go routine
	postMetrics(awsCredentials, functionName, len(response), time.Since(startTime))

	cgoLambdaHTTPAdapter.logger.WithFields(logrus.Fields{
		"Header": header,
		"Error":  err,
	}).Debug("Request response")

	return response, header, err
}

// NewSession returns a CGO-aware AWS session that uses the Python
// credentials provided by the CGO interface.
func NewSession() *session.Session {
	muCredentials.Lock()
	defer muCredentials.Unlock()

	awsConfig := aws.
		NewConfig().
		WithCredentials(credentials.NewStaticCredentialsFromCreds(pythonCredentialsValue))
	return spartaAWS.NewSessionWithConfig(awsConfig, cgoLambdaHTTPAdapter.logger)
}
