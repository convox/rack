package test

import (
	"net/http/httptest"
	"os"

	"github.com/convox/rack/api/awsutil"
	"github.com/convox/rack/api/config"
)

/*
Create a test server that mocks an Docker request/response cycle,
suitable for a single test

Example:
		s := StubDocker(ListContainersCycle())
		defer s.Close()

		d, _ := Docker(test.TestConfig.DockerHost)
		d.ListContainers(...)
*/
func StubDocker(cycles ...awsutil.Cycle) (s *httptest.Server) {
	handler := awsutil.NewHandler(cycles)
	s = httptest.NewServer(handler)

	config.TestConfig.DockerHost = s.URL
	os.Setenv("TEST_DOCKER_HOST", s.URL)

	return s
}

func ListECSContainersCycle() awsutil.Cycle {
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

func ListEmptyOneoffContainersCycle() awsutil.Cycle {
	return awsutil.Cycle{
		Request: awsutil.Request{
			RequestURI: "/containers/json?filters=%7B%22label%22%3A%5B%22com.convox.rack.type%3Doneoff%22%2C%22com.convox.rack.app%3Dmyapp-staging%22%5D%7D",
			Operation:  "",
			Body:       ``,
		},
		Response: awsutil.Response{
			StatusCode: 200,
			Body:       `[]`,
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
			Body:       statsResponse(),
		},
	}
}

func statsResponse() string {
	return `{
 "read" : "2015-01-08T22:57:31.547920715Z",
 "network" : {
    "rx_dropped" : 0,
    "rx_bytes" : 648,
    "rx_errors" : 0,
    "tx_packets" : 8,
    "tx_dropped" : 0,
    "rx_packets" : 8,
    "tx_errors" : 0,
    "tx_bytes" : 648
 },
 "memory_stats" : {
    "stats" : {
       "total_pgmajfault" : 0,
       "cache" : 0,
       "mapped_file" : 0,
       "total_inactive_file" : 0,
       "pgpgout" : 414,
       "rss" : 6537216,
       "total_mapped_file" : 0,
       "writeback" : 0,
       "unevictable" : 0,
       "pgpgin" : 477,
       "total_unevictable" : 0,
       "pgmajfault" : 0,
       "total_rss" : 6537216,
       "total_rss_huge" : 6291456,
       "total_writeback" : 0,
       "total_inactive_anon" : 0,
       "rss_huge" : 6291456,
       "hierarchical_memory_limit" : 67108864,
       "total_pgfault" : 964,
       "total_active_file" : 0,
       "active_anon" : 6537216,
       "total_active_anon" : 6537216,
       "total_pgpgout" : 414,
       "total_cache" : 0,
       "inactive_anon" : 0,
       "active_file" : 0,
       "pgfault" : 964,
       "inactive_file" : 0,
       "total_pgpgin" : 477
    },
    "max_usage" : 6651904,
    "usage" : 6537216,
    "failcnt" : 0,
    "limit" : 67108864
 },
 "blkio_stats" : {},
 "cpu_stats" : {
    "cpu_usage" : {
       "percpu_usage" : [
          16970827,
          1839451,
          7107380,
          10571290
       ],
       "usage_in_usermode" : 10000000,
       "total_usage" : 36488948,
       "usage_in_kernelmode" : 20000000
    },
    "system_cpu_usage" : 20091722000000000,
    "throttling_data" : {}
 }
}`
}
