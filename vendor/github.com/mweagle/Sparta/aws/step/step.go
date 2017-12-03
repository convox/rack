package step

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/mweagle/Sparta"
	spartaCF "github.com/mweagle/Sparta/aws/cloudformation"
	spartaIAM "github.com/mweagle/Sparta/aws/iam"
	gocf "github.com/mweagle/go-cloudformation"
)

// StateError is the reserved type used for AWS Step function error names
// Ref: https://states-language.net/spec.html#appendix-a
type StateError string

const (
	// StatesAll is a wild-card which matches any Error Name.
	StatesAll StateError = "States.ALL"
	// StatesTimeout is a Task State either ran longer than the
	// “TimeoutSeconds” value, or failed to heartbeat for a time
	// longer than the “HeartbeatSeconds” value.
	StatesTimeout StateError = "States.Timeout"
	// StatesTaskFailed is a Task State failed during the execution
	StatesTaskFailed StateError = "States.TaskFailed"
	// StatesPermissions is a Task State failed because it had
	// insufficient privileges to execute the specified code.
	StatesPermissions StateError = "States.Permissions"
	// StatesResultPathMatchFailure is a Task State’s “ResultPath” field
	// cannot be applied to the input the state received
	StatesResultPathMatchFailure StateError = "States.ResultPathMatchFailure"
	// StatesBranchFailed is a branch of a Parallel state failed
	StatesBranchFailed StateError = "States.BranchFailed"
	// StatesNoChoiceMatched is a Choice state failed to find a match for the
	// condition field extracted from its input
	StatesNoChoiceMatched StateError = "States.NoChoiceMatched"
)

/*******************************************************************************
   ___ ___  __  __ ___  _   ___ ___ ___  ___  _  _ ___
  / __/ _ \|  \/  | _ \/_\ | _ \_ _/ __|/ _ \| \| / __|
 | (_| (_) | |\/| |  _/ _ \|   /| |\__ \ (_) | .` \__ \
  \___\___/|_|  |_|_|/_/ \_\_|_\___|___/\___/|_|\_|___/

/******************************************************************************/

// Comparison is the generic comparison operator interface
type Comparison interface {
	json.Marshaler
}

// ChoiceBranch represents a type for a ChoiceState "Choices" entry
type ChoiceBranch interface {
	nextState() MachineState
}

////////////////////////////////////////////////////////////////////////////////
// StringEquals
////////////////////////////////////////////////////////////////////////////////

/**

Validations
	- JSONPath: https://github.com/NodePrime/jsonpath
	- Choices lead to existing states
	- Choice statenames are scoped to same depth
*/

// StringEquals comparison
type StringEquals struct {
	Comparison
	Variable string
	Value    string
}

// MarshalJSON for custom marshalling
func (cmp *StringEquals) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Variable     string
		StringEquals string
	}{
		Variable:     cmp.Variable,
		StringEquals: cmp.Value,
	})
}

////////////////////////////////////////////////////////////////////////////////
// StringLessThan
////////////////////////////////////////////////////////////////////////////////

// StringLessThan comparison
type StringLessThan struct {
	Comparison
	Variable string
	Value    string
}

// MarshalJSON for custom marshalling
func (cmp *StringLessThan) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Variable       string
		StringLessThan string
	}{
		Variable:       cmp.Variable,
		StringLessThan: cmp.Value,
	})
}

////////////////////////////////////////////////////////////////////////////////
// StringGreaterThan
////////////////////////////////////////////////////////////////////////////////

// StringGreaterThan comparison
type StringGreaterThan struct {
	Comparison
	Variable string
	Value    string
}

// MarshalJSON for custom marshalling
func (cmp *StringGreaterThan) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Variable          string
		StringGreaterThan string
	}{
		Variable:          cmp.Variable,
		StringGreaterThan: cmp.Value,
	})
}

////////////////////////////////////////////////////////////////////////////////
// StringLessThanEquals
////////////////////////////////////////////////////////////////////////////////

// StringLessThanEquals comparison
type StringLessThanEquals struct {
	Variable string
	Value    string
}

// MarshalJSON for custom marshalling
func (cmp *StringLessThanEquals) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Variable             string
		StringLessThanEquals string
	}{
		Variable:             cmp.Variable,
		StringLessThanEquals: cmp.Value,
	})
}

////////////////////////////////////////////////////////////////////////////////
// StringGreaterThanEquals
////////////////////////////////////////////////////////////////////////////////

// StringGreaterThanEquals comparison
type StringGreaterThanEquals struct {
	Comparison
	Variable string
	Value    string
}

// MarshalJSON for custom marshalling
func (cmp *StringGreaterThanEquals) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Variable                string
		StringGreaterThanEquals string
	}{
		Variable:                cmp.Variable,
		StringGreaterThanEquals: cmp.Value,
	})
}

////////////////////////////////////////////////////////////////////////////////
// NumericEquals
////////////////////////////////////////////////////////////////////////////////

// NumericEquals comparison
type NumericEquals struct {
	Comparison
	Variable string
	Value    int64
}

// MarshalJSON for custom marshalling
func (cmp *NumericEquals) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Variable      string
		NumericEquals int64
	}{
		Variable:      cmp.Variable,
		NumericEquals: cmp.Value,
	})
}

////////////////////////////////////////////////////////////////////////////////
// NumericLessThan
////////////////////////////////////////////////////////////////////////////////

// NumericLessThan comparison
type NumericLessThan struct {
	Comparison
	Variable string
	Value    int64
}

// MarshalJSON for custom marshalling
func (cmp *NumericLessThan) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Variable        string
		NumericLessThan int64
	}{
		Variable:        cmp.Variable,
		NumericLessThan: cmp.Value,
	})
}

