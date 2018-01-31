package api

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/stvp/rollbar"
)

type Error struct {
	code  int
	err   error
	stack rollbar.Stack
	trace []string
}

func NewError(code int, err error) *Error {
	return &Error{
		code:  code,
		err:   err,
		stack: rollbar.BuildStack(3),
		trace: errorTrace(),
	}
}

func Errorf(code int, format string, a ...interface{}) *Error {
	return NewError(code, fmt.Errorf(format, a))
}

func ServerError(err error) *Error {
	return NewError(503, err)
}

func ServerErrorf(format string, a ...interface{}) *Error {
	return ServerError(fmt.Errorf(format, a...))
}

func (e *Error) Code() int {
	return e.code
}

func (e *Error) Error() string {
	return e.err.Error()
}

func (e *Error) Server() bool {
	return e.code/100 == 5
}

func (e *Error) User() bool {
	return e.code/100 == 4
}

func (e *Error) Record() {
	if RollbarToken != "" {
		rollbar.Token = RollbarToken
		rollbar.ErrorWithStack(rollbar.ERR, e.err, e.stack)
	}
}

func errorTrace() []string {
	buffer := make([]byte, 1024*1024)
	size := runtime.Stack(buffer, false)
	trace := strings.Split(string(buffer[0:size]), "\n")
	skipped := trace[7:]

	return skipped
}
