package aws_test

import (
	"errors"
	"github.com/convox/rack/provider/aws"
	"github.com/stretchr/testify/assert"
	"testing"
)

type errorNotFound string

func (e errorNotFound) Error() string {
	return string(e)
}

func (e errorNotFound) NotFound() bool {
	return true
}

func TestNoSuchBuild_Error(t *testing.T) {
	err := aws.NoSuchBuild("B12345")
	if assert.Error(t, err) {
		assert.Equal(t, errors.New("no such build: B12345").Error(), err.Error())
	}
}