////////////////////////////////////////////////////////////////////////////////
// NumericGreaterThan
////////////////////////////////////////////////////////////////////////////////

// NumericGreaterThan comparison
type NumericGreaterThan struct {
	Comparison
	Variable string
	Value    int64
}

// MarshalJSON for custom marshalling
func (cmp *NumericGreaterThan) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Variable           string
		NumericGreaterThan int64
	}{
		Variable:           cmp.Variable,
		NumericGreaterThan: cmp.Value,
	})
}

////////////////////////////////////////////////////////////////////////////////
// NumericLessThanEquals
////////////////////////////////////////////////////////////////////////////////

// NumericLessThanEquals comparison
type NumericLessThanEquals struct {
	Comparison
	Variable string
	Value    int64
}

// MarshalJSON for custom marshalling
func (cmp *NumericLessThanEquals) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Variable              string
		NumericLessThanEquals int64
	}{
		Variable:              cmp.Variable,
		NumericLessThanEquals: cmp.Value,
	})
}

////////////////////////////////////////////////////////////////////////////////
// NumericGreaterThanEquals
////////////////////////////////////////////////////////////////////////////////

// NumericGreaterThanEquals comparison
type NumericGreaterThanEquals struct {
	Comparison
	Variable string
	Value    int64
}

// MarshalJSON for custom marshalling
func (cmp *NumericGreaterThanEquals) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Variable                 string
		NumericGreaterThanEquals int64
	}{
		Variable:                 cmp.Variable,
		NumericGreaterThanEquals: cmp.Value,
	})
}

////////////////////////////////////////////////////////////////////////////////
// BooleanEquals
////////////////////////////////////////////////////////////////////////////////

// BooleanEquals comparison
type BooleanEquals struct {
	Comparison
	Variable string
	Value    interface{}
}

// MarshalJSON for custom marshalling
func (cmp *BooleanEquals) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Variable      string
		BooleanEquals interface{}
	}{
		Variable:      cmp.Variable,
		BooleanEquals: cmp.Value,
	})
}

////////////////////////////////////////////////////////////////////////////////
// TimestampEquals
////////////////////////////////////////////////////////////////////////////////

// TimestampEquals comparison
type TimestampEquals struct {
	Comparison
	Variable string
	Value    time.Time
}

// MarshalJSON for custom marshalling
func (cmp *TimestampEquals) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Variable        string
		TimestampEquals string
	}{
		Variable:        cmp.Variable,
		TimestampEquals: cmp.Value.Format(time.RFC3339),
	})
}

////////////////////////////////////////////////////////////////////////////////
// TimestampLessThan
////////////////////////////////////////////////////////////////////////////////

// TimestampLessThan comparison
type TimestampLessThan struct {
	Comparison
	Variable string
	Value    time.Time
}

// MarshalJSON for custom marshalling
func (cmp *TimestampLessThan) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Variable          string
		TimestampLessThan string
	}{
		Variable:          cmp.Variable,
		TimestampLessThan: cmp.Value.Format(time.RFC3339),
	})
}

////////////////////////////////////////////////////////////////////////////////
// TimestampGreaterThan
////////////////////////////////////////////////////////////////////////////////

// TimestampGreaterThan comparison
type TimestampGreaterThan struct {
	Variable string
	Value    time.Time
}

// MarshalJSON for custom marshalling
func (cmp *TimestampGreaterThan) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Variable             string
		TimestampGreaterThan string
	}{
		Variable:             cmp.Variable,
		TimestampGreaterThan: cmp.Value.Format(time.RFC3339),
	})
}

////////////////////////////////////////////////////////////////////////////////
// TimestampLessThanEquals
////////////////////////////////////////////////////////////////////////////////

// TimestampLessThanEquals comparison
type TimestampLessThanEquals struct {
	Comparison
	Variable string
	Value    time.Time
}

// MarshalJSON for custom marshalling
func (cmp *TimestampLessThanEquals) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Variable                string
		TimestampLessThanEquals string
	}{
		Variable:                cmp.Variable,
		TimestampLessThanEquals: cmp.Value.Format(time.RFC3339),
	})
}

////////////////////////////////////////////////////////////////////////////////
// TimestampGreaterThanEquals
////////////////////////////////////////////////////////////////////////////////

// TimestampGreaterThanEquals comparison
type TimestampGreaterThanEquals struct {
	Comparison
	Variable string
	Value    time.Time
}

// MarshalJSON for custom marshalling
func (cmp *TimestampGreaterThanEquals) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Variable                   string
		TimestampGreaterThanEquals string
	}{
		Variable:                   cmp.Variable,
		TimestampGreaterThanEquals: cmp.Value.Format(time.RFC3339),
	})
}

/*******************************************************************************
   ___  ___ ___ ___    _ _____ ___  ___  ___
  / _ \| _ \ __| _ \  /_\_   _/ _ \| _ \/ __|
 | (_) |  _/ _||   / / _ \| || (_) |   /\__ \
  \___/|_| |___|_|_\/_/ \_\_| \___/|_|_\|___/
/******************************************************************************/

////////////////////////////////////////////////////////////////////////////////
// And
////////////////////////////////////////////////////////////////////////////////

// And operator
type And struct {
	ChoiceBranch
	Comparison []Comparison
	Next       MachineState
}

func (andOperation *And) nextState() MachineState {
	return andOperation.Next
}

// MarshalJSON for custom marshalling
func (andOperation *And) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Comparison []Comparison `json:"And,omitempty"`
		Next       string       `json:",omitempty"`
	}{
		Comparison: andOperation.Comparison,
		Next:       andOperation.Next.Name(),
	})
}

////////////////////////////////////////////////////////////////////////////////
// Or
////////////////////////////////////////////////////////////////////////////////

