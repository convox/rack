package aws_test

type errorNotFound string

func (e errorNotFound) Error() string {
	return string(e)
}

func (e errorNotFound) NotFound() bool {
	return true
}
