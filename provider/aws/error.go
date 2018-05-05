package aws

import (
	"runtime"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/pkg/errors"
)

type withCode interface {
	Code() int
}

type errorWithCode struct {
	error
	code int
}

func (e errorWithCode) Code() int {
	return e.code
}

func errorNotFound(s string) error {
	return errorWithCode{code: 404, error: errors.New(s)}
}

type apiError struct {
	error
	trace errors.StackTrace
}

func (a apiError) Code() int {
	switch t := a.error.(type) {
	case withCode:
		return t.Code()
	case awserr.Error:
		return 500
	default:
		return 500
	}
}

func (a apiError) Error() string {
	switch t := a.error.(type) {
	case awserr.Error:
		return t.Message()
	default:
		return a.error.Error()
	}
}

func (a apiError) StackTrace() errors.StackTrace {
	return a.trace
}

func (p *AWSProvider) ApiError(err error) error {
	if err == nil {
		return nil
	}

	t := trace()

	if st, ok := err.(tracer); ok {
		t = st.StackTrace()
	}

	return apiError{error: err, trace: t}
}

type tracer interface {
	StackTrace() errors.StackTrace
}

func trace() errors.StackTrace {
	st := errors.StackTrace{}

	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	for _, f := range pcs[0:n] {
		st = append(st, errors.Frame(f))
	}
	return st
}
