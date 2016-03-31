package aws_test

import (
	"os"
	"testing"
	"time"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/structs"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/assert"
)

func TestBuildGet(t *testing.T) {
	os.Setenv("DYNAMO_BUILDS", "test-builds")

	aws := test.StubAws(
		test.DescribeAppStackCycle("myapp"),
		GetItemAppBuildCycle("myapp"),
		GetItemAppBuildCycle("myapp"),
	)
	defer aws.Close()

	b, err := provider.BuildGet("myapp", "B123")

	assert.Nil(t, err)
	assert.EqualValues(t, &structs.Build{
		Id:       "B123",
		App:      "myapp",
		Manifest: "main:\n  image: httpd\n  ports:\n  - 80:80\n",
		Started:  time.Unix(1455916559, 194276614).UTC(),
		Ended:    time.Time{},
	}, b)
}

func GetItemAppBuildCycle(appName string) awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/",
			Operation:  "DynamoDB_20120810.GetItem",
			Body:       `{"ConsistentRead": true, "Key": {"id": { "S": "B123" } }, "TableName": "test-builds"}`,
		},
		Response: awsutil.Response{
			StatusCode: 200,
			Body: `{
    "Item": {
        "app": {
            "S": "` + appName + `"
        },
        "created": {
            "S": "20160219.211559.194276614"
        },
        "id": {
            "S": "B123"
        },
        "manifest": {
            "S": "main:\n  image: httpd\n  ports:\n  - 80:80\n"
        }
    }
}`,
		},
	}
}
