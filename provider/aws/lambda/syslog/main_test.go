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
		"<22>1 2015-08-24T19:03:07Z testhost service/web 00000 - - [ERROR] First test message\n",
		contentFormatter("testhost")(syslog.LOG_INFO, "hostname", "tag", `service/web/00000 1440442987000 [ERROR] First test message`),
	)
}

func assertEqual(t *testing.T, a, b string) {
	if a != b {
		t.Errorf("%q != %q", a, b)
	}
}
