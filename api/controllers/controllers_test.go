package controllers_test

import (
	"bytes"

	"github.com/convox/logger"
	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/provider"
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

func Mock(fn func(*provider.MockProvider)) {
	p := controllers.Provider
	defer func() { controllers.Provider = p }()
	m := &provider.MockProvider{}
	controllers.Provider = m
	fn(m)
}
