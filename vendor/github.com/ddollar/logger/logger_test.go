package logger

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

var (
	buffer bytes.Buffer
)

func NewLogger() *Logger {
	buffer.Truncate(0)
	return NewWriter("ns=test", &buffer)
}

func TestNew(t *testing.T) {
	log := New("ns=test")
	assertEquals(t, log.namespace, "ns=test")
}

func TestAt(t *testing.T) {
	log := NewLogger()
	log.At("target").Log("foo=bar")
	assertLine(t, buffer.String(), `ns=test at=target foo=bar`)
}

func TestAtOverrides(t *testing.T) {
	log := NewLogger()
	log.At("target1").At("target2").Log("foo=bar")
	assertLine(t, buffer.String(), `ns=test at=target2 foo=bar`)
}

func TestError(t *testing.T) {
	log := NewLogger()
	log.Error(fmt.Errorf("broken"))

	lines := strings.Split(strings.TrimSpace(buffer.String()), "\n")

	assertMatch(t, lines[0], `ns=test state=error id=[0-9]+ message="broken"`)

	for i := 1; i < len(lines); i++ {
		assertMatch(t, lines[i], fmt.Sprintf(`ns=test state=error id=[0-9]+ line=%d trace="[^"]*"`, i))
	}
}

func TestLog(t *testing.T) {
	log := NewLogger()
	log.Log("string=%q int=%d float=%0.2f", "foo", 42, 3.14159)
	assertLine(t, buffer.String(), `ns=test string="foo" int=42 float=3.14`)
}

func TestNamespace(t *testing.T) {
	log := NewLogger()
	log.Namespace("foo=bar").Namespace("baz=qux").Log("fred=barney")
	assertLine(t, buffer.String(), `ns=test foo=bar baz=qux fred=barney`)
}

func TestReplace(t *testing.T) {
	log := NewLogger()
	log.Namespace("baz=qux1").Replace("baz", "qux2").Log("foo=bar")
	assertLine(t, buffer.String(), `ns=test baz=qux2 foo=bar`)
}

func TestReplaceExisting(t *testing.T) {
	log := NewLogger()
	log.Namespace("foo=bar").Namespace("baz=qux").Replace("baz", "zux").Log("thud=grunt")
	assertLine(t, buffer.String(), `ns=test foo=bar baz=zux thud=grunt`)
}

func TestStart(t *testing.T) {
	log := NewLogger()
	log.Start().Success("num=%d", 42)
	assertContains(t, buffer.String(), "elapsed=")
}

func TestStep(t *testing.T) {
	log := NewLogger()
	log.Step("target").Log("foo=bar")
	assertLine(t, buffer.String(), `ns=test step=target foo=bar`)
}

func TestStepOverrides(t *testing.T) {
	log := NewLogger()
	log.Step("target1").Step("target2").Log("foo=bar")
	assertLine(t, buffer.String(), `ns=test step=target2 foo=bar`)
}

func TestSuccess(t *testing.T) {
	log := NewLogger()
	log.Success("num=%d", 42)
	assertLine(t, buffer.String(), `ns=test state=success num=42`)
}

func assertContains(t *testing.T, got, search string) {
	if strings.Index(got, search) == -1 {
		t.Errorf("\n   expected: %q\n to contain: %q", got, search)
	}
}

func assertEquals(t *testing.T, got, search string) {
	if got != search {
		t.Errorf("\n   expected: %q\n to equal: %q", got, search)
	}
}

func assertLine(t *testing.T, got, search string) {
	search = search + "\n"
	if search != got {
		t.Errorf("\n   expected: %q\n to be: %q", got, search)
	}
}

func assertMatch(t *testing.T, got, search string) {
	r := regexp.MustCompile(search)

	if !r.MatchString(got) {
		t.Errorf("\n   expected: %q\n to match: %q", got, search)
	}
}
