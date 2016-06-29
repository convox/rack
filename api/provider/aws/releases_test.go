package aws_test

import (
	"testing"
	"time"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/structs"

	"github.com/stretchr/testify/assert"
)

func TestReleaseGet(t *testing.T) {
	aws := StubAwsProvider(
		describeStacksCycle,

		release1GetItemCycle,
	)
	defer aws.Close()

	defer func() {
		//TODO: remove: as we arent updating all tests we need to set current provider back to a
		//clean default one (I miss rspec before)
		provider.CurrentProvider = new(provider.TestProviderRunner)
	}()

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
	aws := StubAwsProvider(
		describeStacksCycle,
		releasesQueryCycle,
	)
	defer aws.Close()

	defer func() {
		//TODO: remove: as we arent updating all tests we need to set current provider back to a
		//clean default one (I miss rspec before)
		provider.CurrentProvider = new(provider.TestProviderRunner)
	}()

	r, err := provider.ReleaseList("httpd")

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

func TestReleaseLatestEmpty(t *testing.T) {
	defer func() {
		provider.CurrentProvider = new(provider.TestProviderRunner)
	}()
	p := provider.CurrentProvider.(*provider.TestProviderRunner)

	p.Mock.On("ReleaseList", "myapp").Return(structs.Releases{}, nil)

	rs, err := p.ReleaseList("myapp")
	assert.Nil(t, err)
	assert.Equal(t, (*structs.Release)(nil), rs.Latest())
}

func TestReleaseLatest(t *testing.T) {
	defer func() {
		provider.CurrentProvider = new(provider.TestProviderRunner)
	}()
	p := provider.CurrentProvider.(*provider.TestProviderRunner)

	p.Mock.On("ReleaseList", "myapp").Return(structs.Releases{
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
	}, nil)

	rs, err := p.ReleaseList("myapp")
	assert.Nil(t, err)
	assert.Equal(t, &structs.Release{
		Id:       "RVFETUHHKKD",
		App:      "httpd",
		Build:    "BHINCLZYYVN",
		Env:      "foo=bar",
		Manifest: "web:\n  image: httpd\n  ports:\n  - 80:80\n",
		Created:  time.Unix(1459780542, 627770380).UTC(),
	}, rs.Latest())
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
