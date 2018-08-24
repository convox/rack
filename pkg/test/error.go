package test

// ErrorNotFound means the requested item was not found
type ErrorNotFound string

// Error satisfies the error interface
func (e ErrorNotFound) Error() string {
	return string(e)
}

// NotFound defines the behavior of this error
func (e ErrorNotFound) NotFound() bool {
	return true
}
