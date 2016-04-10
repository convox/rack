package composure

import (
	"testing"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/convox/rack/composure/provider"
	"github.com/convox/rack/composure/structs"
)

func TestNil(t *testing.T) {
	assert.Nil(t, nil)
}

func TestPull(t *testing.T) {
	m, err := provider.CurrentProvider.Load("fixtures/httpd.yml")
	assert.Nil(t, err)
	assert.EqualValues(t, &structs.Manifest{}, m)
}
