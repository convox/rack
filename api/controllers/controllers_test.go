package controllers_test

import (
	"bytes"

	"github.com/convox/logger"
	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/structs"
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

func Mock(fn func(*structs.MockProvider)) {
	p := controllers.Provider
	defer func() { controllers.Provider = p }()
	m := &structs.MockProvider{}
	controllers.Provider = m
	fn(m)
}