// Or operator
type Or struct {
	ChoiceBranch
	Comparison []Comparison
	Next       MachineState
}

func (orOperation *Or) nextState() MachineState {
	return orOperation.Next
}

// MarshalJSON for custom marshalling
func (orOperation *Or) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Comparison []Comparison `json:"Or,omitempty"`
		Next       string       `json:",omitempty"`
	}{
		Comparison: orOperation.Comparison,
		Next:       orOperation.Next.Name(),
	})
}

////////////////////////////////////////////////////////////////////////////////
// Not
////////////////////////////////////////////////////////////////////////////////

// Not operator
type Not struct {
	ChoiceBranch
	Comparison Comparison
	Next       MachineState
}

func (notOperation *Not) nextState() MachineState {
	return notOperation.Next
}

// MarshalJSON for custom marshalling
func (notOperation *Not) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Not  Comparison
		Next string
	}{
		Not:  notOperation.Comparison,
		Next: notOperation.Next.Name(),
	})
}

// MachineState is the base state for all AWS Step function
type MachineState interface {
	Name() string
	nodeID() string
}

// TransitionState is the generic state according to
// https://states-language.net/spec.html#state-type-table
type TransitionState interface {
	MachineState
	Next(state MachineState) MachineState
	NextState() MachineState
	WithComment(string) TransitionState
	WithInputPath(string) TransitionState
	WithOutputPath(string) TransitionState
}

// Embedding struct for common properties
type baseInnerState struct {
	name       string
	id         int64
	next       MachineState
	comment    string
	inputPath  string
	outputPath string
}

func (bis *baseInnerState) nodeID() string {
	return fmt.Sprintf("%s-%d", bis.name, bis.id)
}

// marshalStateJSON for subclass marshalling of state information
func (bis *baseInnerState) marshalStateJSON(stateType string,
	additionalData map[string]interface{}) ([]byte, error) {
	if additionalData == nil {
		additionalData = make(map[string]interface{})
	}
	additionalData["Type"] = stateType
	if bis.next != nil {
		additionalData["Next"] = bis.next.Name()
	}
	if bis.comment != "" {
		additionalData["Comment"] = bis.comment
	}
	if bis.inputPath != "" {
		additionalData["InputPath"] = bis.inputPath
	}
	if bis.outputPath != "" {
		additionalData["OutputPath"] = bis.outputPath
	}
	return json.Marshal(additionalData)
}

/*******************************************************************************
 ___ _____ _ _____ ___ ___
/ __|_   _/_\_   _| __/ __|
\__ \ | |/ _ \| | | _|\__ \
|___/ |_/_/ \_\_| |___|___/
/******************************************************************************/

////////////////////////////////////////////////////////////////////////////////
// PassState
////////////////////////////////////////////////////////////////////////////////

// PassState represents a NOP state
type PassState struct {
	baseInnerState
	ResultPath string
	Result     interface{}
}

// WithResultPath is the fluent builder for the result path
func (ps *PassState) WithResultPath(resultPath string) *PassState {
	ps.ResultPath = resultPath
	return ps
}

// WithResult is the fluent builder for the result data
func (ps *PassState) WithResult(result interface{}) *PassState {
	ps.Result = result
	return ps
}

// Next returns the next state
func (ps *PassState) Next(nextState MachineState) MachineState {
	ps.next = nextState
	return ps
}

// NextState sets the next state
func (ps *PassState) NextState() MachineState {
	return ps.next
}

// Name returns the name of this Task state
func (ps *PassState) Name() string {
	return ps.name
}

// WithComment returns the TaskState comment
func (ps *PassState) WithComment(comment string) TransitionState {
	ps.comment = comment
	return ps
}

// WithInputPath returns the TaskState input data selector
func (ps *PassState) WithInputPath(inputPath string) TransitionState {
	ps.inputPath = inputPath
	return ps
}

// WithOutputPath returns the TaskState output data selector
func (ps *PassState) WithOutputPath(outputPath string) TransitionState {
	ps.outputPath = outputPath
	return ps
}

// MarshalJSON for custom marshalling
func (ps *PassState) MarshalJSON() ([]byte, error) {
	additionalParams := make(map[string]interface{})
	if ps.ResultPath != "" {
		additionalParams["ResultPath"] = ps.ResultPath
	}
	if ps.Result != nil {
		additionalParams["Result"] = ps.Result
	}
	return ps.marshalStateJSON("Pass", additionalParams)
}

// NewPassState returns a new PassState instance
func NewPassState(name string, resultData interface{}) *PassState {
	return &PassState{
		baseInnerState: baseInnerState{
			name: name,
			id:   rand.Int63(),
		},
		Result: resultData,
	}
}

////////////////////////////////////////////////////////////////////////////////
// ChoiceState
////////////////////////////////////////////////////////////////////////////////

// ChoiceState is a synthetic state that executes a lot of independent
// branches in parallel
type ChoiceState struct {
	baseInnerState
	Choices []ChoiceBranch
	Default TransitionState
}

// WithDefault is the fluent builder for the default state
func (cs *ChoiceState) WithDefault(defaultState TransitionState) *ChoiceState {
	cs.Default = defaultState
	return cs
}

// WithResultPath is the fluent builder for the result path
func (cs *ChoiceState) WithResultPath(resultPath string) *ChoiceState {
	return cs
}

// Name returns the name of this Task state
func (cs *ChoiceState) Name() string {
	return cs.name
}

// WithComment returns the TaskState comment
func (cs *ChoiceState) WithComment(comment string) *ChoiceState {
	cs.comment = comment
	return cs
}

