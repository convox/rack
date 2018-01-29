package step

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	sparta "github.com/mweagle/Sparta"
)

func TestAWSStepFunction(t *testing.T) {
	// Normal Sparta lambda function
	lambdaFn := sparta.HandleAWSLambda(sparta.LambdaName(helloWorld),
		http.HandlerFunc(helloWorld),
		sparta.IAMRoleDefinition{})

	// // Create a Choice state
	lambdaTaskState := NewTaskState("lambdaHelloWorld", lambdaFn)
	delayState := NewWaitDelayState("holdUpNow", 3*time.Second)
	successState := NewSuccessState("success")

	// Hook them up..
	lambdaTaskState.Next(delayState)
	delayState.Next(successState)

	// Startup the machine.
	startMachine := NewStateMachine("SampleStepFunction", lambdaTaskState)

	// Add the state machine to the deployment...
	workflowHooks := &sparta.WorkflowHooks{
		ServiceDecorator: startMachine.StateMachineDecorator(),
	}

	// Test it...
	logger, _ := sparta.NewLogger("info")
	var templateWriter bytes.Buffer
	err := sparta.Provision(true,
		"SampleStepFunction",
		"",
		[]*sparta.LambdaAWSInfo{lambdaFn},
		nil,
		nil,
		os.Getenv("S3_BUCKET"),
		false,
		false,
		"testBuildID",
		"",
		"",
		"",
		&templateWriter,
		workflowHooks,
		logger)
	if nil != err {
		t.Fatal(err.Error())
	}
}

// Standard AWS λ function
func helloWorld(w http.ResponseWriter, r *http.Request) {
	logger, _ := r.Context().Value(sparta.ContextKeyLogger).(*logrus.Logger)
	logger.WithFields(logrus.Fields{
		"Woot": "Found",
	}).Warn("Lambda called")

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"hello" : "world"}`)
}
