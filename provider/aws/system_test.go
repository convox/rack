package aws_test

import (
	"fmt"
	"os"
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

	awsProvider, _ := aws.NewProvider(os.Getenv("us-east-1"), os.Getenv("fakeaccess"), os.Getenv("fakesecret"), os.Getenv("fakeendpoint"))

	err := awsProvider.SystemSave(sys)
	if assert.Error(t, err, "err should state wrong instance type") {
		assert.Equal(t, err, fmt.Errorf("invalid instance type: wrongtype"))
	}
}
