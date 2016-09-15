package aws

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