// MarshalJSON for custom marshalling
func (cs *ChoiceState) MarshalJSON() ([]byte, error) {
	/*
		A state in a Parallel state branch “States” field MUST NOT have a “Next” field that targets a field outside of that “States” field. A state MUST NOT have a “Next” field which matches a state name inside a Parallel state branch’s “States” field unless it is also inside the same “States” field.

		Put another way, states in a branch’s “States” field can transition only to each other, and no state outside of that “States” field can transition into it.
	*/
	additionalParams := make(map[string]interface{})
	additionalParams["Choices"] = cs.Choices
	if cs.Default != nil {
		additionalParams["Default"] = cs.Default.Name()
	}
	return cs.marshalStateJSON("Choice", additionalParams)
}

// NewChoiceState returns a "ChoiceState" with the supplied
// information
func NewChoiceState(choiceStateName string, choices ...ChoiceBranch) *ChoiceState {
	return &ChoiceState{
		baseInnerState: baseInnerState{
			name: choiceStateName,
			id:   rand.Int63(),
		},
		Choices: append([]ChoiceBranch{}, choices...),
	}
}

////////////////////////////////////////////////////////////////////////////////
// TaskRetry
////////////////////////////////////////////////////////////////////////////////

// TaskRetry is an action to perform in response to a Task failure
type TaskRetry struct {
	ErrorEquals     []StateError  `json:",omitempty"`
	IntervalSeconds time.Duration `json:",omitempty"`
	MaxAttempts     int           `json:",omitempty"`
	BackoffRate     float32       `json:",omitempty"`
}

// WithErrors is the fluent builder
func (tr *TaskRetry) WithErrors(errors ...StateError) *TaskRetry {
	if tr.ErrorEquals == nil {
		tr.ErrorEquals = make([]StateError, 0)
	}
	tr.ErrorEquals = append(tr.ErrorEquals, errors...)
	return tr
}

// WithInterval is the fluent builder
func (tr *TaskRetry) WithInterval(interval time.Duration) *TaskRetry {
	tr.IntervalSeconds = interval
	return tr
}

// WithMaxAttempts is the fluent builder
func (tr *TaskRetry) WithMaxAttempts(maxAttempts int) *TaskRetry {
	tr.MaxAttempts = maxAttempts
	return tr
}

// WithBackoffRate is the fluent builder
func (tr *TaskRetry) WithBackoffRate(backoffRate float32) *TaskRetry {
	tr.BackoffRate = backoffRate
	return tr
}

// NewTaskRetry returns a new TaskRetry instance
func NewTaskRetry() *TaskRetry {
	return &TaskRetry{}
}

////////////////////////////////////////////////////////////////////////////////
// TaskCatch
////////////////////////////////////////////////////////////////////////////////

// TaskCatch is an action to handle a failing operation
type TaskCatch struct {
	/*
		The reserved name “States.ALL” appearing in a Retrier’s “ErrorEquals” field is a wild-card and matches any Error Name. Such a value MUST appear alone in the “ErrorEquals” array and MUST appear in the last Catcher in the “Catch” array.
	*/
	ErrorEquals []StateError    `json:",omitempty"`
	NextState   TransitionState `json:"Next,omitempty"`
}

// WithErrors is the fluent builder
func (tc *TaskCatch) WithErrors(errors ...StateError) *TaskCatch {
	if tc.ErrorEquals == nil {
		tc.ErrorEquals = make([]StateError, 0)
	}
	tc.ErrorEquals = append(tc.ErrorEquals, errors...)
	return tc
}

// Next is the fluent builder
func (tc *TaskCatch) Next(nextState TransitionState) *TaskCatch {
	tc.NextState = nextState
	return tc
}

// NewTaskCatch returns a new TaskCatch instance
func NewTaskCatch(errors ...StateError) *TaskCatch {
	return &TaskCatch{
		ErrorEquals: errors,
	}
}

////////////////////////////////////////////////////////////////////////////////
// TaskState
////////////////////////////////////////////////////////////////////////////////

// TaskState is the core state, responsible for delegating to a Lambda function
type TaskState struct {
	baseInnerState
	lambdaFn                  *sparta.LambdaAWSInfo
	lambdaLogicalResourceName string
	ResultPath                string
	TimeoutSeconds            time.Duration
	HeartbeatSeconds          time.Duration
	LambdaDecorator           sparta.TemplateDecorator
	preexistingDecorator      sparta.TemplateDecorator
	Retry                     []*TaskRetry
	Catch                     *TaskCatch
}

// NewTaskState returns a TaskState instance properly initialized
func NewTaskState(stateName string, lambdaFn *sparta.LambdaAWSInfo) *TaskState {
	ts := &TaskState{
		baseInnerState: baseInnerState{
			name: stateName,
			id:   rand.Int63(),
		},
		lambdaFn: lambdaFn,
	}
	ts.LambdaDecorator = func(serviceName string,
		lambdaResourceName string,
		lambdaResource gocf.LambdaFunction,
		resourceMetadata map[string]interface{},
		S3Bucket string,
		S3Key string,
		buildID string,
		cfTemplate *gocf.Template,
		context map[string]interface{},
		logger *logrus.Logger) error {
		if ts.preexistingDecorator != nil {
			preexistingLambdaDecoratorErr := ts.preexistingDecorator(
				serviceName,
				lambdaResourceName,
				lambdaResource,
				resourceMetadata,
				S3Bucket,
				S3Key,
				buildID,
				cfTemplate,
				context,
				logger)
			if preexistingLambdaDecoratorErr != nil {
				return preexistingLambdaDecoratorErr
			}
		}
		// Save the lambda name s.t. we can create the {"Ref"::"lambdaName"} entry...
		ts.lambdaLogicalResourceName = lambdaResourceName
		return nil
	}
	// If there already is a decorator, then save it...
	ts.preexistingDecorator = lambdaFn.Decorator
	ts.lambdaFn.Decorator = ts.LambdaDecorator
	return ts
}

