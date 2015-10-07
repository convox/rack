package controllers

import (
	"fmt"
	"math/rand"
	"runtime"
	"strings"

	"github.com/ddollar/logger"
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

func (e *Error) Error() string {
	return e.err.Error()
}

func (e *Error) Log(log *logger.Logger) {
	id := rand.Int31()

	log.Log("state=error id=%d message=%q", id, e.Error())

	for i, line := range e.trace {
		log.Log("state=error id=%d line=%d trace=%q", id, i, line)
	}
}

func (e *Error) System() bool {
	return e.errorType == ErrorTypeSystem
}

func errorTrace() []string {
	buffer := make([]byte, 1024*1024)
	size := runtime.Stack(buffer, false)

	trace := strings.Split(string(buffer[0:size]), "\n")

	// skip lines associated with error handler
	skipped := trace[ErrorHandlerSkipLines:]

	return skipped
}
