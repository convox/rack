package aws

import "fmt"

// errorNotFound means the requested item was not found
type errorNotFound string

// Error satisfies the error interface
func (e errorNotFound) Error() string {
	return string(e)
}

// NotFound defines the behavior of this error
func (e errorNotFound) NotFound() bool {
	return true
}

// ErrorNotFound returns true if the error is a "not found" type
func ErrorNotFound(err error) bool {
	if e, ok := err.(errorNotFound); ok && e.NotFound() {
		return true
	}
	return false
}

// NoSuchBuild means the requested build id was not found
type NoSuchBuild string

// Error satisfies the Error interface and formats the return error goven id
func (id NoSuchBuild) Error() string {
	return fmt.Sprintf("no such build: %s", string(id))
}