// WithResultPath is the fluent builder for the result path
func (ts *TaskState) WithResultPath(resultPath string) *TaskState {
	ts.ResultPath = resultPath
	return ts
}

// WithTimeout is the fluent builder for TaskState
func (ts *TaskState) WithTimeout(timeout time.Duration) *TaskState {
	ts.TimeoutSeconds = timeout
	return ts
}

// WithHeartbeat is the fluent builder for TaskState
func (ts *TaskState) WithHeartbeat(pulse time.Duration) *TaskState {
	ts.HeartbeatSeconds = pulse
	return ts
}

// WithRetry is the fluent builder for TaskState
func (ts *TaskState) WithRetry(retries ...*TaskRetry) *TaskState {
	if ts.Retry == nil {
		ts.Retry = make([]*TaskRetry, 0)
	}
	ts.Retry = append(ts.Retry, retries...)
	return ts
}

// WithCatch is the fluent builder for TaskState
func (ts *TaskState) WithCatch(catch *TaskCatch) *TaskState {
	ts.Catch = catch
	return ts
}

// Next returns the next state
func (ts *TaskState) Next(nextState MachineState) MachineState {
	ts.next = nextState
	return nextState
}

// NextState sets the next state
func (ts *TaskState) NextState() MachineState {
	return ts.next
}

// Name returns the name of this Task state
func (ts *TaskState) Name() string {
	return ts.name
}

// WithComment returns the TaskState comment
func (ts *TaskState) WithComment(comment string) TransitionState {
	ts.comment = comment
	return ts
}

// WithInputPath returns the TaskState input data selector
func (ts *TaskState) WithInputPath(inputPath string) TransitionState {
	ts.inputPath = inputPath
	return ts
}

// WithOutputPath returns the TaskState output data selector
func (ts *TaskState) WithOutputPath(outputPath string) TransitionState {
	ts.outputPath = outputPath
	return ts
}

// MarshalJSON for custom marshalling
func (ts *TaskState) MarshalJSON() ([]byte, error) {
	additionalParams := make(map[string]interface{})
	additionalParams["Resource"] = fmt.Sprintf("{{%s}}", ts.lambdaLogicalResourceName)

	if ts.TimeoutSeconds.Seconds() != 0 {
		additionalParams["TimeoutSeconds"] = ts.TimeoutSeconds.Seconds()
	}
	if ts.HeartbeatSeconds.Seconds() != 0 {
		additionalParams["HeartbeatSeconds"] = ts.HeartbeatSeconds.Seconds()
	}
	if ts.ResultPath != "" {
		additionalParams["ResultPath"] = ts.ResultPath
	}
	if ts.TimeoutSeconds.Seconds() != 0 {
		additionalParams["TimeoutSeconds"] = ts.TimeoutSeconds.Seconds()
	}
	if ts.HeartbeatSeconds.Seconds() != 0 {
		additionalParams["HeartbeatSeconds"] = ts.HeartbeatSeconds.Seconds()
	}
	if len(ts.Retry) != 0 {
		additionalParams["Retry"] = ts.Retry
	}
	if ts.Catch != nil {
		additionalParams["Catch"] = ts.Catch
	}
	return ts.marshalStateJSON("Task", additionalParams)
}

////////////////////////////////////////////////////////////////////////////////
// WaitDelay
////////////////////////////////////////////////////////////////////////////////

// WaitDelay is a delay with an interval
type WaitDelay struct {
	baseInnerState
	delay time.Duration
}

// Name returns the WaitDelay name
func (wd *WaitDelay) Name() string {
	return wd.name
}

// Next sets the step after the wait delay
func (wd *WaitDelay) Next(nextState MachineState) MachineState {
	wd.next = nextState
	return wd
}

// NextState returns the next State
func (wd *WaitDelay) NextState() MachineState {
	return wd.next
}

// WithComment returns the WaitDelay comment
func (wd *WaitDelay) WithComment(comment string) TransitionState {
	wd.comment = comment
	return wd
}

// WithInputPath returns the TaskState input data selector
func (wd *WaitDelay) WithInputPath(inputPath string) TransitionState {
	wd.inputPath = inputPath
	return wd
}

// WithOutputPath returns the TaskState output data selector
func (wd *WaitDelay) WithOutputPath(outputPath string) TransitionState {
	wd.outputPath = outputPath
	return wd
}

// MarshalJSON for custom marshalling
func (wd *WaitDelay) MarshalJSON() ([]byte, error) {
	additionalParams := make(map[string]interface{})
	additionalParams["Seconds"] = wd.delay.Seconds()
	return wd.marshalStateJSON("Wait", additionalParams)
}

// NewWaitDelayState returns a new WaitDelay pointer instance
func NewWaitDelayState(stateName string, delayInSeconds time.Duration) *WaitDelay {
	return &WaitDelay{
		baseInnerState: baseInnerState{
			name: stateName,
			id:   rand.Int63(),
		},
		delay: delayInSeconds,
	}
}

////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// WaitUntil
////////////////////////////////////////////////////////////////////////////////

// WaitUntil is a delay with an absolute time gate
type WaitUntil struct {
	baseInnerState
	Timestamp time.Time
}

// Name returns the WaitDelay name
func (wu *WaitUntil) Name() string {
	return wu.name
}

// Next sets the step after the wait delay
func (wu *WaitUntil) Next(nextState MachineState) MachineState {
	wu.next = nextState
	return wu
}

// NextState returns the next State
func (wu *WaitUntil) NextState() MachineState {
	return wu.next
}

