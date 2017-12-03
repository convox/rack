package sparta

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/mweagle/Sparta/explore"
)

func exploreTestHelloWorld(w http.ResponseWriter, r *http.Request) {
	logger, _ := r.Context().Value(ContextKeyLogger).(*logrus.Logger)

	event, _ := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	logger.Info("Hello World: ", string(event))
	w.Header().Add("Content-Type", "application/json")
	w.Write(event)
}

func exploreTestHelloWorldHTTP(w http.ResponseWriter, r *http.Request) {
	lambdaContext, lambdaContextOk := r.Context().Value(ContextKeyLambdaContext).(*LambdaContext)
	logger, _ := r.Context().Value(ContextKeyLogger).(*logrus.Logger)
	logger.WithFields(logrus.Fields{
		"Context":   lambdaContext,
		"ContextOk": lambdaContextOk,
	}).Warn("Checking context")

	fmt.Printf("Hello World 🌍")
	fmt.Fprint(w, "Done!")
}

func TestExplore(t *testing.T) {
	// Create the function to test
	var lambdaFunctions []*LambdaAWSInfo
	lambdaFn := HandleAWSLambda(LambdaName(exploreTestHelloWorld),
		http.HandlerFunc(exploreTestHelloWorld),
		IAMRoleDefinition{})
	lambdaFunctions = append(lambdaFunctions, lambdaFn)

	// Mock event specific data to send to the lambda function
	eventData := ArbitraryJSONObject{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3"}

	// Make the request and confirm
	logger, _ := NewLogger("warning")
	ts := httptest.NewServer(NewServeMuxLambda(lambdaFunctions, logger))
	defer ts.Close()
	resp, err := explore.NewLambdaRequest(lambdaFn.URLPath(), eventData, ts.URL)
	if err != nil {
		t.Fatal(err.Error())
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	t.Log("Status: ", resp.Status)
	t.Log("Headers: ", resp.Header)
	t.Log("Body: ", string(body))
}

func TestExploreAPIGateway(t *testing.T) {
	// Create the function to test
	var lambdaFunctions []*LambdaAWSInfo
	lambdaFn := HandleAWSLambda(LambdaName(exploreTestHelloWorld),
		http.HandlerFunc(exploreTestHelloWorld),
		IAMRoleDefinition{})
	lambdaFunctions = append(lambdaFunctions, lambdaFn)

	// Mock event specific data to send to the lambda function
	eventData := ArbitraryJSONObject{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3"}

	// Make the request and confirm
	logger, _ := NewLogger("warning")
	ts := httptest.NewServer(NewServeMuxLambda(lambdaFunctions, logger))
	defer ts.Close()
	var emptyWhitelist map[string]string
	resp, err := explore.NewAPIGatewayRequest(lambdaFn.URLPath(),
		"GET",
		emptyWhitelist,
		eventData,
		ts.URL)

	if err != nil {
		t.Fatal(err.Error())
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	t.Log("Status: ", resp.Status)
	t.Log("Headers: ", resp.Header)
	t.Log("Body: ", string(body))
}

func TestExploreUserName(t *testing.T) {
	// Create the function to test
	var lambdaFunctions []*LambdaAWSInfo
	lambdaFn := HandleAWSLambda(LambdaName(exploreTestHelloWorldHTTP),
		http.HandlerFunc(exploreTestHelloWorldHTTP),
		IAMRoleDefinition{})
	lambdaFunctions = append(lambdaFunctions, lambdaFn)

	// Mock event specific data to send to the lambda function
	eventData := ArbitraryJSONObject{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3"}

	// Make the request and confirm
	logger, _ := NewLogger("warning")
	ts := httptest.NewServer(NewServeMuxLambda(lambdaFunctions, logger))
	defer ts.Close()
	resp, err := explore.NewLambdaRequest(lambdaFn.URLPath(), eventData, ts.URL)
	if err != nil {
		t.Fatal(err.Error())
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	t.Log("Status: ", resp.Status)
	t.Log("Headers: ", resp.Header)
	t.Log("Body: ", string(body))
}

func TestNewAPIGatewayRequest(t *testing.T) {
	// Create the function to test
	var lambdaFunctions []*LambdaAWSInfo
	lambdaFn := HandleAWSLambda(LambdaName(exploreTestHelloWorld),
		http.HandlerFunc(exploreTestHelloWorld),
		IAMRoleDefinition{})
	lambdaFunctions = append(lambdaFunctions, lambdaFn)

	// Mock event specific data to send to the lambda function
	eventData := ArbitraryJSONObject{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3"}

	// Make the request and confirm
	logger, _ := NewLogger("warning")
	ts := httptest.NewServer(NewServeMuxLambda(lambdaFunctions, logger))
	defer ts.Close()
	var emptyWhitelist map[string]string
	resp, err := explore.NewAPIGatewayRequest(lambdaFn.URLPath(),
		"GET",
		emptyWhitelist,
		eventData,
		ts.URL)

	if err != nil {
		t.Fatal(err.Error())
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	// test unmarshalling mock to APIGatewayLambdaJSONEvent
	var testlambdaevent APIGatewayLambdaJSONEvent
	err = json.Unmarshal(body, &testlambdaevent)
	if err != nil {
		t.Fatal(err.Error())
	}

	t.Log("Method:", testlambdaevent.Method)
	t.Log("Body:", string(testlambdaevent.Body))
	t.Log("Headers:", testlambdaevent.Headers)
	t.Log("QueryParams:", testlambdaevent.QueryParams)
	t.Log("PathParams:", testlambdaevent.PathParams)
	t.Log("Context:", testlambdaevent.Context)
}
