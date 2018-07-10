package logger

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/pkg/errors"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

var Discard = NewWriter("", ioutil.Discard)

type Logger struct {
	namespace string
	now       func() string
	started   time.Time
	writer    io.Writer
}

var Output io.Writer = nil

func New(ns string) *Logger {
	return NewWriter(ns, os.Stdout)
}

func NewWriter(ns string, writer io.Writer) *Logger {
	return &Logger{namespace: ns, writer: writer}
}

func (l *Logger) At(at string) *Logger {
	return l.Replace("at", at)
}

func (l *Logger) Step(step string) *Logger {
	return l.Replace("step", step)
}

func (l *Logger) Append(format string, args ...interface{}) *Logger {
	return &Logger{
		namespace: fmt.Sprintf("%s %s", l.namespace, fmt.Sprintf(format, args...)),
		started:   l.started,
		writer:    l.writer,
	}
}

func (l *Logger) Prepend(format string, args ...interface{}) *Logger {
	return &Logger{
		namespace: fmt.Sprintf("%s %s", fmt.Sprintf(format, args...), l.namespace),
		started:   l.started,
		writer:    l.writer,
	}
}

type tracer interface {
	StackTrace() errors.StackTrace
}

func (l *Logger) Error(err error) error {
	if err == nil {
		return nil
	}

	message := strings.Replace(err.Error(), "\n", " ", -1)

	switch t := err.(type) {
	case tracer:
		file := strings.Split(fmt.Sprintf("%+s", t.StackTrace()[0]), "\n\t")[1]
		if f, err := stripGoPath(file); err == nil {
			file = f
		}
		l.Logf("error=%q location=%q", message, fmt.Sprintf("%s:%d", file, t.StackTrace()[0]))
	case error:
		l.Logf("error=%q", message)
	}

	return err
}

func (l *Logger) Errorf(format string, args ...interface{}) error {
	return l.Error(fmt.Errorf(format, args...))
}

func (l *Logger) ErrorBacktrace(err error) {
	id := rand.Int31()

	l.Logf("state=error id=%d message=%q", id, err)

	stack := debug.Stack()
	scanner := bufio.NewScanner(bytes.NewReader(stack))
	line := 1

	for scanner.Scan() {
		l.Logf("state=error id=%d line=%d trace=%q", id, line, scanner.Text())
		line += 1
	}
}

func (l *Logger) Logf(format string, args ...interface{}) {
	if l.started.IsZero() {
		l.Writer().Write([]byte(fmt.Sprintf("%s %s\n", l.namespace, fmt.Sprintf(format, args...))))
	} else {
		elapsed := float64(time.Now().Sub(l.started).Nanoseconds()) / 1000000
		l.Writer().Write([]byte(fmt.Sprintf("%s %s elapsed=%0.3f\n", l.namespace, fmt.Sprintf(format, args...), elapsed)))
	}
}

func (l *Logger) Namespace(format string, args ...interface{}) *Logger {
	return &Logger{
		namespace: fmt.Sprintf("%s %s", l.namespace, fmt.Sprintf(format, args...)),
		started:   l.started,
		writer:    l.writer,
	}
}

func (l *Logger) Replace(key, value string) *Logger {
	pair := fmt.Sprintf("%s=%s", key, value)

	r := regexp.MustCompile(fmt.Sprintf(`\s%s=\S+`, key))

	if r.MatchString(l.namespace) {
		return &Logger{
			namespace: r.ReplaceAllString(l.namespace, " "+pair),
			started:   l.started,
			writer:    l.writer,
		}
	} else {
		return l.Namespace(fmt.Sprintf("%s=%s", key, value))
	}
}

func (l *Logger) Start() *Logger {
	return &Logger{
		namespace: l.namespace,
		started:   time.Now(),
		writer:    l.writer,
	}
}

func (l *Logger) Success() error {
	l.Logf("state=success")
	return nil
}

func (l *Logger) Successf(format string, args ...interface{}) error {
	l.Logf("state=success %s", fmt.Sprintf(format, args...))
	return nil
}

func (l *Logger) Writer() io.Writer {
	if Output != nil {
		return Output
	}

	return l.writer
}

func stripGoPath(file string) (string, error) {
	return filepath.Rel(filepath.Join(os.Getenv("GOPATH"), "src"), file)
}