// WithComment returns the WaitDelay comment
func (wu *WaitUntil) WithComment(comment string) TransitionState {
	wu.comment = comment
	return wu
}

// WithInputPath returns the TaskState input data selector
func (wu *WaitUntil) WithInputPath(inputPath string) TransitionState {
	wu.inputPath = inputPath
	return wu
}

// WithOutputPath returns the TaskState output data selector
func (wu *WaitUntil) WithOutputPath(outputPath string) TransitionState {
	wu.outputPath = outputPath
	return wu
}

// MarshalJSON for custom marshalling
func (wu *WaitUntil) MarshalJSON() ([]byte, error) {
	additionalParams := make(map[string]interface{})
	additionalParams["Timestamp"] = wu.Timestamp.Format(time.RFC3339)
	return wu.marshalStateJSON("Wait", additionalParams)
}

// NewWaitUntilState returns a new WaitDelay pointer instance
func NewWaitUntilState(stateName string, waitUntil time.Time) *WaitUntil {
	return &WaitUntil{
		baseInnerState: baseInnerState{
			name: stateName,
			id:   rand.Int63(),
		},
		Timestamp: waitUntil,
	}
}

////////////////////////////////////////////////////////////////////////////////

// WaitDynamicUntil is a delay based on a previous response
type WaitDynamicUntil struct {
	baseInnerState
	TimestampPath string
}

// Name returns the WaitDelay name
func (wdu *WaitDynamicUntil) Name() string {
	return wdu.name
}

// Next sets the step after the wait delay
func (wdu *WaitDynamicUntil) Next(nextState MachineState) MachineState {
	wdu.next = nextState
	return wdu
}

// NextState returns the next State
func (wdu *WaitDynamicUntil) NextState() MachineState {
	return wdu.next
}

// WithComment returns the WaitDelay comment
func (wdu *WaitDynamicUntil) WithComment(comment string) TransitionState {
	wdu.comment = comment
	return wdu
}

// WithInputPath returns the TaskState input data selector
func (wdu *WaitDynamicUntil) WithInputPath(inputPath string) TransitionState {
	wdu.inputPath = inputPath
	return wdu
}

// WithOutputPath returns the TaskState output data selector
func (wdu *WaitDynamicUntil) WithOutputPath(outputPath string) TransitionState {
	wdu.outputPath = outputPath
	return wdu
}

// MarshalJSON for custom marshalling
func (wdu *WaitDynamicUntil) MarshalJSON() ([]byte, error) {
	additionalParams := make(map[string]interface{})
	additionalParams["TimestampPath"] = wdu.TimestampPath
	return wdu.marshalStateJSON("Wait", additionalParams)
}

// NewWaitDynamicUntilState returns a new WaitDynamicUntil pointer instance
func NewWaitDynamicUntilState(stateName string, timestampPath string) *WaitDynamicUntil {
	return &WaitDynamicUntil{
		baseInnerState: baseInnerState{
			name: stateName,
			id:   rand.Int63(),
		},
		TimestampPath: timestampPath,
	}
}

////////////////////////////////////////////////////////////////////////////////
// SuccessState
////////////////////////////////////////////////////////////////////////////////

// SuccessState represents the end of the state machine
type SuccessState struct {
	baseInnerState
}

// Name returns the WaitDelay name
func (ss *SuccessState) Name() string {
	return ss.name
}

// Next sets the step after the wait delay
func (ss *SuccessState) Next(nextState MachineState) MachineState {
	ss.next = nextState
	return ss
}

// NextState returns the next State
func (ss *SuccessState) NextState() MachineState {
	return ss.next
}

// WithComment returns the WaitDelay comment
func (ss *SuccessState) WithComment(comment string) TransitionState {
	ss.comment = comment
	return ss
}

// WithInputPath returns the TaskState input data selector
func (ss *SuccessState) WithInputPath(inputPath string) TransitionState {
	ss.inputPath = inputPath
	return ss
}

// WithOutputPath returns the TaskState output data selector
func (ss *SuccessState) WithOutputPath(outputPath string) TransitionState {
	ss.outputPath = outputPath
	return ss
}

// MarshalJSON for custom marshalling
func (ss *SuccessState) MarshalJSON() ([]byte, error) {
	return ss.marshalStateJSON("Succeed", nil)
}

// NewSuccessState returns a "SuccessState" with the supplied
// name
func NewSuccessState(name string) *SuccessState {
	return &SuccessState{
		baseInnerState: baseInnerState{
			name: name,
			id:   rand.Int63(),
		},
	}
}

////////////////////////////////////////////////////////////////////////////////

// FailState represents the end of state machine
type FailState struct {
	baseInnerState
	ErrorName string
	Cause     error
}

// Name returns the WaitDelay name
func (fs *FailState) Name() string {
	return fs.name
}

// Next sets the step after the wait delay
func (fs *FailState) Next(nextState MachineState) MachineState {
	return fs
}

// NextState returns the next State
func (fs *FailState) NextState() MachineState {
	return nil
}

// WithComment returns the WaitDelay comment
func (fs *FailState) WithComment(comment string) TransitionState {
	fs.comment = comment
	return fs
}

// WithInputPath returns the TaskState input data selector
func (fs *FailState) WithInputPath(inputPath string) TransitionState {
	return fs
}

// WithOutputPath returns the TaskState output data selector
func (fs *FailState) WithOutputPath(outputPath string) TransitionState {
	return fs
}

// MarshalJSON for custom marshaling
func (fs *FailState) MarshalJSON() ([]byte, error) {
	additionalParams := make(map[string]interface{})
	additionalParams["Error"] = fs.ErrorName
	if fs.Cause != nil {
		additionalParams["Cause"] = fs.Cause.Error()
	}
	return fs.marshalStateJSON("Fail", additionalParams)
}

