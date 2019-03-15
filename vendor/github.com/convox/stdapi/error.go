package stdapi

import "fmt"

type Error interface {
	Code() int
	Error() string
}

type apiError struct {
	error
	code int
}

func (a apiError) Code() int {
	return a.code
}

func (a apiError) Error() string {
	return a.error.Error()
}

func Errorf(code int, format string, args ...interface{}) error {
	return apiError{
		error: fmt.Errorf(format, args...),
		code:  code,
	}
}
