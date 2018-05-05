package aws_test

import (
	"testing"

	"github.com/convox/rack/provider/aws"
	"github.com/stretchr/testify/assert"
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
	assert.EqualError(t, err, "build not found: B12345")
}
