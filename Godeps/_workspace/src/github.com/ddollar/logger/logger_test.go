package logger

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

var (
	buffer bytes.Buffer
	log    = NewWriter("ns=test", &buffer)
)

func TestAt(t *testing.T) {
	buffer.Truncate(0)
	log.At("target").Log("foo=bar")
	assertContains(t, buffer.String(), `ns=test at=target foo=bar`)
}

func TestError(t *testing.T) {
	buffer.Truncate(0)
	log.Error(fmt.Errorf("broken"))
	assertContains(t, buffer.String(), `ns=test state=error error="broken"`)
}

func TestLog(t *testing.T) {
	buffer.Truncate(0)
	log.Log("string=%q int=%d float=%0.2f", "foo", 42, 3.14159)
	assertContains(t, buffer.String(), `ns=test string="foo" int=42 float=3.14`)
}

func TestNamespace(t *testing.T) {
	buffer.Truncate(0)
	log.Namespace("foo=bar").Namespace("baz=qux").Log("fred=barney")
	assertContains(t, buffer.String(), `ns=test foo=bar baz=qux fred=barney`)
}

func TestStart(t *testing.T) {
	buffer.Truncate(0)
	log.Start().Success("num=%d", 42)
	assertContains(t, buffer.String(), "elapsed=")
}

func TestSuccess(t *testing.T) {
	buffer.Truncate(0)
	log.Success("num=%d", 42)
	assertContains(t, buffer.String(), `ns=test state=success num=42`)
}

func assertContains(t *testing.T, got, search string) {
	if strings.Index(got, search) == -1 {
		t.Errorf("\n   expected: %q\n to contain: %q", got, search)
	}
}
