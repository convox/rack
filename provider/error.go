package provider

type errorNotFound interface {
	NotFound() bool
}

// ErrorNotFound returns true if the error is a "not found" type
func ErrorNotFound(err error) bool {
	if e, ok := err.(errorNotFound); ok && e.NotFound() {
		return true
	}
	return false
}
