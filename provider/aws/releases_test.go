package aws_test

import (
	"testing"
	"time"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/api/structs"

	"github.com/stretchr/testify/assert"
)

func TestReleaseGet(t *testing.T) {
	provider := StubAwsProvider(
		describeStacksCycle,

		release1GetItemCycle,
	)
	defer provider.Close()

	r, err := provider.ReleaseGet("httpd", "RVFETUHHKKD")

	assert.Nil(t, err)
	assert.EqualValues(t, &structs.Release{
		Id:       "RVFETUHHKKD",
		App:      "httpd",
		Build:    "BHINCLZYYVN",
		Env:      "foo=bar",
		Manifest: "web:\n  image: httpd\n  ports:\n  - 80:80\n",
		Created:  time.Unix(1459780542, 627770380).UTC(),
	}, r)
}

func TestReleaseList(t *testing.T) {
	provider := StubAwsProvider(
		describeStacksCycle,

		releasesQueryCycle,
	)
	defer provider.Close()

	r, err := provider.ReleaseList("httpd", 20)

	assert.Nil(t, err)

	assert.EqualValues(t, structs.Releases{
		structs.Release{
			Id:       "RVFETUHHKKD",
			App:      "httpd",
			Build:    "BHINCLZYYVN",
			Env:      "foo=bar",
			Manifest: "web:\n  image: httpd\n  ports:\n  - 80:80\n",
			Created:  time.Unix(1459780542, 627770380).UTC(),
		},
		structs.Release{
			Id:       "RFVZFLKVTYO",
			App:      "httpd",
			Build:    "BNOARQMVHUO",
			Env:      "foo=bar",
			Manifest: "web:\n  image: httpd\n  ports:\n  - 80:80\n",
			Created:  time.Unix(1459709199, 166694813).UTC(),
		},
	}, r)
}

var release1GetItemCycle = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.GetItem",
		Body:       `{"ConsistentRead":true,"Key":{"id":{"S":"RVFETUHHKKD"}},"TableName":"convox-releases"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{"Item":{"id":{"S":"RVFETUHHKKD"},"build":{"S":"BHINCLZYYVN"},"app":{"S":"httpd"},"manifest":{"S":"web:\n  image: httpd\n  ports:\n  - 80:80\n"},"env":{"S":"foo=bar"},"created":{"S":"20160404.143542.627770380"}}}`,
	},
}
