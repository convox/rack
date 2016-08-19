package aws

type ErrorNotFound string

func (e ErrorNotFound) Error() string {
	return string(e)
}

func (e ErrorNotFound) NotFound() bool {
	return true
}
