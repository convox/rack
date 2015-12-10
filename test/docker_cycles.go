package test

import "github.com/convox/rack/api/awsutil"

func ListContainersCycle() awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/containers/json?filters=%7B%22label%22%3A%5B%22com.amazonaws.ecs.task-arn%3Darn%3Aaws%3Aecs%3Aus-east-1%3A901416387788%3Atask%2F320a8b6a-c243-47d3-a1d1-6db5dfcb3f58%22%2C%22com.amazonaws.ecs.container-name%3Dworker%22%5D%7D",
			Operation:  "",
			Body:       ``,
		},
		Response: awsutil.Response{
			StatusCode: 200,
			Body:       `[{"Id": "8dfafdbc3a40","Command": "echo 1"}]`,
		},
	}
}

func StatsCycle() awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/containers/8dfafdbc3a40/stats?stream=false",
			Operation:  "",
			Body:       ``,
		},
		Response: awsutil.Response{
			StatusCode: 200,
			Body:       `{"read": "2015-01-08T22:57:31.547920715Z"}`,
		},
	}
}
