package controllers

import (
	"fmt"
	"math/rand"
	"runtime"
	"strings"

	"github.com/ddollar/logger"
	"github.com/stvp/rollbar"
)

const ErrorHandlerSkipLines = 7

type HttpError struct {
	code  int
	err   error
	trace []string
}

func NewHttpError(code int, err error) *HttpError {
	if err != nil {
		return nil
	}

	e := &HttpError{
		code:  code,
		err:   err,
		trace: errorTrace(),
	}

	if e.ServerError() {
		rollbar.ErrorWithStackSkip(rollbar.ERR, err, 1)
	}

	return e
}

func ServerError(err error) *HttpError {
	return NewHttpError(500, err)
}

func HttpErrorf(code int, format string, args ...interface{}) *HttpError {
	return NewHttpError(code, fmt.Errorf(format, args...))
}

func (e *HttpError) Error() string {
	return e.err.Error()
}

func (e *HttpError) Log(log *logger.Logger) {
	if e.UserError() {
		log.Log("state=error type=user message=%q", e.Error())
		return
	}

	id := rand.Int31()

	log.Log("state=error id=%d message=%q", id, e.Error())

	for i, line := range e.trace {
		log.Log("state=error id=%d line=%d trace=%q", id, i, line)
	}
}

func (e *HttpError) ServerError() bool {
	return e.code >= 500 && e.code < 600
}

func (e *HttpError) UserError() bool {
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
