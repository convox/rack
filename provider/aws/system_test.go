package aws_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/provider/aws"
	"github.com/stretchr/testify/assert"
)

func TestSystemSaveWrongType(t *testing.T) {

	sys := structs.System{
		Name:    "name",
		Version: "version",
		Type:    "wrongtype",
	}

	provider := &aws.AWSProvider{}

	err := provider.SystemSave(sys)

	assert.Equal(t, err, fmt.Errorf("invalid instance type: wrongtype"))
}
