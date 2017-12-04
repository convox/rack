package main

import (
	"os"
	"testing"

	syslog "github.com/RackSec/srslog"
)

var PapertrailUrl = "tcp+tls://logs1.papertrailapp.com:11235"

func TestFormatter(t *testing.T) {
	os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "convox-syslog-6125")

	assertEqual(t,
		contentFormatter(syslog.LOG_INFO, "hostname", "tag", `testLogGroup 1440442987000 [ERROR] First test message`),
		"<22>1 2015-08-24T12:03:07-07:00 testLogGroup convox/syslog unknown - - [ERROR] First test message\n",
	)

	assertEqual(t,
		contentFormatter(syslog.LOG_INFO, "hostname", "tag", `convox-httpd-LogGroup-1KIJO8SS9F3Q9 1461030802652 web:RGBCKLEZHCX/aedfffead7ad 10.0.3.37 - - [19/Apr/2016:01:53:22 +0000] "GET / HTTP/1.1" 304 -`),
		"<22>1 2016-04-18T18:53:22-07:00 convox-httpd-LogGroup-1KIJO8SS9F3Q9 web:RGBCKLEZHCX aedfffead7ad - - 10.0.3.37 - - [19/Apr/2016:01:53:22 +0000] \"GET / HTTP/1.1\" 304 -\n",
	)
}

func assertEqual(t *testing.T, a, b string) {
	if a != b {
		t.Errorf("%q != %q", a, b)
	}
}
