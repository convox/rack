package aws_test

import "github.com/convox/rack/api/awsutil"

var cycleReleaseGetItem = awsutil.Cycle{
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

var cycleReleaseGetItemNotFound = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.GetItem",
		Body:       `{"ConsistentRead":true,"Key":{"id":{"S":"RVFETUHHKKD"}},"TableName":"convox-releases"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{}`,
	},
}

var cycleReleaseGetItemBadManifest = awsutil.Cycle{
	Request: awsutil.Request{
		RequestURI: "/",
		Operation:  "DynamoDB_20120810.GetItem",
		Body:       `{"ConsistentRead":true,"Key":{"id":{"S":"RVFETUHHKKD"}},"TableName":"convox-releases"}`,
	},
	Response: awsutil.Response{
		StatusCode: 200,
		Body:       `{"Item":{"id":{"S":"RVFETUHHKKD"},"build":{"S":"BHINCLZYYVN"},"app":{"S":"httpd"},"manifest":{"S":"!!garbage/::"},"env":{"S":"foo=bar"},"created":{"S":"20160404.143542.627770380"}}}`,
	},
}