// NewFailState returns a "FailState" with the supplied
// information
func NewFailState(failStateName string, errorName string, cause error) *FailState {
	return &FailState{
		baseInnerState: baseInnerState{
			name: failStateName,
			id:   rand.Int63(),
		},
		ErrorName: errorName,
		Cause:     cause,
	}
}

////////////////////////////////////////////////////////////////////////////////
// ParallelState
////////////////////////////////////////////////////////////////////////////////

// ParallelState is a synthetic state that executes a lot of independent
// branches in parallel
type ParallelState struct {
	baseInnerState
	States     StateMachine
	ResultPath string
	Retry      []*TaskRetry
	Catch      *TaskCatch
}

// WithResultPath is the fluent builder for the result path
func (ps *ParallelState) WithResultPath(resultPath string) *ParallelState {
	ps.ResultPath = resultPath
	return ps
}

// WithRetry is the fluent builder for TaskState
func (ps *ParallelState) WithRetry(retries ...*TaskRetry) *ParallelState {
	if ps.Retry == nil {
		ps.Retry = make([]*TaskRetry, 0)
	}
	ps.Retry = append(ps.Retry, retries...)
	return ps
}

// WithCatch is the fluent builder for TaskState
func (ps *ParallelState) WithCatch(catch *TaskCatch) *ParallelState {
	ps.Catch = catch
	return ps
}

// Next returns the next state
func (ps *ParallelState) Next(nextState MachineState) MachineState {
	ps.next = nextState
	return nextState
}

// NextState sets the next state
func (ps *ParallelState) NextState() MachineState {
	return ps.next
}

// Name returns the name of this Task state
func (ps *ParallelState) Name() string {
	return ps.name
}

// WithComment returns the TaskState comment
func (ps *ParallelState) WithComment(comment string) TransitionState {
	ps.comment = comment
	return ps
}

// WithInputPath returns the TaskState input data selector
func (ps *ParallelState) WithInputPath(inputPath string) TransitionState {
	ps.inputPath = inputPath
	return ps
}

// WithOutputPath returns the TaskState output data selector
func (ps *ParallelState) WithOutputPath(outputPath string) TransitionState {
	ps.outputPath = outputPath
	return ps
}

// MarshalJSON for custom marshalling
func (ps *ParallelState) MarshalJSON() ([]byte, error) {
	/*
		A state in a Parallel state branch “States” field MUST NOT have a “Next” field that targets a field outside of that “States” field. A state MUST NOT have a “Next” field which matches a state name inside a Parallel state branch’s “States” field unless it is also inside the same “States” field.

		Put another way, states in a branch’s “States” field can transition only to each other, and no state outside of that “States” field can transition into it.
	*/
	additionalParams := make(map[string]interface{})
	if ps.ResultPath != "" {
		additionalParams["ResultPath"] = ps.ResultPath
	}
	if len(ps.Retry) != 0 {
		additionalParams["Retry"] = ps.Retry
	}
	if ps.Catch != nil {
		additionalParams["Catch"] = ps.Catch
	}
	return ps.marshalStateJSON("Parallel", additionalParams)
}

// NewParallelState returns a "ParallelState" with the supplied
// information
func NewParallelState(parallelStateName string, states StateMachine) *ParallelState {
	return &ParallelState{
		baseInnerState: baseInnerState{
			name: parallelStateName,
			id:   rand.Int63(),
		},
		States: states,
	}
}

////////////////////////////////////////////////////////////////////////////////
// StateMachine
////////////////////////////////////////////////////////////////////////////////

// StateMachine is the top level item
type StateMachine struct {
	name                 string
	comment              string
	stateDefinitionError error
	startAt              TransitionState
	uniqueStates         map[string]MachineState
}

//Comment sets the StateMachine comment
func (sm *StateMachine) Comment(comment string) *StateMachine {
	sm.comment = comment
	return sm
}

// validate performs any validation against the state machine
// prior to marshaling
func (sm *StateMachine) validate() []error {
	validationErrors := make([]error, 0)
	if sm.stateDefinitionError != nil {
		validationErrors = append(validationErrors, sm.stateDefinitionError)
	}
	return validationErrors
}

