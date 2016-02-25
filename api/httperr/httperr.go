package httperr

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/stvp/rollbar"
	"github.com/convox/rack/api/helpers"
)

const ErrorHandlerSkipLines = 7

type Error struct {
	code  int
	err   error
	stack rollbar.Stack
	trace []string
}

func New(code int, err error) *Error {
	if err == nil {
		return nil
	}

	e := &Error{
		code:  code,
		err:   err,
		stack: rollbar.BuildStack(3),
		trace: errorTrace(),
	}

	return e
}

func Server(err error) *Error {
	return New(500, err)
}

func Errorf(code int, format string, args ...interface{}) *Error {
	return New(code, fmt.Errorf(format, args...))
}

// Convenience function to track an internal server error in a controller handler
// See also helpers.TrackSuccess
func TrackServer(handler string, at string, err error) *Error {
	helpers.TrackEvent(
		fmt.Sprintf("api-%s-error", handler),
		map[string]interface{}{
			"at": at,
		},
	)

	return Server(err)
}

// Convenience function to track a user error (warning) in a controller handler
// See also helpers.TrackSuccess
func TrackErrorf(handler string, at string, code int, format string, args ...interface{}) *Error {
	helpers.TrackEvent(
		fmt.Sprintf("api-%s-warning", handler),
		map[string]interface{}{
			"at": at,
		},
	)

	return Errorf(code, format, args...)
}

func (e *Error) Code() int {
	return e.code
}

func (e *Error) Error() string {
	return e.err.Error()
}

func (e *Error) Save() error {
	rollbar.ErrorWithStack(rollbar.ERR, e.err, e.stack)
	return nil
}

func (e *Error) Trace() []string {
	return e.trace
}

func (e *Error) Server() bool {
	return e.code >= 500 && e.code < 600
}

func (e *Error) User() bool {
	return e.code >= 400 && e.code < 500
}

func errorTrace() []string {
	buffer := make([]byte, 1024*1024)
	size := runtime.Stack(buffer, false)

	trace := strings.Split(string(buffer[0:size]), "\n")

	// skip lines associated with error handler
	skipped := trace[ErrorHandlerSkipLines:]

	return skipped
}
