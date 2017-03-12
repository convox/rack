package aws_test

import (
	"testing"

	"github.com/convox/rack/provider/aws"
	"github.com/stretchr/testify/assert"
)

// Test table for combinations of:
//
// http://bucket.s3.amazonaws.com
// http://bucket.s3.amazonaws.com/key
// http://bucket.s3-aws-region.amazonaws.com
// http://bucket.s3-aws-region.amazonaws.com/key
// http://s3.amazonaws.com/bucket
// http://s3.amazonaws.com/bucket/key
// http://s3-aws-region.amazonaws.com/bucket
// http://s3-aws-region.amazonaws.com/bucket/key
var parseS3UrlTests = []struct {
	url      string
	expected [3]string
}{
	// Note: Numbers on bucketX and keyX to ensure test results can't be faked (by accident)
	{"https://bucket1.s3.amazonaws.com", [3]string{"bucket1", "", ""}},
	{"https://bucket2.s3.amazonaws.com/key2", [3]string{"bucket2", "key2", ""}},
	{"https://bucket3.s3.amazonaws.com/some/key3", [3]string{"bucket3", "some/key3", ""}},
	{"https://bucket4.s3-us-west-1.amazonaws.com", [3]string{"bucket4", "", "us-west-1"}},
	{"https://bucket5.s3-us-west-2.amazonaws.com/key5", [3]string{"bucket5", "key5", "us-west-2"}},
	{"https://bucket6.s3-us-east-2.amazonaws.com/some/key6", [3]string{"bucket6", "some/key6", "us-east-2"}},
	{"https://s3.amazonaws.com/bucket7", [3]string{"bucket7", "", "us-east-1"}},
	{"https://s3.amazonaws.com/bucket8/key8", [3]string{"bucket8", "key8", "us-east-1"}},
	{"https://s3.amazonaws.com/bucket9/some/key9", [3]string{"bucket9", "some/key9", "us-east-1"}},
	{"https://s3-us-east-2.amazonaws.com/bucket10", [3]string{"bucket10", "", "us-east-2"}},
	{"https://s3-us-west-1.amazonaws.com/bucket11/key11", [3]string{"bucket11", "key11", "us-west-1"}},
	{"https://s3-us-west-1.amazonaws.com/bucket12/some/key12", [3]string{"bucket12", "some/key12", "us-west-1"}},

	// HTTP tests
	{"http://bucket13.s3.amazonaws.com/some/key13", [3]string{"bucket13", "some/key13", ""}},
	{"http://bucket14.s3-us-east-2.amazonaws.com/some/key14", [3]string{"bucket14", "some/key14", "us-east-2"}},
	{"http://s3.amazonaws.com/bucket15/some/key15", [3]string{"bucket15", "some/key15", "us-east-1"}},
	{"http://s3-us-west-1.amazonaws.com/bucket16/some/key16", [3]string{"bucket16", "some/key16", "us-west-1"}},
}

func TestParseS3Url(t *testing.T) {
	for _, tt := range parseS3UrlTests {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf(`Test for "%s" failed\n%s\n`, tt.url, r)
				}
			}()
			bucket, key, region, err := aws.ParseS3Url(tt.url)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected[0], bucket)
			assert.Equal(t, tt.expected[1], key)
			assert.Equal(t, tt.expected[2], region)
		}()
	}
}

func TestParseS3UrlError(t *testing.T) {
	bucket, key, region, err := aws.ParseS3Url("https://")
	assert.Equal(t, "", bucket)
	assert.Equal(t, "", key)
	assert.Equal(t, "", region)
	assert.Error(t, err)
}
