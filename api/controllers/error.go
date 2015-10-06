package controllers

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/stvp/rollbar"
)

const (
	ErrorTypeSystem = iota
	ErrorTypeUser
)

const ErrorHandlerSkipLines = 7

type Error struct {
	err       error
	errorType int
	trace     []string
}

func (e *Error) Error() string {
	return e.err.Error()
}

func (e *Error) System() bool {
	return e.errorType == ErrorTypeSystem
}

func NewError(errorType int, err error) *Error {
	return &Error{
		err:       err,
		errorType: errorType,
		trace:     errorTrace(),
	}
}

func SystemError(err error) *Error {
	if err == nil {
		return nil
	}

	rollbar.ErrorWithStackSkip(rollbar.ERR, err, 1)

	return NewError(ErrorTypeSystem, err)
}

func UserErrorf(format string, args ...interface{}) *Error {
	return NewError(ErrorTypeUser, fmt.Errorf(format, args...))
}

func errorTrace() []string {
	buffer := make([]byte, 1024*1024)
	size := runtime.Stack(buffer, false)

	trace := strings.Split(string(buffer[0:size]), "\n")

	// skip lines associated with error handler
	skipped := trace[ErrorHandlerSkipLines:]

	return skipped
}