// StateMachineDecorator is the hook exposed by the StateMachine
// to insert the AWS Step function into the CloudFormation template
func (sm *StateMachine) StateMachineDecorator() sparta.ServiceDecoratorHook {
	return func(context map[string]interface{},
		serviceName string,
		template *gocf.Template,
		S3Bucket string,
		buildID string,
		awsSession *session.Session,
		noop bool,
		logger *logrus.Logger) error {

		machineErrors := sm.validate()
		if len(machineErrors) != 0 {
			errorText := make([]string, len(machineErrors))
			for index := range machineErrors {
				errorText[index] = machineErrors[index].Error()
			}
			return fmt.Errorf("Invalid state machine. Errors: %s",
				strings.Join(errorText, ", "))
		}

		lambdaFunctionResourceNames := []string{}
		for _, eachState := range sm.uniqueStates {
			taskState, taskStateOk := eachState.(*TaskState)
			if taskStateOk {
				lambdaFunctionResourceNames = append(lambdaFunctionResourceNames,
					taskState.lambdaLogicalResourceName)
			}
		}

		// Assume policy document
		regionalPrincipal := gocf.Join(".",
			gocf.String("states"),
			gocf.Ref("AWS::Region"),
			gocf.String("amazonaws.com"))
		var AssumePolicyDocument = sparta.ArbitraryJSONObject{
			"Version": "2012-10-17",
			"Statement": []sparta.ArbitraryJSONObject{
				{
					"Effect": "Allow",
					"Principal": sparta.ArbitraryJSONObject{
						"Service": regionalPrincipal,
					},
					"Action": []string{"sts:AssumeRole"},
				},
			},
		}
		statesIAMRole := &gocf.IAMRole{
			AssumeRolePolicyDocument: AssumePolicyDocument,
		}
		if len(lambdaFunctionResourceNames) != 0 {
			statements := make([]spartaIAM.PolicyStatement, 0)
			for _, eachLambdaName := range lambdaFunctionResourceNames {
				statements = append(statements,
					spartaIAM.PolicyStatement{
						Effect: "Allow",
						Action: []string{
							"lambda:InvokeFunction",
						},
						Resource: gocf.GetAtt(eachLambdaName, "Arn").String(),
					},
				)
			}
			iamPolicies := gocf.IAMRolePolicyList{}
			iamPolicies = append(iamPolicies, gocf.IAMRolePolicy{
				PolicyDocument: sparta.ArbitraryJSONObject{
					"Version":   "2012-10-17",
					"Statement": statements,
				},
				PolicyName: gocf.String("StatesExecutionPolicy"),
			})
			statesIAMRole.Policies = &iamPolicies
		}
		iamRoleResource := sparta.CloudFormationResourceName("StatesIAMRole",
			"StatesIAMRole")
		template.AddResource(iamRoleResource, statesIAMRole)

		// Sweet - serialize it without indentation so that the
		// ConvertToTemplateExpression can actually parse the inline `Ref` objects
		jsonBytes, jsonBytesErr := json.Marshal(sm)
		if jsonBytesErr != nil {
			return fmt.Errorf("Failed to marshal: %s", jsonBytesErr.Error())
		}
		logger.WithFields(logrus.Fields{
			"StateMachine": string(jsonBytes),
		}).Debug("State machine definition")

		// Great, so we've serialized the "Resource", but we actually
		// need to replace each lambda "Resource" definition with a
		// properly quoted Fn::GetAtt. Not sure how to make this as part of
		// the MarshalJSON, since it's invalid JSON :(
		stateMachineString := string(jsonBytes)
		for _, eachLambdaResourceName := range lambdaFunctionResourceNames {
			// Look for the reserved pattern that was exported in MarshalJSON
			reReplace := regexp.MustCompile(fmt.Sprintf(`"\{\{%s\}\}"`, eachLambdaResourceName))
			// Create the replacement text that quotes the GetAtt call
			replaceText := fmt.Sprintf(`"{"Fn::GetAtt": ["%s","Arn"]}"`, eachLambdaResourceName)
			stateMachineString = reReplace.ReplaceAllString(stateMachineString, replaceText)
		}

		// Super, now parse this into an Fn::Join representation
		// so that we can get inline expansion of the AWS pseudo params
		smReader := bytes.NewReader([]byte(stateMachineString))
		templateExpr, templateExprErr := spartaCF.ConvertToTemplateExpression(smReader, nil)
		if nil != templateExprErr {
			return fmt.Errorf("Failed to parser: %s", templateExprErr.Error())
		}

		// Awsome - add an AWS::StepFunction to the template with this info and roll with it...
		stepFunctionResource := &gocf.StepFunctionsStateMachine{
			StateMachineName: gocf.String(sm.name),
			DefinitionString: templateExpr,
			RoleArn:          gocf.GetAtt(iamRoleResource, "Arn").String(),
		}
		stepFunctionResourceName := sparta.CloudFormationResourceName("StateMachine",
			"StateMachine")
		template.AddResource(stepFunctionResourceName, stepFunctionResource)
		return nil
	}
}

// MarshalJSON for custom marshalling
func (sm *StateMachine) MarshalJSON() ([]byte, error) {

	// If there aren't any states, then it's the end
	return json.Marshal(&struct {
		Comment string                  `json:",omitempty"`
		StartAt string                  `json:",omitempty"`
		States  map[string]MachineState `json:",omitempty"`
		End     bool                    `json:",omitempty"`
	}{
		Comment: sm.comment,
		StartAt: sm.startAt.Name(),
		States:  sm.uniqueStates,
		End:     len(sm.uniqueStates) == 1,
	})
}

// NewStateMachine returns a new StateMachine instance
func NewStateMachine(stateMachineName string, startState TransitionState) *StateMachine {
	uniqueStates := make(map[string]MachineState, 0)
	pendingStates := []MachineState{startState}
	duplicateStateNames := make(map[string]bool, 0)

	nodeVisited := func(node MachineState) bool {
		if node == nil {
			return true
		}
		_, visited := uniqueStates[node.Name()]
		return visited
	}

	for len(pendingStates) != 0 {
		headState, tailStates := pendingStates[0], pendingStates[1:]
		uniqueStates[headState.Name()] = headState

		switch stateNode := headState.(type) {
		case *ChoiceState:
			for _, eachChoice := range stateNode.Choices {
				if !nodeVisited(eachChoice.nextState()) {
					tailStates = append(tailStates, eachChoice.nextState())
				}
			}
			if !nodeVisited(stateNode.Default) {
				tailStates = append(tailStates, stateNode.Default)
			}
		case TransitionState:
			if !nodeVisited(stateNode.NextState()) {
				tailStates = append(tailStates, stateNode.NextState())
			}
		}
		pendingStates = tailStates
	}

	// Walk all the states and assemble them into the states slice
	sm := &StateMachine{
		name:         stateMachineName,
		startAt:      startState,
		uniqueStates: uniqueStates,
	}
	// Store duplicate state names
	if len(duplicateStateNames) != 0 {
		sm.stateDefinitionError = fmt.Errorf("Duplicate state names: %#v", duplicateStateNames)
	}
	return sm
}

////////////////////////////////////////////////////////////////////////////////
