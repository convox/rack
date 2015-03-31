package logger

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"regexp"
	"runtime"
	"time"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

type Logger struct {
	namespace string
	now       func() string
	started   time.Time
	writer    io.Writer
}

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

func (l *Logger) Error(err error) {
	id := rand.Int31()

	l.Log("state=error id=%d message=%q", id, err)

	stack := make([]byte, 102400)
	runtime.Stack(stack, false)

	scanner := bufio.NewScanner(bytes.NewReader(stack))
	line := 1

	for scanner.Scan() {
		l.Log("state=error id=%d line=%d trace=%q", id, line, scanner.Text())
		line += 1
	}
}

func (l *Logger) Log(format string, args ...interface{}) {
	if l.started.IsZero() {
		l.writer.Write([]byte(fmt.Sprintf("%s %s\n", l.namespace, fmt.Sprintf(format, args...))))
	} else {
		elapsed := float64(time.Now().Sub(l.started).Nanoseconds()) / 1000000
		l.writer.Write([]byte(fmt.Sprintf("%s %s elapsed=%0.3f\n", l.namespace, fmt.Sprintf(format, args...), elapsed)))
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

func (l *Logger) Success(format string, args ...interface{}) {
	l.Log("state=success %s", fmt.Sprintf(format, args...))
}
