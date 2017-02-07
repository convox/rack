package controllers_test

import (
	"bytes"

	"github.com/convox/logger"
)

type errorNotFound string

func (e errorNotFound) NotFound() bool {
	return true
}

func (e errorNotFound) Error() string {
	return string(e)
}

func init() {
	var buf bytes.Buffer
	logger.Output = &buf
}
